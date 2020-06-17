package metrics

import (
	"strconv"
	"strings"

	"go.xrstf.de/github_exporter/pkg/prow"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/shurcooL/githubv4"

	"go.xrstf.de/github_exporter/pkg/client"
	"go.xrstf.de/github_exporter/pkg/fetcher"
	"go.xrstf.de/github_exporter/pkg/github"
)

var (
	AllPullRequestStates = []string{
		string(githubv4.PullRequestStateOpen),
		string(githubv4.PullRequestStateClosed),
		string(githubv4.PullRequestStateMerged),
	}

	AllIssueStates = []string{
		string(githubv4.IssueStateOpen),
		string(githubv4.IssueStateClosed),
	}

	AllMilestoneStates = []string{
		string(githubv4.MilestoneStateOpen),
		string(githubv4.MilestoneStateClosed),
	}
)

type Collector struct {
	repos   map[string]*github.Repository
	fetcher *fetcher.Fetcher
	client  *client.Client
}

func NewCollector(repos map[string]*github.Repository, fetcher *fetcher.Fetcher, client *client.Client) *Collector {
	return &Collector{
		repos:   repos,
		fetcher: fetcher,
		client:  client,
	}
}

func (mc *Collector) Describe(ch chan<- *prometheus.Desc) {
	prometheus.DescribeByCollect(mc, ch)
}

func (mc *Collector) Collect(ch chan<- prometheus.Metric) {
	requestCounts := mc.client.GetRequestCounts()
	costs := mc.client.GetTotalCosts()

	for _, repo := range mc.repos {
		// do not publish metrics for repos for which we have not even fetched
		// the bare minimum of information
		if repo.FetchedAt == nil {
			continue
		}

		fullName := repo.FullName()

		_ = repo.RLocked(func(r *github.Repository) error {
			return mc.collectRepository(ch, r)
		})

		ch <- constMetric(githubRequestsTotal, prometheus.CounterValue, float64(requestCounts[fullName]), fullName)
		ch <- constMetric(githubCostsTotal, prometheus.CounterValue, float64(costs[fullName]), fullName)
	}

	ch <- constMetric(githubPointsRemaining, prometheus.GaugeValue, float64(mc.client.GetRemainingPoints()))
}

func (mc *Collector) collectRepository(ch chan<- prometheus.Metric, repo *github.Repository) error {
	if err := mc.collectRepoInfo(ch, repo); err != nil {
		return err
	}

	if err := mc.collectRepoPullRequests(ch, repo); err != nil {
		return err
	}

	if err := mc.collectRepoIssues(ch, repo); err != nil {
		return err
	}

	if err := mc.collectRepoMilestones(ch, repo); err != nil {
		return err
	}

	return nil
}

func boolVal(b bool) float64 {
	if b {
		return 1
	}

	return 0
}

func (mc *Collector) collectRepoInfo(ch chan<- prometheus.Metric, repo *github.Repository) error {
	repoName := repo.FullName()

	ch <- constMetric(repositoryDiskUsage, prometheus.GaugeValue, float64(repo.DiskUsageBytes), repoName)
	ch <- constMetric(repositoryForks, prometheus.GaugeValue, float64(repo.Forks), repoName)
	ch <- constMetric(repositoryStargazers, prometheus.GaugeValue, float64(repo.Stargazers), repoName)
	ch <- constMetric(repositoryWatchers, prometheus.GaugeValue, float64(repo.Watchers), repoName)
	ch <- constMetric(repositoryPrivate, prometheus.GaugeValue, boolVal(repo.IsPrivate), repoName)
	ch <- constMetric(repositoryArchived, prometheus.GaugeValue, boolVal(repo.IsArchived), repoName)
	ch <- constMetric(repositoryDisabled, prometheus.GaugeValue, boolVal(repo.IsDisabled), repoName)
	ch <- constMetric(repositoryFork, prometheus.GaugeValue, boolVal(repo.IsFork), repoName)
	ch <- constMetric(repositoryLocked, prometheus.GaugeValue, boolVal(repo.IsLocked), repoName)
	ch <- constMetric(repositoryMirror, prometheus.GaugeValue, boolVal(repo.IsMirror), repoName)
	ch <- constMetric(repositoryTemplate, prometheus.GaugeValue, boolVal(repo.IsTemplate), repoName)

	for language, size := range repo.Languages {
		ch <- constMetric(repositoryLanguageSize, prometheus.GaugeValue, float64(size), repoName, language)
	}

	return nil
}

func (mc *Collector) collectRepoPullRequests(ch chan<- prometheus.Metric, repo *github.Repository) error {
	totals := newStateLabelMap(repo, AllPullRequestStates)
	repoName := repo.FullName()

	for number, pr := range repo.PullRequests {
		num := strconv.Itoa(number)

		for _, label := range pr.Labels {
			totals[string(pr.State)][label]++
		}

		infoLabels := []string{
			repoName,
			num,
			pr.Author,
			strings.ToLower(string(pr.State)),
		}
		infoLabels = append(infoLabels, prow.PullRequestLabels(&pr)...)

		ch <- constMetric(pullRequestInfo, prometheus.GaugeValue, 1, infoLabels...)
		ch <- constMetric(pullRequestCreatedAt, prometheus.GaugeValue, float64(pr.CreatedAt.Unix()), repoName, num)
		ch <- constMetric(pullRequestUpdatedAt, prometheus.GaugeValue, float64(pr.UpdatedAt.Unix()), repoName, num)
		ch <- constMetric(pullRequestFetchedAt, prometheus.GaugeValue, float64(pr.FetchedAt.Unix()), repoName, num)
	}

	totals.ToMetrics(ch, repo, pullRequestLabelCount)

	ch <- constMetric(pullRequestQueueSize, prometheus.GaugeValue, float64(mc.fetcher.PriorityPullRequestQueueSize(repo)), repoName, "priority")
	ch <- constMetric(pullRequestQueueSize, prometheus.GaugeValue, float64(mc.fetcher.RegularPullRequestQueueSize(repo)), repoName, "regular")

	return nil
}

