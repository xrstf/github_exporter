package fetcher

import (
	"time"

	"github.com/sirupsen/logrus"

	"go.xrstf.de/github_exporter/pkg/github"
)

const (
	scanMilestonesJobKey        = "scan-milestones"
	updateMilestonesJobKey      = "update-milestones"
	findUpdatedMilestonesJobKey = "find-updated-milestones"
)

type updateMilestonesJobMeta struct {
	numbers []int
}

// processUpdateMilestonesJob updates a list of already fetched
// milestones to ensure they stay up to date. This is done for all open
// milestones.
func (f *Fetcher) processUpdateMilestonesJob(repo *github.Repository, log logrus.FieldLogger, job string, data interface{}) error {
	meta := data.(updateMilestonesJobMeta)

	milestones, err := f.client.GetRepositoryMilestones(repo.Owner, repo.Name, meta.numbers)

	fetchedNumbers := []int{}
	fetchedNumbersMap := map[int]struct{}{}
	for _, milestone := range milestones {
		fetchedNumbers = append(fetchedNumbers, milestone.Number)
		fetchedNumbersMap[milestone.Number] = struct{}{}
	}

	log.Debugf("Fetched %d out of %d milestones.", len(fetchedNumbers), len(meta.numbers))

	deleted := []int{}
	for _, number := range meta.numbers {
		if _, ok := fetchedNumbersMap[number]; !ok {
			deleted = append(deleted, number)
		}
	}

	if len(milestones) > 0 {
		repo.AddMilestones(milestones)
	}

	// only delete not found milestones from our local cache if the request was a success, otherwise
	// we would remove all milestones if e.g. GitHub is unavailable
	if err == nil && len(deleted) > 0 {
		repo.DeleteMilestones(deleted)
	}

	f.removeJob(repo, job)
	f.dequeueMilestones(repo, meta.numbers)

	return err
}

// processFindUpdatedMilestonesJob fetches the 100 most recently updated
// milestones in the given repository and updates repo. The job will be removed
// from the job queue afterwards and all fetched milestones will be removed from
// the priority/regular milestone queues.
func (f *Fetcher) processFindUpdatedMilestonesJob(repo *github.Repository, log logrus.FieldLogger, job string) error {
	fetchedNumbers := []int{}

	milestones, _, err := f.client.ListMilestones(repo.Owner, repo.Name, nil, "")
	for _, milestone := range milestones {
		fetchedNumbers = append(fetchedNumbers, milestone.Number)
	}

	log.Debugf("Fetched %d recently updated milestones.", len(fetchedNumbers))

	repo.AddMilestones(milestones)

	f.removeJob(repo, job)
	f.dequeueMilestones(repo, fetchedNumbers)

	return err
}

type scanMilestonesJobMeta struct {
	max     int
	fetched int
	cursor  string
}

// processScanMilestonesJob is the initial job for every repository.
// It lists all existing milestones and adds them to repo.
//
// Because the initial scan is vital for proper functioning of every
// other job, this job must succeed before anything else can happen
// with a repository. For this reason a failed scan job is re-queued
// a few seconds later.
func (f *Fetcher) processScanMilestonesJob(repo *github.Repository, log logrus.FieldLogger, job string, data interface{}) error {
	meta := data.(scanMilestonesJobMeta)
	fullName := repo.FullName()
	fetchedNumbers := []int{}

	milestones, cursor, err := f.client.ListMilestones(repo.Owner, repo.Name, nil, meta.cursor)

	// if a max limit was set, enforce it (using ">=" here makes
	// it so that we stop cleanly when the list of milestones is exactly
	// the right amount that was left to fetch)
	if meta.max > 0 && len(milestones)+meta.fetched >= meta.max {
		milestones = milestones[:meta.max-meta.fetched]
		cursor = ""
	}

	for _, milestone := range milestones {
		fetchedNumbers = append(fetchedNumbers, milestone.Number)
	}

	repo.AddMilestones(milestones)
	f.dequeueMilestones(repo, fetchedNumbers)

	// always delete the job, no matter the outcome
	f.lock.Lock()
	delete(f.jobQueues[fullName], job)
	f.lock.Unlock()

	// batch query was successful
	if err == nil {
		log.WithField("new-cursor", cursor).Debugf("Fetched %d milestones.", len(milestones))

		// queue the query for the next page
		if cursor != "" {
			f.enqueueJob(repo, job, scanMilestonesJobMeta{
				max:     meta.max,
				fetched: meta.fetched + len(milestones),
				cursor:  cursor,
			})
		}

		return nil
	}

	retryAfter := 30 * time.Second
	log.Errorf("Failed to list milestones, will retry in %s: %v", retryAfter.String(), err)

	// query failed, re-try later
	go func() {
		time.Sleep(retryAfter)
		f.enqueueJob(repo, job, data)
	}()

	return err
}
