package metrics

import (
	"go.xrstf.de/github_exporter/pkg/prow"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	//////////////////////////////////////////////
	// repository

	repositoryDiskUsage = prometheus.NewDesc(
		"github_exporter_repo_disk_usage_bytes",
		"Repository size in bytes",
		[]string{"repo"},
		nil,
	)

	repositoryForks = prometheus.NewDesc(
		"github_exporter_repo_forks",
		"Number of forks of this repository",
		[]string{"repo"},
		nil,
	)

	repositoryStargazers = prometheus.NewDesc(
		"github_exporter_repo_stargazers",
		"Number of stargazers for this repository",
		[]string{"repo"},
		nil,
	)

	repositoryWatchers = prometheus.NewDesc(
		"github_exporter_repo_watchers",
		"Number of watchers for this repository",
		[]string{"repo"},
		nil,
	)

	repositoryPrivate = prometheus.NewDesc(
		"github_exporter_repo_is_private",
		"1 if the repository is private, 0 otherwise",
		[]string{"repo"},
		nil,
	)

	repositoryArchived = prometheus.NewDesc(
		"github_exporter_repo_is_archived",
		"1 if the repository is archived, 0 otherwise",
		[]string{"repo"},
		nil,
	)

	repositoryDisabled = prometheus.NewDesc(
		"github_exporter_repo_is_disabled",
		"1 if the repository is disabled, 0 otherwise",
		[]string{"repo"},
		nil,
	)

	repositoryFork = prometheus.NewDesc(
		"github_exporter_repo_is_fork",
		"1 if the repository is a fork, 0 otherwise",
		[]string{"repo"},
		nil,
	)

	repositoryLocked = prometheus.NewDesc(
		"github_exporter_repo_is_locked",
		"1 if the repository is locked, 0 otherwise",
		[]string{"repo"},
		nil,
	)

	repositoryMirror = prometheus.NewDesc(
		"github_exporter_repo_is_mirror",
		"1 if the repository is a mirror, 0 otherwise",
		[]string{"repo"},
		nil,
	)

	repositoryTemplate = prometheus.NewDesc(
		"github_exporter_repo_is_template",
		"1 if the repository is a template, 0 otherwise",
		[]string{"repo"},
		nil,
	)

	repositoryLanguageSize = prometheus.NewDesc(
		"github_exporter_repo_language_size_bytes",
		"Number of bytes in the repository detected as using a given language",
		[]string{"repo", "language"},
		nil,
	)

	//////////////////////////////////////////////
	// pull requests

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

	//////////////////////////////////////////////
	// issues

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

	//////////////////////////////////////////////
	// milestones

	milestoneInfo = prometheus.NewDesc(
		"github_exporter_milestone_info",
		"Various milestone related meta information with the static value 1",
		[]string{"repo", "number", "state", "title"},
		nil,
	)

	milestoneIssues = prometheus.NewDesc(
		"github_exporter_milestone_issues",
		"Total number issues (includes pull requests) belonging to a milestone, grouped by kind and state",
		[]string{"repo", "number", "kind", "state"},
		nil,
	)

	milestoneCreatedAt = prometheus.NewDesc(
		"github_exporter_milestone_created_at",
		"UNIX timestamp of a Milestone's creation time",
		[]string{"repo", "number"},
		nil,
	)

	milestoneUpdatedAt = prometheus.NewDesc(
		"github_exporter_milestone_updated_at",
		"UNIX timestamp of a Milestone's last update time",
		[]string{"repo", "number"},
		nil,
	)

	milestoneFetchedAt = prometheus.NewDesc(
		"github_exporter_milestone_fetched_at",
		"UNIX timestamp of a Milestone's last fetch time (when it was retrieved from the API)",
		[]string{"repo", "number"},
		nil,
	)

	milestoneClosedAt = prometheus.NewDesc(
		"github_exporter_milestone_closed_at",
		"UNIX timestamp of a Milestone's close time (0 if the milestone is open)",
		[]string{"repo", "number"},
		nil,
	)

	milestoneDueOn = prometheus.NewDesc(
		"github_exporter_milestone_due_on",
		"UNIX timestamp of a Milestone's due date (0 if there is no due date set)",
		[]string{"repo", "number"},
		nil,
	)

	milestoneQueueSize = prometheus.NewDesc(
		"github_exporter_milestone_queue_size",
		"Number of milestones currently queued for an update",
		[]string{"repo", "queue"},
		nil,
	)

	//////////////////////////////////////////////
	// exporter-related

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

	githubCostsTotal = prometheus.NewDesc(
		"github_exporter_api_costs_total",
		"Total sum of API credits spent for all performed API requests",
		[]string{"repo"},
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
