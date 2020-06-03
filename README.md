# xrstf's github_exporter for Prometheus

This github_exporter exposes Prometheus metrics for a list of pre-configured repositories.
The focus is on providing more insights about pull requests, so the following metrics are
available:

* github_exporter_pr_info
* github_exporter_pr_count
* github_exporter_pr_mergable_count
* github_exporter_label_pr_count
* github_exporter_pr_created_at
* github_exporter_pr_updated_at

It uses GitHub's API v4 and tries its best to not exceed the request quotas, but fore large
repositories (5k+ PRs) it's recommended to tweak the settings a bit.
