package fetcher

import (
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	"go.xrstf.de/github_exporter/pkg/client"
	"go.xrstf.de/github_exporter/pkg/github"
)

type Fetcher struct {
	client            *client.Client
	log               logrus.FieldLogger
	repositories      map[string]*github.Repository
	jobQueues         map[string]jobQueue
	pullRequestQueues map[string]prioritizedIntegerQueue
	issueQueues       map[string]prioritizedIntegerQueue
	lock              sync.RWMutex
}

func NewFetcher(client *client.Client, repos map[string]*github.Repository, log logrus.FieldLogger) *Fetcher {
	return &Fetcher{
		client:            client,
		log:               log,
		repositories:      repos,
		jobQueues:         makeJobQueues(repos),
		pullRequestQueues: makePrioritizedIntegerQueues(repos),
		issueQueues:       makePrioritizedIntegerQueues(repos),
		lock:              sync.RWMutex{},
	}
}

func makePrioritizedIntegerQueues(repos map[string]*github.Repository) map[string]prioritizedIntegerQueue {
	queues := map[string]prioritizedIntegerQueue{}

	for fullName := range repos {
		queues[fullName] = newPrioritizedIntegerQueue()
	}

	return queues
}

func makeJobQueues(repos map[string]*github.Repository) map[string]jobQueue {
	queues := map[string]jobQueue{}

	for fullName := range repos {
		queues[fullName] = jobQueue{}
	}

	return queues
}

func (f *Fetcher) EnqueueLabelUpdate(r *github.Repository) {
	f.enqueueJob(r, updateLabelsJobKey, nil)
}

func (f *Fetcher) EnqueueUpdatedPullRequests(r *github.Repository) {
	f.enqueueJob(r, findUpdatedPullRequestsJobKey, nil)
}

func (f *Fetcher) EnqueuePullRequestScan(r *github.Repository, max int) {
	f.enqueueJob(r, scanPullRequestsJobKey, scanPullRequestsJobMeta{
		max: max,
	})
}

func (f *Fetcher) enqueueUpdatedPullRequests(r *github.Repository, numbers []int) {
	f.enqueueJob(r, updatePullRequestsJobKey, updatePullRequestsJobMeta{
		numbers: numbers,
	})
}

func (f *Fetcher) EnqueueUpdatedIssues(r *github.Repository) {
	f.enqueueJob(r, findUpdatedIssuesJobKey, nil)
}

func (f *Fetcher) EnqueueIssueScan(r *github.Repository, max int) {
	f.enqueueJob(r, scanIssuesJobKey, scanIssuesJobMeta{
		max: max,
	})
}

func (f *Fetcher) enqueueUpdatedIssues(r *github.Repository, numbers []int) {
	f.enqueueJob(r, updateIssuesJobKey, updateIssuesJobMeta{
		numbers: numbers,
	})
}

func (f *Fetcher) enqueueJob(r *github.Repository, key string, data interface{}) {
	f.lock.Lock()
	defer f.lock.Unlock()

	f.log.WithField("repo", r.FullName()).WithField("job", key).Debug("Enqueueing job.")

	f.jobQueues[r.FullName()][key] = data
}

func (f *Fetcher) EnqueuePriorityPullRequests(r *github.Repository, numbers []int) {
	f.enqueue(r, numbers, f.pullRequestQueues, true)
}

func (f *Fetcher) EnqueueRegularPullRequests(r *github.Repository, numbers []int) {
	f.enqueue(r, numbers, f.pullRequestQueues, false)
}

func (f *Fetcher) EnqueuePriorityIssues(r *github.Repository, numbers []int) {
	f.enqueue(r, numbers, f.issueQueues, true)
}

func (f *Fetcher) EnqueueRegularIssues(r *github.Repository, numbers []int) {
	f.enqueue(r, numbers, f.issueQueues, false)
}

func (f *Fetcher) enqueue(r *github.Repository, numbers []int, queues map[string]prioritizedIntegerQueue, priority bool) {
	queue, ok := queues[r.FullName()]
	if !ok {
		f.log.Fatalf("No queue defined for repository %s", r.FullName())
	}

	f.lock.Lock()
	defer f.lock.Unlock()

	f.log.WithField("repo", r.FullName()).Debugf("Enqueueing %d items for updating.", len(numbers))

	if priority {
		queue.priorityEnqueue(numbers)
	} else {
		queue.regularEnqueue(numbers)
	}
}

func (f *Fetcher) PriorityPullRequestQueueSize(r *github.Repository) int {
	return f.queueSize(r, f.pullRequestQueues, true)
}

func (f *Fetcher) RegularPullRequestQueueSize(r *github.Repository) int {
	return f.queueSize(r, f.pullRequestQueues, false)
}

func (f *Fetcher) PriorityIssueQueueSize(r *github.Repository) int {
	return f.queueSize(r, f.issueQueues, true)
}

func (f *Fetcher) RegularIssueQueueSize(r *github.Repository) int {
	return f.queueSize(r, f.issueQueues, false)
}

func (f *Fetcher) queueSize(r *github.Repository, queues map[string]prioritizedIntegerQueue, priority bool) int {
	queue, ok := queues[r.FullName()]
	if !ok {
		f.log.Fatalf("No queue defined for repository %s", r.FullName())
	}

	f.lock.RLock()
	defer f.lock.RUnlock()

	if priority {
		return queue.prioritySize()
	} else {
		return queue.regularSize()
	}
}

