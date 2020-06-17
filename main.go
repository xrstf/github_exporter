package main

import (
	"context"
	"flag"
	"net/http"
	"os"
	"time"

	"go.xrstf.de/github_exporter/pkg/client"
	"go.xrstf.de/github_exporter/pkg/fetcher"
	"go.xrstf.de/github_exporter/pkg/github"
	"go.xrstf.de/github_exporter/pkg/metrics"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/shurcooL/githubv4"
	"github.com/sirupsen/logrus"
)

type options struct {
	repositories             repositoryList
	realnames                bool
	repoRefreshInterval      time.Duration
	prRefreshInterval        time.Duration
	prResyncInterval         time.Duration
	prDepth                  int
	issueRefreshInterval     time.Duration
	issueResyncInterval      time.Duration
	issueDepth               int
	milestoneRefreshInterval time.Duration
	milestoneResyncInterval  time.Duration
	milestoneDepth           int
	listenAddr               string
	debugLog                 bool
}

type AppContext struct {
	ctx     context.Context
	client  *client.Client
	fetcher *fetcher.Fetcher
	options *options
}

func main() {
	opt := options{
		repoRefreshInterval:      5 * time.Minute,
		prRefreshInterval:        5 * time.Minute,
		prResyncInterval:         12 * time.Hour,
		prDepth:                  -1,
		issueRefreshInterval:     5 * time.Minute,
		issueResyncInterval:      12 * time.Hour,
		issueDepth:               -1,
		milestoneRefreshInterval: 5 * time.Minute,
		milestoneResyncInterval:  12 * time.Hour,
		milestoneDepth:           -1,
		listenAddr:               ":9612",
	}

	flag.Var(&opt.repositories, "repo", "repository (owner/name format) to include, can be given multiple times")
	flag.BoolVar(&opt.realnames, "realnames", opt.realnames, "use usernames instead of internal IDs for author labels (this will make metrics contain personally identifiable information)")
	flag.DurationVar(&opt.repoRefreshInterval, "repo-refresh-interval", opt.repoRefreshInterval, "time in between repository metadata refreshes")
	flag.IntVar(&opt.prDepth, "pr-depth", opt.prDepth, "max number of pull requests to fetch per repository upon startup (-1 disables the limit, 0 disables PR fetching entirely)")
	flag.DurationVar(&opt.prRefreshInterval, "pr-refresh-interval", opt.prRefreshInterval, "time in between PR refreshes")
	flag.DurationVar(&opt.prResyncInterval, "pr-resync-interval", opt.prResyncInterval, "time in between full PR re-syncs")
	flag.IntVar(&opt.issueDepth, "issue-depth", opt.issueDepth, "max number of issues to fetch per repository upon startup (-1 disables the limit, 0 disables issue fetching entirely)")
	flag.DurationVar(&opt.issueRefreshInterval, "issue-refresh-interval", opt.issueRefreshInterval, "time in between issue refreshes")
	flag.DurationVar(&opt.issueResyncInterval, "issue-resync-interval", opt.issueResyncInterval, "time in between full issue re-syncs")
	flag.IntVar(&opt.milestoneDepth, "milestone-depth", opt.milestoneDepth, "max number of milestones to fetch per repository upon startup (-1 disables the limit, 0 disables milestone fetching entirely)")
	flag.DurationVar(&opt.milestoneRefreshInterval, "milestone-refresh-interval", opt.milestoneRefreshInterval, "time in between milestone refreshes")
	flag.DurationVar(&opt.milestoneResyncInterval, "milestone-resync-interval", opt.milestoneResyncInterval, "time in between full milestone re-syncs")
	flag.StringVar(&opt.listenAddr, "listen", opt.listenAddr, "address and port to listen on")
	flag.BoolVar(&opt.debugLog, "debug", opt.debugLog, "enable more verbose logging")
	flag.Parse()

	// setup logging
	var log = logrus.New()
	log.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: time.RFC1123,
	})

	if opt.debugLog {
		log.SetLevel(logrus.DebugLevel)
	}

	// validate CLI flags
	if len(opt.repositories) == 0 {
		log.Fatal("No -repo defined.")
	}

	if opt.prRefreshInterval >= opt.prResyncInterval {
		log.Fatal("-pr-refresh-interval must be < than -pr-resync-interval.")
	}

	if opt.issueRefreshInterval >= opt.issueResyncInterval {
		log.Fatal("-issue-refresh-interval must be < than -issue-resync-interval.")
	}

	if opt.milestoneRefreshInterval >= opt.milestoneResyncInterval {
		log.Fatal("-milestone-refresh-interval must be < than -milestone-resync-interval.")
	}

	token := os.Getenv("GITHUB_TOKEN")
	if len(token) == 0 {
		log.Fatal("No GITHUB_TOKEN environment variable defined.")
	}

	// setup API client
	ctx := context.Background()

	client, err := client.NewClient(ctx, log.WithField("component", "client"), token, opt.realnames)
	if err != nil {
		log.Fatalf("Failed to create API client: %v", err)
	}

	appCtx := AppContext{
		ctx:     ctx,
		client:  client,
		options: &opt,
	}

	// start fetching data in the background, but start metrics
	// server as soon as possible
	go setup(appCtx, log)

	log.Printf("Starting server on %s…", opt.listenAddr)

	http.Handle("/metrics", promhttp.Handler())
	log.Fatal(http.ListenAndServe(opt.listenAddr, nil))
}

