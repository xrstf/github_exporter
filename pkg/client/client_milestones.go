package client

import (
	"strings"
	"time"

	"okp4/github-exporter/pkg/github"

	"github.com/shurcooL/githubv4"
	"github.com/sirupsen/logrus"
)

type graphqlMilestone struct {
	Number    int
	Title     string
	State     githubv4.MilestoneState
	CreatedAt time.Time
	UpdatedAt time.Time
	ClosedAt  *time.Time
	DueOn     *time.Time

	OpenIssues struct {
		TotalCount int
	} `graphql:"openIssues: issues(states: OPEN)"`

	ClosedIssues struct {
		TotalCount int
	} `graphql:"closedIssues: issues(states: CLOSED)"`

	OpenPullRequests struct {
		TotalCount int
	} `graphql:"openPullRequests: pullRequests(states: OPEN)"`

	ClosedPullRequests struct {
		TotalCount int
	} `graphql:"closedPullRequests: pullRequests(states: [MERGED, CLOSED])"`
}

func (c *Client) convertMilestone(api graphqlMilestone, fetchedAt time.Time) github.Milestone {
	return github.Milestone{
		Number:             api.Number,
		Title:              api.Title,
		State:              api.State,
		CreatedAt:          api.CreatedAt,
		UpdatedAt:          api.UpdatedAt,
		ClosedAt:           api.ClosedAt,
		DueOn:              api.DueOn,
		FetchedAt:          fetchedAt,
		OpenIssues:         api.OpenIssues.TotalCount,
		ClosedIssues:       api.ClosedIssues.TotalCount,
		OpenPullRequests:   api.OpenPullRequests.TotalCount,
		ClosedPullRequests: api.ClosedPullRequests.TotalCount,
	}
}

func (c *Client) GetRepositoryMilestones(owner string, name string, numbers []int) ([]github.Milestone, error) {
	variables := getNumberedQueryVariables(numbers, MaxMilestonesPerQuery)
	variables["owner"] = githubv4.String(owner)
	variables["name"] = githubv4.String(name)

	var q numberedMilestoneQuery

	err := c.client.Query(c.ctx, &q, variables)
	c.countRequest(owner, name, q.RateLimit)

	c.log.WithFields(logrus.Fields{
		"owner":      owner,
		"name":       name,
		"milestones": len(numbers),
		"cost":       q.RateLimit.Cost,
	}).Debugf("GetRepositoryMilestones()")

	// As of 2020-06-12, the GitHub API does not return an error, but instead just sets
	// the milestone field to null if it was not found. For safety we keep the check anyway.
	if err != nil && !strings.Contains(err.Error(), "Could not resolve to a Milestone") {
		return nil, err
	}

	now := time.Now()
	milestones := []github.Milestone{}
	for _, milestone := range q.GetAll() {
		milestones = append(milestones, c.convertMilestone(milestone, now))
	}

	return milestones, nil
}

type listMilestonesQuery struct {
	RateLimit  rateLimit
	Repository struct {
		Milestones struct {
			Nodes    []graphqlMilestone
			PageInfo struct {
				EndCursor   githubv4.String
				HasNextPage bool
			}
		} `graphql:"milestones(states: $states, first: 100, orderBy: {field: UPDATED_AT, direction: DESC}, after: $cursor)"`
	} `graphql:"repository(owner: $owner, name: $name)"`
}

func (c *Client) ListMilestones(owner string, name string, states []githubv4.MilestoneState, cursor string) ([]github.Milestone, string, error) {
	if states == nil {
		states = []githubv4.MilestoneState{
			githubv4.MilestoneStateOpen,
			githubv4.MilestoneStateClosed,
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

	var q listMilestonesQuery

	err := c.client.Query(c.ctx, &q, variables)
	c.countRequest(owner, name, q.RateLimit)

	c.log.WithFields(logrus.Fields{
		"owner":  owner,
		"name":   name,
		"cursor": cursor,
		"cost":   q.RateLimit.Cost,
	}).Debugf("ListMilestones()")

	if err != nil {
		return nil, "", err
	}

	now := time.Now()
	milestones := []github.Milestone{}
	for _, node := range q.Repository.Milestones.Nodes {
		milestones = append(milestones, c.convertMilestone(node, now))
	}

	cursor = ""
	if q.Repository.Milestones.PageInfo.HasNextPage {
		cursor = string(q.Repository.Milestones.PageInfo.EndCursor)
	}

	return milestones, cursor, nil
}
