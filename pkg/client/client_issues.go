package client

import (
	"strings"
	"time"

	"github.com/shurcooL/githubv4"
	"github.com/sirupsen/logrus"
	"go.xrstf.de/github_exporter/pkg/github"
)

type graphqlIssue struct {
	Number    int
	State     githubv4.IssueState
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
}

func (c *Client) convertIssue(api graphqlIssue, fetchedAt time.Time) github.Issue {
	issue := github.Issue{
		Number:    api.Number,
		Author:    api.Author.User.ID,
		State:     api.State,
		CreatedAt: api.CreatedAt,
		UpdatedAt: api.UpdatedAt,
		FetchedAt: fetchedAt,
		Labels:    []string{},
	}

	if c.realnames {
		issue.Author = api.Author.Login
	}

	for _, label := range api.Labels.Nodes {
		issue.Labels = append(issue.Labels, label.Name)
	}

	return issue
}

func (c *Client) GetRepositoryIssues(owner string, name string, numbers []int) ([]github.Issue, error) {
	variables := getNumberedQueryVariables(numbers, MaxIssuesPerQuery)
	variables["owner"] = githubv4.String(owner)
	variables["name"] = githubv4.String(name)

	var q numberedIssueQuery

	err := c.client.Query(c.ctx, &q, variables)
	c.countRequest(owner, name, q.RateLimit)

	c.log.WithFields(logrus.Fields{
		"owner":  owner,
		"name":   name,
		"issues": len(numbers),
		"cost":   q.RateLimit.Cost,
	}).Debugf("GetRepositoryIssues()")

	if err != nil && !strings.Contains(err.Error(), "Could not resolve to an Issue") {
		return nil, err
	}

	now := time.Now()
	issues := []github.Issue{}
	for _, issue := range q.GetAll() {
		issues = append(issues, c.convertIssue(issue, now))
	}

	return issues, nil
}

type listIssuesQuery struct {
	RateLimit  rateLimit
	Repository struct {
		Issues struct {
			Nodes    []graphqlIssue
			PageInfo struct {
				EndCursor   githubv4.String
				HasNextPage bool
			}
		} `graphql:"issues(states: $states, first: 100, orderBy: {field: UPDATED_AT, direction: DESC}, after: $cursor)"`
	} `graphql:"repository(owner: $owner, name: $name)"`
}

func (c *Client) ListIssues(owner string, name string, states []githubv4.IssueState, cursor string) ([]github.Issue, string, error) {
	if states == nil {
		states = []githubv4.IssueState{
			githubv4.IssueStateOpen,
			githubv4.IssueStateClosed,
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

	var q listIssuesQuery

	err := c.client.Query(c.ctx, &q, variables)
	c.countRequest(owner, name, q.RateLimit)

	c.log.WithFields(logrus.Fields{
		"owner":  owner,
		"name":   name,
		"cursor": cursor,
		"cost":   q.RateLimit.Cost,
	}).Debugf("ListIssues()")

	if err != nil {
		return nil, "", err
	}

	now := time.Now()
	issues := []github.Issue{}
	for _, node := range q.Repository.Issues.Nodes {
		issues = append(issues, c.convertIssue(node, now))
	}

	cursor = ""
	if q.Repository.Issues.PageInfo.HasNextPage {
		cursor = string(q.Repository.Issues.PageInfo.EndCursor)
	}

	return issues, cursor, nil
}