func setup(ctx AppContext, log logrus.FieldLogger) {
	repositories := map[string]*github.Repository{}

	// create a PR database for each repo
	for _, repo := range ctx.options.repositories {
		repositories[repo.String()] = github.NewRepository(repo.owner, repo.name)
	}

	// setup the single-threaded fetcher
	ctx.fetcher = fetcher.NewFetcher(ctx.client, repositories, log.WithField("component", "fetcher"))
	go ctx.fetcher.Worker()

	prometheus.MustRegister(metrics.NewCollector(repositories, ctx.fetcher, ctx.client))

	// perform the initial scan sequentially across all repositories, otherwise
	// it's likely that we trigger GitHub's anti abuse system
	log.Info("Initializing repositories…")

	hasLabelledMetrics := ctx.options.prDepth != 0 || ctx.options.issueDepth != 0 || ctx.options.milestoneDepth != 0

	for identifier, repo := range repositories {
		repoLog := log.WithField("repo", identifier)

		repoLog.Info("Scheduling initial scans…")
		ctx.fetcher.EnqueueRepoUpdate(repo)

		// keep repository metadata up-to-date
		go refreshRepositoryInfoWorker(ctx, repoLog, repo)

		if hasLabelledMetrics {
			ctx.fetcher.EnqueueLabelUpdate(repo)
		}

		if ctx.options.prDepth != 0 {
			ctx.fetcher.EnqueuePullRequestScan(repo, ctx.options.prDepth)

			// keep refreshing open PRs
			go refreshPullRequestsWorker(ctx, repoLog, repo)

			// in a much larger interval, crawl all existing PRs to detect deletions and changes
			// after a PR has been merged
			go resyncPullRequestsWorker(ctx, repoLog, repo)
		}

		if ctx.options.issueDepth != 0 {
			ctx.fetcher.EnqueueIssueScan(repo, ctx.options.issueDepth)

			// keep refreshing open issues
			go refreshIssuesWorker(ctx, repoLog, repo)

			// in a much larger interval, crawl all existing issues to detect status changes
			go resyncIssuesWorker(ctx, repoLog, repo)
		}

		if ctx.options.milestoneDepth != 0 {
			ctx.fetcher.EnqueueMilestoneScan(repo, ctx.options.milestoneDepth)

			// keep refreshing open milestones
			go refreshMilestonesWorker(ctx, repoLog, repo)

			// in a much larger interval, crawl all existing milestones to detect status changes
			go resyncMilestonesWorker(ctx, repoLog, repo)
		}
	}
}

