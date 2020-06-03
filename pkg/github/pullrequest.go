package github

import (
	"regexp"
	"strings"
	"time"

	"github.com/shurcooL/githubv4"
)

type BuildContext struct {
	Name  string
	State githubv4.StatusState
}

type PullRequest struct {
	Number    int
	State     githubv4.PullRequestState
	CreatedAt time.Time
	UpdatedAt time.Time
	FetchedAt time.Time
	Labels    []string
	Contexts  []BuildContext
}

func (p *PullRequest) Mergable() bool {
	if p.State != githubv4.PullRequestStateOpen {
		return false
	}

	for _, ctx := range p.Contexts {
		if ctx.Name == "tide" {
			continue
		}

		if ctx.State != githubv4.StatusStateSuccess {
			return false
		}
	}

	return true
}

func (p *PullRequest) HasLabel(label string) bool {
	label = strings.ToLower(label)

	for _, l := range p.Labels {
		if label == strings.ToLower(l) {
			return true
		}
	}

	return false
}

var sizeRegex = regexp.MustCompile(`^size/(.+)$`)

func (p *PullRequest) Size() string {
	for _, label := range p.Labels {
		label := strings.ToLower(label)

		if match := sizeRegex.FindStringSubmatch(label); match != nil {
			return strings.ToLower(match[1])
		}
	}

	return ""
}
