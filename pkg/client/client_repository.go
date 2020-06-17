package client

import (
	"github.com/shurcooL/githubv4"
	"github.com/sirupsen/logrus"
)

type repositoryInfoQuery struct {
	RateLimit  rateLimit
	Repository struct {
		DiskUsage  int
		ForkCount  int
		Stargazers struct {
			TotalCount int
		}
		Watchers struct {
			TotalCount int
		}
		IsPrivate  bool
		IsArchived bool
		IsDisabled bool
		IsFork     bool
		IsLocked   bool
		IsMirror   bool
		IsTemplate bool
		Languages  struct {
			Edges []struct {
				Size int
				Node struct {
					Name string
				}
			}
		} `graphql:"languages(first: 100)"`
	} `graphql:"repository(owner: $owner, name: $name)"`
}

type RepositoryInfo struct {
	// DiskUsage is returned in KBytes
	DiskUsage  int
	Forks      int
	Stargazers int
	Watchers   int
	IsPrivate  bool
	IsArchived bool
	IsDisabled bool
	IsFork     bool
	IsLocked   bool
	IsMirror   bool
	IsTemplate bool
	Languages  map[string]int
}

func (c *Client) RepositoryInfo(owner string, name string) (*RepositoryInfo, error) {
	variables := map[string]interface{}{
		"owner": githubv4.String(owner),
		"name":  githubv4.String(name),
	}

	var q repositoryInfoQuery

	err := c.client.Query(c.ctx, &q, variables)
	c.countRequest(owner, name, q.RateLimit)

	c.log.WithFields(logrus.Fields{
		"owner": owner,
		"name":  name,
		"cost":  q.RateLimit.Cost,
	}).Debugf("RepositoryInfo()")

	if err != nil {
		return nil, err
	}

	info := &RepositoryInfo{
		DiskUsage:  q.Repository.DiskUsage,
		Forks:      q.Repository.ForkCount,
		Stargazers: q.Repository.Stargazers.TotalCount,
		Watchers:   q.Repository.Watchers.TotalCount,
		IsPrivate:  q.Repository.IsPrivate,
		IsArchived: q.Repository.IsArchived,
		IsDisabled: q.Repository.IsDisabled,
		IsFork:     q.Repository.IsFork,
		IsLocked:   q.Repository.IsLocked,
		IsMirror:   q.Repository.IsMirror,
		IsTemplate: q.Repository.IsTemplate,
		Languages:  map[string]int{},
	}

	for _, lang := range q.Repository.Languages.Edges {
		info.Languages[lang.Node.Name] = lang.Size
	}

	return info, nil
}