func refreshRepositoryInfoWorker(ctx AppContext, log logrus.FieldLogger, repo *github.Repository) {
	for range time.NewTicker(ctx.options.repoRefreshInterval).C {
		log.Debug("Refreshing repository metadata…")
		ctx.fetcher.EnqueueRepoUpdate(repo)
	}
}

// refreshRepositoriesWorker refreshes all OPEN pull requests, because changes
// to the build contexts do not change the updatedAt timestamp on GitHub and we
// want to closely track the mergability. It also fetches the last 50 updated
// PRs to find cases where a PR was merged and is not open anymore.
func refreshPullRequestsWorker(ctx AppContext, log logrus.FieldLogger, repo *github.Repository) {
	for range time.NewTicker(ctx.options.prRefreshInterval).C {
		log.Debug("Refreshing open pull requests…")

		numbers := []int{}
		for _, pr := range repo.GetPullRequests(githubv4.PullRequestStateOpen) {
			numbers = append(numbers, pr.Number)
		}

		ctx.fetcher.EnqueuePriorityPullRequests(repo, numbers)
		ctx.fetcher.EnqueueUpdatedPullRequests(repo)
	}
}

func resyncPullRequestsWorker(ctx AppContext, log logrus.FieldLogger, repo *github.Repository) {
	for range time.NewTicker(ctx.options.prResyncInterval).C {
		log.Info("Synchronizing repository pull requests…")

		numbers := []int{}
		for _, pr := range repo.GetPullRequests(githubv4.PullRequestStateClosed, githubv4.PullRequestStateMerged) {
			numbers = append(numbers, pr.Number)
		}

		ctx.fetcher.EnqueueRegularPullRequests(repo, numbers)
		ctx.fetcher.EnqueueLabelUpdate(repo)
	}
}

func refreshIssuesWorker(ctx AppContext, log logrus.FieldLogger, repo *github.Repository) {
	for range time.NewTicker(ctx.options.issueRefreshInterval).C {
		log.Debug("Refreshing open pull issues…")

		numbers := []int{}
		for _, issue := range repo.GetIssues(githubv4.IssueStateOpen) {
			numbers = append(numbers, issue.Number)
		}

		ctx.fetcher.EnqueuePriorityIssues(repo, numbers)
		ctx.fetcher.EnqueueUpdatedIssues(repo)
	}
}

func resyncIssuesWorker(ctx AppContext, log logrus.FieldLogger, repo *github.Repository) {
	for range time.NewTicker(ctx.options.issueResyncInterval).C {
		log.Info("Synchronizing repository issues…")

		numbers := []int{}
		for _, issue := range repo.GetIssues(githubv4.IssueStateClosed) {
			numbers = append(numbers, issue.Number)
		}

		ctx.fetcher.EnqueueRegularIssues(repo, numbers)
		ctx.fetcher.EnqueueLabelUpdate(repo)
	}
}

func refreshMilestonesWorker(ctx AppContext, log logrus.FieldLogger, repo *github.Repository) {
	for range time.NewTicker(ctx.options.milestoneRefreshInterval).C {
		log.Debug("Refreshing open pull milestones…")

		numbers := []int{}
		for _, milestone := range repo.GetMilestones(githubv4.MilestoneStateOpen) {
			numbers = append(numbers, milestone.Number)
		}

		ctx.fetcher.EnqueuePriorityMilestones(repo, numbers)
		ctx.fetcher.EnqueueUpdatedMilestones(repo)
	}
}

func resyncMilestonesWorker(ctx AppContext, log logrus.FieldLogger, repo *github.Repository) {
	for range time.NewTicker(ctx.options.milestoneResyncInterval).C {
		log.Info("Synchronizing repository milestones…")

		numbers := []int{}
		for _, milestone := range repo.GetMilestones(githubv4.MilestoneStateClosed) {
			numbers = append(numbers, milestone.Number)
		}

		ctx.fetcher.EnqueueRegularMilestones(repo, numbers)
		ctx.fetcher.EnqueueLabelUpdate(repo)
	}
}
