package fetcher

import (
	"github.com/sirupsen/logrus"

	"go.xrstf.de/github_exporter/pkg/github"
)

const (
	updateLabelsJobKey = "update-labels"
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
