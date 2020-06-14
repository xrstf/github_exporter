package fetcher

import (
	"github.com/sirupsen/logrus"

	"go.xrstf.de/github_exporter/pkg/github"
)

const (
	updateLabelsJobKey   = "update-labels"
	updateRepoInfoJobKey = "update-repository-info"
)

type jobQueue map[string]interface{}

// processUpdateLabelsJob fetches the repository's labels and removes
// the job afterwards.
func (f *Fetcher) processUpdateLabelsJob(repo *github.Repository, log logrus.FieldLogger, job string) error {
	labels, err := f.client.RepositoryLabels(repo.Owner, repo.Name)

	log.Debugf("Fetched %d labels.", len(labels))

	repo.SetLabels(labels)
	f.removeJob(repo, job)

	return err
}

// processUpdateRepoInfos fetches the repository's metadata.
func (f *Fetcher) processUpdateRepoInfos(repo *github.Repository, log logrus.FieldLogger, job string) error {
	info, err := f.client.RepositoryInfo(repo.Owner, repo.Name)

	if info != nil {
		repo.Locked(func(r *github.Repository) error {
			r.DiskUsage = info.DiskUsage
			r.Forks = info.Forks
			r.Stargazers = info.Stargazers
			r.Watchers = info.Watchers
			r.IsPrivate = info.IsPrivate
			r.IsArchived = info.IsArchived
			r.IsDisabled = info.IsDisabled
			r.IsFork = info.IsFork
			r.IsLocked = info.IsLocked
			r.IsMirror = info.IsMirror
			r.IsTemplate = info.IsTemplate
			r.Languages = info.Languages

			return nil
		})
	}

	f.removeJob(repo, job)

	return err
}
