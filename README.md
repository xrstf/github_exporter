# xrstf's GitHub Exporter for Prometheus

This exporter exposes Prometheus metrics for a list of pre-configured GitHub repositories.
The focus is on providing more insights about issues, pull requests and milestones.

![Grafana Screenshot](https://github.com/xrstf/github_exporter/blob/main/contrib/grafana/screenshot.png?raw=true)

It uses GitHub's API v4 and tries its best to not exceed the request quotas, but for large
repositories (5k+ PRs) it's recommended to tweak the settings a bit.

## Operation

The goal of this particular exporter is to provide metrics for **all** pull requests, issues
and milestones (collectively called "items" from here on) within a given set of repositories.
At the same time, **open** items should be refreshed much more often and quickly than older
data.

To achieve this, the exporter upon startup scans all repositories for all items. After
this is complete, it will

* fetch the most recently updated 100 items (to detect new elements and elements
  whose status has changed),
* re-fetch all open items frequently (every 5 minutes by default) and
* re-fetch **all** items every 12 hours by default.

While the scheduling for the re-fetches happens concurrently in multiple go routines,
the fetching itself is done sequentially to avoid triggering GitHub's anti-abuse system.

Fetching open items has higher priority, so that even large amounts of old items
cannot interfere with the freshness of open items.

It is possible to limit the initial scan (using `-pr-depth`, `-issue-depth` and `-milestone-depth`),
so that for very large repositories not all items are fetched. But this only limits the
initial scan, over time the exporter will learn about new items and not forget the old ones
(and since it always keeps all items up-to-date, the number of items fetched will slooooowly
over time grow).

Jobs are always removed from the queue, even if they failed. The exporter relies on the
goroutines to re-schedule them later anyway, and this prevents flooding GitHub when the
API has issues or misconfiguration occurs. Job queues can only contain one job per kind,
so even if the API is down for an hour, the queue will not fill up with the re-fetch job.

## Installation

You need Go 1.14 installed on your machine.

```
go get go.xrstf.de/github_exporter
```

A Docker image is available as [`xrstf/github_exporter`](https://hub.docker.com/r/xrstf/github_exporter).

## Usage

You need an OAuth2 token to authenticate against the API. Make it available
as the `GITHUB_TOKEN` environment variable.

By default, the exporter listens on `0.0.0.0:9612`.

All configuration happens via commandline arguments. At the bare minimum, you need to
specify a single repository to scrape:

```
./github_exporter -repo myself/my-repository
```

You can configure multiple `-repo` (which is also recommended over running the exporter
multiple times in parallel, so a single exporter can serialize all API requests) and
tweak the exporter further using the available flags:

```
Usage of ./github_exporter:
  -debug
        enable more verbose logging
  -issue-depth int
        max number of issues to fetch per repository upon startup (-1 disables the limit, 0 disables issue fetching entirely) (default -1)
  -issue-refresh-interval duration
        time in between issue refreshes (default 5m0s)
  -issue-resync-interval duration
        time in between full issue re-syncs (default 12h0m0s)
  -listen string
        address and port to listen on (default ":9612")
  -milestone-depth int
        max number of milestones to fetch per repository upon startup (-1 disables the limit, 0 disables milestone fetching entirely) (default -1)
  -milestone-refresh-interval duration
        time in between milestone refreshes (default 5m0s)
  -milestone-resync-interval duration
        time in between full milestone re-syncs (default 12h0m0s)
  -pr-depth int
        max number of pull requests to fetch per repository upon startup (-1 disables the limit, 0 disables PR fetching entirely) (default -1)
  -pr-refresh-interval duration
        time in between PR refreshes (default 5m0s)
  -pr-resync-interval duration
        time in between full PR re-syncs (default 12h0m0s)
  -realnames
        use usernames instead of internal IDs for author labels (this will make metrics contain personally identifiable information)
  -repo value
        repository (owner/name format) to include, can be given multiple times
  -owner string
        github login (username or organization) of the owner of the repositories that will be included. Excludes forked and locked repo, includes 100 first private & public repos
```

## Metrics

**All** metrics are labelled with `repo=(full repo name)`, for example
`repo="xrstf/github_exporter"`.

For each repository, the following metrics are available:

* `github_exporter_repo_disk_usage_bytes`
* `github_exporter_repo_forks`
* `github_exporter_repo_stargazers`
* `github_exporter_repo_watchers`
* `github_exporter_repo_is_private`
* `github_exporter_repo_is_archived`
* `github_exporter_repo_is_disabled`
* `github_exporter_repo_is_fork`
* `github_exporter_repo_is_locked`
* `github_exporter_repo_is_mirror`
* `github_exporter_repo_is_template`
* `github_exporter_repo_language_size_bytes` is additionally labelled with `language`.

For pull requests, these metrics are available:

* `github_exporter_pr_info` contains lots of metadata labels and always has a constant
  value of `1`. Labels are:

  * `number` is the PR's number.
  * `state` is one of `open`, `closed` or `merged`.
  * `author` is the author ID (or username if `-realnames` is configured).

  In addition, the exporter recognizes a few common label conventions, namely:

  * `size/*` is reflected as a `size` label (e.g. the `size/xs` label on GitHub becomes
    a `size="xs"` label on the Prometheus metric).
  * `team/*` is reflected as a `team` label.
  * `kind/*` is reflected as a `kind` label.
  * `priority/*` is reflected as a `priority` label.
  * `approved` is reflected as a boolean `approved` label.
  * `lgtm` is reflected as a boolean `lgtm` label.
  * `do-no-merge/*` is reflected as a boolean `pending` label.

* `github_exporter_pr_label_count` is the number of PRs that have a given label
  and state. This counts all labels individually, not just those recognized for
  the `_info` metric.

* `github_exporter_pr_created_at` is the UNIX timestamp of when the PR was
  created on GitHub. This metric only has `repo` and `number` labels.

* `github_exporter_pr_updated_at` is the UNIX timestamp of when the PR was
  last updated on GitHub. This metric only has `repo` and `number` labels.

* `github_exporter_pr_fetched_at` is the UNIX timestamp of when the PR was
  last fetched from the GitHub API. This metric only has `repo` and `number` labels.

The PR metrics are mirrored for issues:

* `github_exporter_issue_info`
* `github_exporter_issue_label_count`
* `github_exporter_issue_created_at`
* `github_exporter_issue_updated_at`
* `github_exporter_issue_fetched_at`

The metrics for milestones are similar:

* `github_exporter_milestone_info` has `repo`, `number`, `title` and `state` labels.
* `github_exporter_milestone_issues` counts the number of open/closed issues/PRs
  for a given milestone, so it has `repo`, `number`, `kind` (issue or pullrequest)
  and `state` labels.
* `github_exporter_milestone_created_at`
* `github_exporter_milestone_updated_at`
* `github_exporter_milestone_fetched_at`
* `github_exporter_milestone_closed_at` is optional and 0 if the milestone is open.
* `github_exporter_milestone_due_on` is optional and 0 if no due date is set.

And a few more metrics for monitoring the exporter itself are available as well:

* `github_exporter_pr_queue_size` is the number of PRs currently queued for
  being fetched from the API. This is split via the `queue` label into `priority`
  (open PRs) and `regular` (older PRs).
* `github_exporter_issue_queue_size` is the same as for the PR queue.
* `github_exporter_milestone_queue_size` is the same as for the PR queue.
* `github_exporter_api_requests_total` counts the number of API requests per
  repository.
* `github_exporter_api_costs_total` is the sum of costs (in API points) that have
  been used, grouped by `repo`.
* `github_exporter_api_points_remaining` is a gauge representing the remaining
  API points. 5k points can be consumed per hour, with resets after 1 hour.

## Long-term storage

If you plan on performing long-term analysis over repositories, make sure to put proper
recording rules into place so that queries can be performed quickly. The exporter
intentionally does not pre-aggregate most things, as to not spam Prometheus or restrict
the available information.

A few example rules can be found in `contrib/prometheus/rules.yaml`.

## License

MIT
