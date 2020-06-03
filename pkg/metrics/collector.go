package metrics

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/shurcooL/githubv4"

	"go.xrstf.de/github_exporter/pkg/client"
	"go.xrstf.de/github_exporter/pkg/fetcher"
	"go.xrstf.de/github_exporter/pkg/github"
)

var (
	pullRequestInfo = prometheus.NewDesc(
		"github_exporter_pr_info",
		"Various Pull Request related meta information with the static value 1",
		[]string{"repo", "number", "state", "mergable", "approved", "lgtm", "size"},
		nil,
	)

	pullRequestCount = prometheus.NewDesc(
		"github_exporter_pr_count",
		"Total count of Pull Requests",
		[]string{"repo", "state"},
		nil,
	)

	pullRequestMergableCount = prometheus.NewDesc(
		"github_exporter_pr_mergable_count",
		"Total count of mergable Pull Requests, according to the Prow Tide status",
		[]string{"repo"},
		nil,
	)

	pullRequestLabelCount = prometheus.NewDesc(
		"github_exporter_label_pr_count",
		"Total count of Pull Requests using a given label",
		[]string{"repo", "label", "state"},
		nil,
	)

	pullRequestCreatedAt = prometheus.NewDesc(
		"github_exporter_pr_created_at",
		"UNIX timestamp of a Pull Request's creation time",
		[]string{"repo", "number"},
		nil,
	)

	pullRequestUpdatedAt = prometheus.NewDesc(
		"github_exporter_pr_updated_at",
		"UNIX timestamp of a Pull Request's last update time",
		[]string{"repo", "number"},
		nil,
	)

	githubPointsRemaining = prometheus.NewDesc(
		"github_exporter_api_points_remaining",
		"Number of currently remaining GitHub API points",
		nil,
		nil,
	)

	githubRequestsTotal = prometheus.NewDesc(
		"github_exporter_api_requests_total",
		"Total number of requests against the GitHub API",
		[]string{"repo"},
		nil,
	)

	pullRequestQueueSize = prometheus.NewDesc(
		"github_exporter_pr_queue_size",
		"Number of pull requests currently queued for an update",
		[]string{"repo", "queue"},
		nil,
	)

	issueInfo = prometheus.NewDesc(
		"github_exporter_issue_info",
		"Various issue related meta information with the static value 1",
		[]string{"repo", "number", "state"},
		nil,
	)

	issueQueueSize = prometheus.NewDesc(
		"github_exporter_issue_queue_size",
		"Number of issues currently queued for an update",
		[]string{"repo", "queue"},
		nil,
	)
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

		ch <- prometheus.MustNewConstMetric(
			pullRequestQueueSize,
			prometheus.GaugeValue,
			float64(mc.fetcher.PriorityPullRequestQueueSize(repo)),
			fullName,
			"priority",
		)

		ch <- prometheus.MustNewConstMetric(
			pullRequestQueueSize,
			prometheus.GaugeValue,
			float64(mc.fetcher.RegularPullRequestQueueSize(repo)),
			fullName,
			"regular",
		)

		ch <- prometheus.MustNewConstMetric(
			issueQueueSize,
			prometheus.GaugeValue,
			float64(mc.fetcher.PriorityIssueQueueSize(repo)),
			fullName,
			"priority",
		)

		ch <- prometheus.MustNewConstMetric(
			issueQueueSize,
			prometheus.GaugeValue,
			float64(mc.fetcher.RegularIssueQueueSize(repo)),
			fullName,
			"regular",
		)

		ch <- prometheus.MustNewConstMetric(
			githubRequestsTotal,
			prometheus.CounterValue,
			float64(requestCounts[fullName]),
			fullName,
		)
	}

	ch <- prometheus.MustNewConstMetric(
		githubPointsRemaining,
		prometheus.GaugeValue,
		float64(mc.client.GetRemainingPoints()),
	)
}

func (mc *Collector) collectRepository(ch chan<- prometheus.Metric, repo *github.Repository) error {
	totalsByStateAndLabel := map[githubv4.PullRequestState]map[string]int{}
	totalMergable := 0 // with respect to tide
	totalsByState := map[githubv4.PullRequestState]int{}
	repository := fmt.Sprintf("%s/%s", repo.Owner, repo.Name)

	for number, pr := range repo.PullRequests {
		if _, ok := totalsByState[pr.State]; !ok {
			totalsByState[pr.State] = 0
			totalsByStateAndLabel[pr.State] = map[string]int{}
		}
		totalsByState[pr.State]++

		if pr.Mergable() {
			totalMergable++
		}

		mergable := fmt.Sprintf("%v", pr.Mergable())
		lgtm := fmt.Sprintf("%v", pr.HasLabel("lgtm"))
		approved := fmt.Sprintf("%v", pr.HasLabel("approved"))
		size := pr.Size()

		ch <- prometheus.MustNewConstMetric(
			pullRequestInfo,
			prometheus.GaugeValue,
			1,
			repository,
			strconv.Itoa(number),
			strings.ToLower(string(pr.State)),
			mergable,
			approved,
			lgtm,
			size,
		)

		ch <- prometheus.MustNewConstMetric(
			pullRequestCreatedAt,
			prometheus.GaugeValue,
			float64(pr.CreatedAt.Unix()),
			repository,
			strconv.Itoa(number),
		)

		ch <- prometheus.MustNewConstMetric(
			pullRequestUpdatedAt,
			prometheus.GaugeValue,
			float64(pr.UpdatedAt.Unix()),
			repository,
			strconv.Itoa(number),
		)

		for _, label := range pr.Labels {
			if _, ok := totalsByStateAndLabel[pr.State][label]; !ok {
				totalsByStateAndLabel[pr.State][label] = 0
			}

			totalsByStateAndLabel[pr.State][label]++
		}
	}

	for state, counts := range totalsByStateAndLabel {
		for label, count := range counts {
			ch <- prometheus.MustNewConstMetric(
				pullRequestLabelCount,
				prometheus.GaugeValue,
				float64(count),
				repository,
				label,
				strings.ToLower(string(state)),
			)
		}
	}

	for state, count := range totalsByState {
		ch <- prometheus.MustNewConstMetric(
			pullRequestCount,
			prometheus.GaugeValue,
			float64(count),
			repository,
			strings.ToLower(string(state)),
		)
	}

	ch <- prometheus.MustNewConstMetric(
		pullRequestMergableCount,
		prometheus.GaugeValue,
		float64(totalMergable),
		repository,
	)

	for number, issue := range repo.Issues {
		ch <- prometheus.MustNewConstMetric(
			issueInfo,
			prometheus.GaugeValue,
			1,
			repository,
			strconv.Itoa(number),
			strings.ToLower(string(issue.State)),
		)
	}

	return nil
}
