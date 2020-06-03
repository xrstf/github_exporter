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
	repositories    repositoryList
	ignoredContexts stringSlice
	refreshInterval time.Duration
	resyncInterval  time.Duration
	depth           int
	refreshDepth    int
	listenAddr      string
	debugLog        bool
}

const (
	GithubClientCtxKey = "github-client"
	OptionsCtxKey      = "options"
)

type AppContext struct {
	ctx     context.Context
	client  *client.Client
	fetcher *fetcher.Fetcher
	options *options
}

func main() {
	opt := options{
		refreshInterval: 5 * time.Minute,
		resyncInterval:  12 * time.Hour,
		depth:           -1,
		refreshDepth:    50,
		listenAddr:      ":8080",
	}

	flag.Var(&opt.repositories, "repo", "repository (owner/name format) to include, can be given multiple times")
	flag.Var(&opt.ignoredContexts, "ignore-context", "build context to ignore when determining PR mergability, can be given multiple times")
	flag.IntVar(&opt.depth, "depth", opt.depth, "max number of pull request to fetch in any given repository (PRs are always sorted in descending order by updated time) (-1 disables the limit)")
	flag.DurationVar(&opt.refreshInterval, "refresh-interval", opt.refreshInterval, "time in between refreshes (only open PRs and up to N recently updated PRs)")
	flag.IntVar(&opt.refreshDepth, "refresh-depth", opt.refreshDepth, "number of recently updated PRs to fetch in every refresh")
	flag.DurationVar(&opt.resyncInterval, "resync-interval", opt.resyncInterval, "time in between full re-syncs (fetching all labels and pull requests)")
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

	token := os.Getenv("GITHUB_TOKEN")
	if len(token) == 0 {
		log.Fatal("No GITHUB_TOKEN environment variable defined.")
	}

	// setup API client
	ctx := context.Background()

	client, err := client.NewClient(ctx, log.WithField("component", "client"), token)
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

	for identifier, repo := range repositories {
		repoLog := log.WithField("repo", identifier)

		repoLog.Info("Scheduling initial scans…")
		ctx.fetcher.EnqueueLabelUpdate(repo)
		ctx.fetcher.EnqueuePullRequestScan(repo)
		ctx.fetcher.EnqueueIssueScan(repo)

		// keep refreshing open PRs
		go refreshWorker(ctx, repoLog, repo)

		// in a much larger interval, crawl all existing PRs to detect deletions and changes
		// after a PR has been merged
		go resyncWorker(ctx, repoLog, repo)
	}
}

// refreshRepositoriesWorker refreshes all OPEN pull requests, because changes
// to the build contexts do not change the updatedAt timestamp on GitHub and we
// want to closely track the mergability. It also fetches the last 50 updated
// PRs to find cases where a PR was merged and is not open anymore.
func refreshWorker(ctx AppContext, log logrus.FieldLogger, repo *github.Repository) {
	for _ = range time.NewTicker(ctx.options.refreshInterval).C {
		log.Debug("Refreshing open pull requests…")

		numbers := []int{}
		for _, pr := range repo.GetPullRequests(githubv4.PullRequestStateOpen) {
			numbers = append(numbers, pr.Number)
		}

		ctx.fetcher.EnqueuePriorityPullRequests(repo, numbers)
		ctx.fetcher.EnqueueUpdatedPullRequests(repo)

		log.Debug("Refreshing open pull issues…")

		numbers = []int{}
		for _, issue := range repo.GetIssues(githubv4.IssueStateOpen) {
			numbers = append(numbers, issue.Number)
		}

		ctx.fetcher.EnqueuePriorityIssues(repo, numbers)
		ctx.fetcher.EnqueueUpdatedIssues(repo)
	}
}

func resyncWorker(ctx AppContext, log logrus.FieldLogger, repo *github.Repository) {
	for _ = range time.NewTicker(ctx.options.resyncInterval).C {
		log.Info("Synchronizing repository…")

		numbers := []int{}
		for _, pr := range repo.GetPullRequests(githubv4.PullRequestStateClosed, githubv4.PullRequestStateMerged) {
			numbers = append(numbers, pr.Number)
		}

		ctx.fetcher.EnqueueRegularPullRequests(repo, numbers)
		ctx.fetcher.EnqueueLabelUpdate(repo)

		numbers = []int{}
		for _, issue := range repo.GetIssues(githubv4.IssueStateClosed) {
			numbers = append(numbers, issue.Number)
		}

		ctx.fetcher.EnqueueRegularIssues(repo, numbers)
	}
}
