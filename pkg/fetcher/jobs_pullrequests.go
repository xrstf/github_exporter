// SPDX-FileCopyrightText: 2023 Christoph Mewes
// SPDX-License-Identifier: MIT

package fetcher

import (
	"time"

	"github.com/sirupsen/logrus"

	"go.xrstf.de/github_exporter/pkg/github"
)

const (
	scanPullRequestsJobKey        = "scan-pull-requests"
	updatePullRequestsJobKey      = "update-pull-requests"
	findUpdatedPullRequestsJobKey = "find-updated-pull-requests"
)

type updatePullRequestsJobMeta struct {
	numbers []int
}

// processUpdatePullRequestsJob updates a list of already fetched
// PRs to ensure they stay up to date. This is done for all open
// PRs.
func (f *Fetcher) processUpdatePullRequestsJob(repo *github.Repository, log logrus.FieldLogger, job string, data interface{}) error {
	meta := data.(updatePullRequestsJobMeta)

	prs, err := f.client.GetRepositoryPullRequests(repo.Owner, repo.Name, meta.numbers)

	fetchedNumbers := []int{}
	fetchedNumbersMap := map[int]struct{}{}
	for _, pr := range prs {
		fetchedNumbers = append(fetchedNumbers, pr.Number)
		fetchedNumbersMap[pr.Number] = struct{}{}
	}

	log.Debugf("Fetched %d out of %d PRs.", len(fetchedNumbers), len(meta.numbers))

	deleted := []int{}
	for _, number := range meta.numbers {
		if _, ok := fetchedNumbersMap[number]; !ok {
			deleted = append(deleted, number)
		}
	}

	if len(prs) > 0 {
		repo.AddPullRequests(prs)
	}

	// only delete not found PRs from our local cache if the request was a success, otherwise
	// we would remove all PRs if e.g. GitHub is unavailable
	if err == nil && len(deleted) > 0 {
		repo.DeletePullRequests(deleted)
	}

	f.removeJob(repo, job)
	f.dequeuePullRequests(repo, meta.numbers)

	return err
}

// processFindUpdatedPullRequestsJob fetches the 100 most recently updated
// PRs in the given repository and updates repo. The job will be removed
// from the job queue afterwards and all fetched PRs will be removed from
// the priority/regular PR queues.
func (f *Fetcher) processFindUpdatedPullRequestsJob(repo *github.Repository, log logrus.FieldLogger, job string) error {
	fetchedNumbers := []int{}

	prs, _, err := f.client.ListPullRequests(repo.Owner, repo.Name, nil, "")
	for _, pr := range prs {
		fetchedNumbers = append(fetchedNumbers, pr.Number)
	}

	log.Debugf("Fetched %d recently updated PRs.", len(fetchedNumbers))

	repo.AddPullRequests(prs)

	f.removeJob(repo, job)
	f.dequeuePullRequests(repo, fetchedNumbers)

	return err
}

type scanPullRequestsJobMeta struct {
	max     int
	fetched int
	cursor  string
}

// processScanPullRequestsJob is the initial job for every repository.
// It lists all existing pull requests and adds them to repo.
//
// Because the initial scan is vital for proper functioning of every
// other job, this job must succeed before anything else can happen
// with a repository. For this reason a failed scan job is re-queued
// a few seconds later.
func (f *Fetcher) processScanPullRequestsJob(repo *github.Repository, log logrus.FieldLogger, job string, data interface{}) error {
	meta := data.(scanPullRequestsJobMeta)
	fullName := repo.FullName()
	fetchedNumbers := []int{}

	prs, cursor, err := f.client.ListPullRequests(repo.Owner, repo.Name, nil, meta.cursor)

	// if a max limit was set, enforce it (using ">=" here makes
	// it so that we stop cleanly when the list of PRs is exactly
	// the right amount that was left to fetch)
	if meta.max > 0 && len(prs)+meta.fetched >= meta.max {
		prs = prs[:meta.max-meta.fetched]
		cursor = ""
	}

	for _, pr := range prs {
		fetchedNumbers = append(fetchedNumbers, pr.Number)
	}

	repo.AddPullRequests(prs)
	f.dequeuePullRequests(repo, fetchedNumbers)

	// always delete the job, no matter the outcome
	f.lock.Lock()
	delete(f.jobQueues[fullName], job)
	f.lock.Unlock()

	// batch query was successful
	if err == nil {
		log.WithField("new-cursor", cursor).Debugf("Fetched %d PRs.", len(prs))

		// queue the query for the next page
		if cursor != "" {
			f.enqueueJob(repo, job, scanPullRequestsJobMeta{
				max:     meta.max,
				fetched: meta.fetched + len(prs),
				cursor:  cursor,
			})
		}

		return nil
	}

	retryAfter := 30 * time.Second
	log.Errorf("Failed to list PRs, will retry in %s: %v", retryAfter.String(), err)

	// query failed, re-try later
	go func() {
		time.Sleep(retryAfter)
		f.enqueueJob(repo, job, data)
	}()

	return err
}