func (mc *Collector) collectRepoIssues(ch chan<- prometheus.Metric, repo *github.Repository) error {
	totals := newStateLabelMap(repo, AllIssueStates)
	repoName := repo.FullName()

	for number, issue := range repo.Issues {
		num := strconv.Itoa(number)

		for _, label := range issue.Labels {
			totals[string(issue.State)][label]++
		}

		infoLabels := []string{
			repoName,
			num,
			issue.Author,
			strings.ToLower(string(issue.State)),
		}
		infoLabels = append(infoLabels, prow.IssueLabels(&issue)...)

		ch <- constMetric(issueInfo, prometheus.GaugeValue, 1, infoLabels...)
		ch <- constMetric(issueCreatedAt, prometheus.GaugeValue, float64(issue.CreatedAt.Unix()), repoName, num)
		ch <- constMetric(issueUpdatedAt, prometheus.GaugeValue, float64(issue.UpdatedAt.Unix()), repoName, num)
		ch <- constMetric(issueFetchedAt, prometheus.GaugeValue, float64(issue.FetchedAt.Unix()), repoName, num)
	}

	totals.ToMetrics(ch, repo, issueLabelCount)

	ch <- constMetric(issueQueueSize, prometheus.GaugeValue, float64(mc.fetcher.PriorityIssueQueueSize(repo)), repoName, "priority")
	ch <- constMetric(issueQueueSize, prometheus.GaugeValue, float64(mc.fetcher.RegularIssueQueueSize(repo)), repoName, "regular")

	return nil
}

func (mc *Collector) collectRepoMilestones(ch chan<- prometheus.Metric, repo *github.Repository) error {
	repoName := repo.FullName()
	openState := strings.ToLower(string(githubv4.MilestoneStateOpen))
	closedState := strings.ToLower(string(githubv4.MilestoneStateClosed))

	for number, milestone := range repo.Milestones {
		num := strconv.Itoa(number)

		var closedAt int64
		if milestone.ClosedAt != nil {
			closedAt = milestone.ClosedAt.Unix()
		}

		var dueOn int64
		if milestone.DueOn != nil {
			dueOn = milestone.DueOn.Unix()
		}

		ch <- constMetric(milestoneInfo, prometheus.GaugeValue, 1, repoName, num, strings.ToLower(string(milestone.State)), milestone.Title)
		ch <- constMetric(milestoneCreatedAt, prometheus.GaugeValue, float64(milestone.CreatedAt.Unix()), repoName, num)
		ch <- constMetric(milestoneUpdatedAt, prometheus.GaugeValue, float64(milestone.UpdatedAt.Unix()), repoName, num)
		ch <- constMetric(milestoneClosedAt, prometheus.GaugeValue, float64(closedAt), repoName, num)
		ch <- constMetric(milestoneDueOn, prometheus.GaugeValue, float64(dueOn), repoName, num)
		ch <- constMetric(milestoneFetchedAt, prometheus.GaugeValue, float64(milestone.FetchedAt.Unix()), repoName, num)
		ch <- constMetric(milestoneIssues, prometheus.GaugeValue, float64(milestone.OpenIssues), repoName, num, "issue", openState)
		ch <- constMetric(milestoneIssues, prometheus.GaugeValue, float64(milestone.ClosedIssues), repoName, num, "issue", closedState)
		ch <- constMetric(milestoneIssues, prometheus.GaugeValue, float64(milestone.OpenPullRequests), repoName, num, "pullrequest", openState)
		ch <- constMetric(milestoneIssues, prometheus.GaugeValue, float64(milestone.ClosedPullRequests), repoName, num, "pullrequest", closedState)
	}

	ch <- constMetric(milestoneQueueSize, prometheus.GaugeValue, float64(mc.fetcher.PriorityMilestoneQueueSize(repo)), repoName, "priority")
	ch <- constMetric(milestoneQueueSize, prometheus.GaugeValue, float64(mc.fetcher.RegularMilestoneQueueSize(repo)), repoName, "regular")

	return nil
}

// constMetric just helps reducing code noise
func constMetric(desc *prometheus.Desc, valueType prometheus.ValueType, value float64, labelValues ...string) prometheus.Metric {
	return prometheus.MustNewConstMetric(desc, valueType, value, labelValues...)
}

type stateLabelMap map[string]map[string]int

func newStateLabelMap(repo *github.Repository, states []string) stateLabelMap {
	result := stateLabelMap{}

	for _, state := range states {
		result[state] = map[string]int{}

		for _, label := range repo.Labels {
			result[state][label] = 0
		}
	}

	return result
}

func (m stateLabelMap) ToMetrics(ch chan<- prometheus.Metric, repo *github.Repository, metric *prometheus.Desc) {
	repoName := repo.FullName()

	for state, counts := range m {
		for label, count := range counts {
			ch <- prometheus.MustNewConstMetric(metric, prometheus.GaugeValue, float64(count), repoName, label, strings.ToLower(state))
		}
	}
}
