package client

import (
	"github.com/shurcooL/githubv4"
	"github.com/sirupsen/logrus"
)

type repositoryLabelsQuery struct {
	RateLimit  rateLimit
	Repository struct {
		Labels struct {
			Nodes []struct {
				Name string
			}
			PageInfo struct {
				EndCursor   githubv4.String
				HasNextPage bool
			}
		} `graphql:"labels(first: 100, after: $cursor)"`
	} `graphql:"repository(owner: $owner, name: $name)"`
}

func (c *Client) RepositoryLabels(owner string, name string) ([]string, error) {
	variables := map[string]interface{}{
		"owner":  githubv4.String(owner),
		"name":   githubv4.String(name),
		"cursor": (*githubv4.String)(nil),
	}

	var q repositoryLabelsQuery

	labels := []string{}

	for {
		err := c.client.Query(c.ctx, &q, variables)
		c.countRequest(owner, name, q.RateLimit)

		c.log.WithFields(logrus.Fields{
			"owner":  owner,
			"name":   name,
			"cursor": variables["cursor"],
			"cost":   q.RateLimit.Cost,
		}).Debugf("RepositoryLabels()")

		if err != nil {
			return labels, err
		}

		for _, label := range q.Repository.Labels.Nodes {
			labels = append(labels, label.Name)
		}

		if !q.Repository.Labels.PageInfo.HasNextPage {
			break
		}

		variables["cursor"] = githubv4.NewString(q.Repository.Labels.PageInfo.EndCursor)
	}

	return labels, nil
}
