package github

import (
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

func (p *PullRequest) HasLabel(label string) bool {
	label = strings.ToLower(label)

	for _, l := range p.Labels {
		if label == strings.ToLower(l) {
			return true
		}
	}

	return false
}

func (p *PullRequest) Context(name string) *BuildContext {
	for i, ctx := range p.Contexts {
		if ctx.Name == name {
			return &p.Contexts[i]
		}
	}

	return nil
}
