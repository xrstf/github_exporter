package prow

import (
	"fmt"
	"regexp"
	"strings"

	"okp4/github-exporter/pkg/github"
)

func PullRequestLabelNames() []string {
	return []string{"approved", "lgtm", "pending", "size", "kind", "priority", "team"}
}

func PullRequestLabels(pr *github.PullRequest) []string {
	return []string{
		fmt.Sprintf("%v", pr.HasLabel("lgtm")),
		fmt.Sprintf("%v", pr.HasLabel("approved")),
		fmt.Sprintf("%v", prefixedLabel("do-not-merge", pr.Labels) != ""),
		prefixedLabel("size", pr.Labels),
		prefixedLabel("kind", pr.Labels),
		prefixedLabel("priority", pr.Labels),
		prefixedLabel("team", pr.Labels),
	}
}

func IssueLabelNames() []string {
	return []string{"kind", "priority", "team"}
}

func IssueLabels(issue *github.Issue) []string {
	return []string{
		prefixedLabel("kind", issue.Labels),
		prefixedLabel("priority", issue.Labels),
		prefixedLabel("team", issue.Labels),
	}
}

func prefixedLabel(prefix string, labels []string) string {
	prefix = strings.ToLower(strings.TrimSuffix(prefix, "/"))
	regex := regexp.MustCompile(fmt.Sprintf(`^%s/(.+)$`, prefix))

	for _, label := range labels {
		label := strings.ToLower(label)

		if match := regex.FindStringSubmatch(label); match != nil {
			return strings.ToLower(match[1])
		}
	}

	return ""
}
