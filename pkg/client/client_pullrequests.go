package client

import (
	"strings"
	"time"

	"github.com/shurcooL/githubv4"
	"github.com/sirupsen/logrus"

	"go.xrstf.de/github_exporter/pkg/github"
)

type graphqlPullRequest struct {
	Number    int
	State     githubv4.PullRequestState
	CreatedAt time.Time
	UpdatedAt time.Time

	Author struct {
		Login string
		User  struct {
			ID string
		} `graphql:"... on User"`
	}

	Labels struct {
		Nodes []struct {
			Name string
		}
	} `graphql:"labels(first: 50)"`

	Commits struct {
		Nodes []struct {
			Commit struct {
				Status struct {
					Contexts []struct {
						Context string
						State   githubv4.StatusState
					}
				}
			}
		}
	} `graphql:"commits(last: 1)"`
}

func (c *Client) convertPullRequest(api graphqlPullRequest, fetchedAt time.Time) github.PullRequest {
	pr := github.PullRequest{
		Number:    api.Number,
		Author:    api.Author.User.ID,
		State:     api.State,
		CreatedAt: api.CreatedAt,
		UpdatedAt: api.UpdatedAt,
		FetchedAt: fetchedAt,
		Labels:    []string{},
		Contexts:  []github.BuildContext{},
	}

	if c.realnames {
		pr.Author = api.Author.Login
	}

	for _, label := range api.Labels.Nodes {
		pr.Labels = append(pr.Labels, label.Name)
	}

	if len(api.Commits.Nodes) > 0 {
		for _, context := range api.Commits.Nodes[0].Commit.Status.Contexts {
			pr.Contexts = append(pr.Contexts, github.BuildContext{
				Name:  context.Context,
				State: context.State,
			})
		}
	}

	return pr
}

func (c *Client) GetRepositoryPullRequests(owner string, name string, numbers []int) ([]github.PullRequest, error) {
	variables := getNumberedQueryVariables(numbers, MaxPullRequestsPerQuery)
	variables["owner"] = githubv4.String(owner)
	variables["name"] = githubv4.String(name)

	var q numberedPullRequestQuery

	err := c.client.Query(c.ctx, &q, variables)
	c.countRequest(owner, name, q.RateLimit)

	c.log.WithFields(logrus.Fields{
		"owner": owner,
		"name":  name,
		"prs":   len(numbers),
		"cost":  q.RateLimit.Cost,
	}).Debugf("GetRepositoryPullRequests()")

	if err != nil && !strings.Contains(err.Error(), "Could not resolve to a PullRequest") {
		return nil, err
	}

	now := time.Now()
	prs := []github.PullRequest{}
	for _, pr := range q.GetAll() {
		prs = append(prs, c.convertPullRequest(pr, now))
	}

	return prs, nil
}

type listPullRequestsQuery struct {
	RateLimit  rateLimit
	Repository struct {
		PullRequests struct {
			Nodes    []graphqlPullRequest
			PageInfo struct {
				EndCursor   githubv4.String
				HasNextPage bool
			}
		} `graphql:"pullRequests(states: $states, first: 100, orderBy: {field: UPDATED_AT, direction: DESC}, after: $cursor)"`
	} `graphql:"repository(owner: $owner, name: $name)"`
}

func (c *Client) ListPullRequests(owner string, name string, states []githubv4.PullRequestState, cursor string) ([]github.PullRequest, string, error) {
	if states == nil {
		states = []githubv4.PullRequestState{
			githubv4.PullRequestStateClosed,
			githubv4.PullRequestStateMerged,
			githubv4.PullRequestStateOpen,
		}
	}

	variables := map[string]interface{}{
		"owner":  githubv4.String(owner),
		"name":   githubv4.String(name),
		"states": states,
	}

	if cursor == "" {
		variables["cursor"] = (*githubv4.String)(nil)
	} else {
		variables["cursor"] = githubv4.String(cursor)
	}

	var q listPullRequestsQuery

	err := c.client.Query(c.ctx, &q, variables)
	c.countRequest(owner, name, q.RateLimit)

	c.log.WithFields(logrus.Fields{
		"owner":  owner,
		"name":   name,
		"cursor": cursor,
		"cost":   q.RateLimit.Cost,
	}).Debugf("ListPullRequests()")

	if err != nil {
		return nil, "", err
	}

	now := time.Now()
	prs := []github.PullRequest{}
	for _, node := range q.Repository.PullRequests.Nodes {
		prs = append(prs, c.convertPullRequest(node, now))
	}

	cursor = ""
	if q.Repository.PullRequests.PageInfo.HasNextPage {
		cursor = string(q.Repository.PullRequests.PageInfo.EndCursor)
	}

	return prs, cursor, nil
}
