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

	for _, repo := range mc.repos {
		fullName := repo.FullName()

		repo.RLocked(func(r *github.Repository) error {
			return mc.collectRepository(ch, r)
		})

		ch <- prometheus.MustNewConstMetric(githubRequestsTotal, prometheus.CounterValue, float64(requestCounts[fullName]), fullName)
	}

	ch <- prometheus.MustNewConstMetric(githubPointsRemaining, prometheus.GaugeValue, float64(mc.client.GetRemainingPoints()))
}

func (mc *Collector) collectRepository(ch chan<- prometheus.Metric, repo *github.Repository) error {
	if err := mc.collectRepoPullRequests(ch, repo); err != nil {
		return err
	}

	if err := mc.collectRepoIssues(ch, repo); err != nil {
		return err
	}

	return nil
}

func (mc *Collector) collectRepoPullRequests(ch chan<- prometheus.Metric, repo *github.Repository) error {
	totals := newStateLabelMap(repo, AllPullRequestStates)
	repoName := repo.FullName()

	for number, pr := range repo.PullRequests {
		for _, label := range pr.Labels {
			totals[string(pr.State)][label]++
		}

		infoLabels := []string{
			repoName,
			strconv.Itoa(number),
			pr.Author,
			strings.ToLower(string(pr.State)),
		}
		infoLabels = append(infoLabels, prow.PullRequestLabels(&pr)...)

		ch <- prometheus.MustNewConstMetric(pullRequestInfo, prometheus.GaugeValue, 1, infoLabels...)
		ch <- prometheus.MustNewConstMetric(pullRequestCreatedAt, prometheus.GaugeValue, float64(pr.CreatedAt.Unix()), repoName, strconv.Itoa(number))
		ch <- prometheus.MustNewConstMetric(pullRequestUpdatedAt, prometheus.GaugeValue, float64(pr.UpdatedAt.Unix()), repoName, strconv.Itoa(number))
		ch <- prometheus.MustNewConstMetric(pullRequestFetchedAt, prometheus.GaugeValue, float64(pr.FetchedAt.Unix()), repoName, strconv.Itoa(number))
	}

	totals.ToMetrics(ch, repo, pullRequestLabelCount)

	ch <- prometheus.MustNewConstMetric(pullRequestQueueSize, prometheus.GaugeValue, float64(mc.fetcher.PriorityPullRequestQueueSize(repo)), repoName, "priority")
	ch <- prometheus.MustNewConstMetric(pullRequestQueueSize, prometheus.GaugeValue, float64(mc.fetcher.RegularPullRequestQueueSize(repo)), repoName, "regular")

	return nil
}

func (mc *Collector) collectRepoIssues(ch chan<- prometheus.Metric, repo *github.Repository) error {
	totals := newStateLabelMap(repo, AllIssueStates)
	repoName := repo.FullName()

	for number, issue := range repo.Issues {
		for _, label := range issue.Labels {
			totals[string(issue.State)][label]++
		}

		infoLabels := []string{
			repoName,
			strconv.Itoa(number),
			issue.Author,
			strings.ToLower(string(issue.State)),
		}
		infoLabels = append(infoLabels, prow.IssueLabels(&issue)...)

		ch <- prometheus.MustNewConstMetric(issueInfo, prometheus.GaugeValue, 1, infoLabels...)
		ch <- prometheus.MustNewConstMetric(issueCreatedAt, prometheus.GaugeValue, float64(issue.CreatedAt.Unix()), repoName, strconv.Itoa(number))
		ch <- prometheus.MustNewConstMetric(issueUpdatedAt, prometheus.GaugeValue, float64(issue.UpdatedAt.Unix()), repoName, strconv.Itoa(number))
		ch <- prometheus.MustNewConstMetric(issueFetchedAt, prometheus.GaugeValue, float64(issue.FetchedAt.Unix()), repoName, strconv.Itoa(number))
	}

	totals.ToMetrics(ch, repo, issueLabelCount)

	ch <- prometheus.MustNewConstMetric(issueQueueSize, prometheus.GaugeValue, float64(mc.fetcher.PriorityIssueQueueSize(repo)), repoName, "priority")
	ch <- prometheus.MustNewConstMetric(issueQueueSize, prometheus.GaugeValue, float64(mc.fetcher.RegularIssueQueueSize(repo)), repoName, "regular")

	return nil
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
