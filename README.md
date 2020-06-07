# xrstf's github_exporter for Prometheus

This github_exporter exposes Prometheus metrics for a list of pre-configured repositories.
The focus is on providing more insights about issues and pull requests.

It uses GitHub's API v4 and tries its best to not exceed the request quotas, but fore large
repositories (5k+ PRs) it's recommended to tweak the settings a bit.

## Operation

The goal of this particular exporter is to provide metrics for *all* pull requests and
issues within a given set of repositories. At the same time, *open* issues/PRs should be
refreshed much more often and quickly than older data.

To achieve this, the exporter upon startup scans all repositories for all PRs/issues. After
this is complete, it will

* fetch the most recently updated 100 PRs/issues (to detect new elements and elements
  whose status has changed),
* re-fetch all open PRs/issues frequently (every 5 minutes by default) and
* re-fetch *all* PRs/issues every 12 hours by default.

While the scheduling for the re-fetches happens concurrently in multiple go routines,
the fetching itself is done sequentially to avoid triggering GitHub's anti-abuse system.

Fetching open PRs/issues has higher priority, so that even large amounts of old PRs/issues
cannot interfere with the freshness of open PRs/issues.

It is possible to limit the initial scan, so that for very large repositories not all
items are fetched. But this only limits the initial scan, over time the exporter will
learn about new PRs/issues and not forget the old ones (and since it always keeps all
PRs/issues up-to-date, the number of items fetched will slooooowly over time grow).

## Installation

```
go get go.xrstf.de/github_exporter
```

## Usage

You need an OAuth2 token to authenticate against the API. Make it available
as the `GITHUB_TOKEN` environment variable.

All configuration happens via commandline arguments:

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
        address and port to listen on (default ":8080")
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
```

## Metrics

The following metrics are available:

* `github_exporter_pr_info`
* `github_exporter_pr_label_count`
* `github_exporter_pr_created_at`
* `github_exporter_pr_updated_at`
* `github_exporter_pr_fetched_at`
* `github_exporter_issue_info`
* `github_exporter_issue_label_count`
* `github_exporter_issue_created_at`
* `github_exporter_issue_updated_at`
* `github_exporter_issue_fetched_at`

The `_info` metrics contain a large number of labels and have a constant value of `1`,
whereas all other metrics only contain the bare minimum to identify an issue/PR
(repository + number).

And a few more metrics for monitoring the exporter itself are available as well:

* `github_exporter_pr_queue_size`
* `github_exporter_issue_queue_size`
* `github_exporter_api_requests_total`
* `github_exporter_api_points_remaining`

## License

MIT
