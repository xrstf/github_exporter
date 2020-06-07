package metrics

import (
	"go.xrstf.de/github_exporter/pkg/prow"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	pullRequestInfo *prometheus.Desc

	pullRequestLabelCount = prometheus.NewDesc(
		"github_exporter_pr_label_count",
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

	pullRequestFetchedAt = prometheus.NewDesc(
		"github_exporter_pr_fetched_at",
		"UNIX timestamp of a Pull Request's last fetch time (when it was retrieved from the API)",
		[]string{"repo", "number"},
		nil,
	)

	pullRequestQueueSize = prometheus.NewDesc(
		"github_exporter_pr_queue_size",
		"Number of pull requests currently queued for an update",
		[]string{"repo", "queue"},
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

	issueInfo *prometheus.Desc

	issueLabelCount = prometheus.NewDesc(
		"github_exporter_issue_label_count",
		"Total count of Pull Requests using a given label",
		[]string{"repo", "label", "state"},
		nil,
	)

	issueCreatedAt = prometheus.NewDesc(
		"github_exporter_issue_created_at",
		"UNIX timestamp of an Issue's creation time",
		[]string{"repo", "number"},
		nil,
	)

	issueUpdatedAt = prometheus.NewDesc(
		"github_exporter_issue_updated_at",
		"UNIX timestamp of an Issue's last update time",
		[]string{"repo", "number"},
		nil,
	)

	issueFetchedAt = prometheus.NewDesc(
		"github_exporter_issue_fetched_at",
		"UNIX timestamp of an Issue's last fetch time (when it was retrieved from the API)",
		[]string{"repo", "number"},
		nil,
	)

	issueQueueSize = prometheus.NewDesc(
		"github_exporter_issue_queue_size",
		"Number of issues currently queued for an update",
		[]string{"repo", "queue"},
		nil,
	)
)

func init() {
	prLabels := []string{"repo", "number", "author", "state"}
	prLabels = append(prLabels, prow.PullRequestLabelNames()...)

	pullRequestInfo = prometheus.NewDesc(
		"github_exporter_pr_info",
		"Various Pull Request related meta information with the static value 1",
		prLabels,
		nil,
	)

	issueLabels := []string{"repo", "number", "author", "state"}
	issueLabels = append(issueLabels, prow.IssueLabelNames()...)

	issueInfo = prometheus.NewDesc(
		"github_exporter_issue_info",
		"Various issue related meta information with the static value 1",
		issueLabels,
		nil,
	)
}