func (f *Fetcher) Worker() {
	lastForceFlush := time.Now()
	maxWaitTime := 1 * time.Minute

	for {
		// the job queue has priority over crawling numbered PRs from the other queues
		repo, job, data := f.getNextJob()

		// there is a job ready to be processed
		if repo != nil {
			err := f.processJob(repo, job, data)
			if err != nil {
				f.log.Errorf("Failed to process job: %v", err)
			}

			continue
		}

		// if there was no job, try to create a job to update the existing
		// numbered PRs
		repo, candidates := f.getPullRequestBatch(10, client.MaxPullRequestsPerQuery)

		// a repository has amassed enough PRs to warrant a new job
		if repo != nil {
			f.enqueueUpdatedPullRequests(repo, candidates)
			continue
		}

		// try batching up issues next
		repo, candidates = f.getIssueBatch(10, client.MaxIssuesPerQuery)
		if repo != nil {
			f.enqueueUpdatedIssues(repo, candidates)
			continue
		}

		// no repo has enough items for a good batch; in order to not burn
		// CPU cycles, we will wait a bit and check again. But we don't wait
		// forever, otherwise repositories with very few PRs might never get
		// updated.
		if time.Since(lastForceFlush) < maxWaitTime {
			time.Sleep(1 * time.Second)
			continue
		}

		// we waited long enough, give up and accept 1-element batches
		repo, candidates = f.getPullRequestBatch(1, client.MaxPullRequestsPerQuery)

		// got a mini batch
		if repo != nil {
			f.enqueueUpdatedPullRequests(repo, candidates)
			continue
		}

		repo, candidates = f.getIssueBatch(1, client.MaxIssuesPerQuery)
		if repo != nil {
			f.enqueueUpdatedIssues(repo, candidates)
			continue
		}

		// all repository queues are entirely empty, we finished the
		// force flush and can remember the time; this means on the next
		// iteration we will begin to sleep again.
		lastForceFlush = time.Now()

		f.log.Debug("All queues emptied, force flush completed.")
	}
}

func (f *Fetcher) getNextJob() (*github.Repository, string, interface{}) {
	f.lock.RLock()
	defer f.lock.RUnlock()

	for fullName, queue := range f.jobQueues {
		// the scan jobs have priority over everything else, as other jobs are based on it

		if data, ok := queue[scanIssuesJobKey]; ok {
			return f.repositories[fullName], scanIssuesJobKey, data
		}

		if data, ok := queue[scanPullRequestsJobKey]; ok {
			return f.repositories[fullName], scanPullRequestsJobKey, data
		}

		for job, data := range queue {
			return f.repositories[fullName], job, data
		}
	}

	return nil, "", nil
}

func (f *Fetcher) getPullRequestBatch(minBatchSize int, maxBatchSize int) (*github.Repository, []int) {
	return f.getBatch(f.pullRequestQueues, minBatchSize, maxBatchSize)
}

func (f *Fetcher) getIssueBatch(minBatchSize int, maxBatchSize int) (*github.Repository, []int) {
	return f.getBatch(f.issueQueues, minBatchSize, maxBatchSize)
}

func (f *Fetcher) getBatch(queues map[string]prioritizedIntegerQueue, minBatchSize int, maxBatchSize int) (*github.Repository, []int) {
	f.lock.RLock()
	defer f.lock.RUnlock()

	for fullName := range f.repositories {
		queue := queues[fullName]

		batch := queue.getBatch(minBatchSize, maxBatchSize)
		if batch != nil {
			return f.repositories[fullName], batch
		}
	}

	// no repository has (combined) enough items to satisfy minBatchSize
	return nil, nil
}

func (f *Fetcher) processJob(repo *github.Repository, job string, data interface{}) error {
	var err error

	log := f.log.WithField("repo", repo.FullName()).WithField("job", job)
	log.Debug("Processing job…")

	switch job {
	case updateLabelsJobKey:
		err = f.processUpdateLabelsJob(repo, log, job)
	case updatePullRequestsJobKey:
		err = f.processUpdatePullRequestsJob(repo, log, job, data)
	case findUpdatedPullRequestsJobKey:
		err = f.processFindUpdatedPullRequestsJob(repo, log, job)
	case scanPullRequestsJobKey:
		err = f.processScanPullRequestsJob(repo, log, job, data)
	case updateIssuesJobKey:
		err = f.processUpdateIssuesJob(repo, log, job, data)
	case findUpdatedIssuesJobKey:
		err = f.processFindUpdatedIssuesJob(repo, log, job)
	case scanIssuesJobKey:
		err = f.processScanIssuesJob(repo, log, job, data)
	default:
		f.log.Fatalf("Encountered unknown job type %q for repo %q", job, repo.FullName())
	}

	return err
}

func (f *Fetcher) removeJob(repo *github.Repository, job string) {
	f.log.WithField("job", job).Debugf("Removing job.")

	fullName := repo.FullName()

	f.lock.Lock()
	defer f.lock.Unlock()

	delete(f.jobQueues[fullName], job)
}

func (f *Fetcher) dequeuePullRequests(repo *github.Repository, numbers []int) {
	f.log.Debugf("Removing %d fetched PRs.", len(numbers))
	f.dequeue(repo, f.pullRequestQueues, numbers)
}

func (f *Fetcher) dequeueIssues(repo *github.Repository, numbers []int) {
	f.log.Debugf("Removing %d fetched issues.", len(numbers))
	f.dequeue(repo, f.issueQueues, numbers)
}

func (f *Fetcher) dequeue(repo *github.Repository, queues map[string]prioritizedIntegerQueue, numbers []int) {
	fullName := repo.FullName()

	f.lock.Lock()
	defer f.lock.Unlock()

	queue := queues[fullName]
	queue.dequeue(numbers)
}
