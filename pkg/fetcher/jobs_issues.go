// SPDX-FileCopyrightText: 2023 Christoph Mewes
// SPDX-License-Identifier: MIT

package fetcher

import (
	"time"

	"github.com/sirupsen/logrus"

	"go.xrstf.de/github_exporter/pkg/github"
)

const (
	scanIssuesJobKey        = "scan-issues"
	updateIssuesJobKey      = "update-issues"
	findUpdatedIssuesJobKey = "find-updated-issues"
)

type updateIssuesJobMeta struct {
	numbers []int
}

// processUpdateIssuesJob updates a list of already fetched
// issues to ensure they stay up to date. This is done for all open
// issues.
func (f *Fetcher) processUpdateIssuesJob(repo *github.Repository, log logrus.FieldLogger, job string, data interface{}) error {
	meta := data.(updateIssuesJobMeta)

	issues, err := f.client.GetRepositoryIssues(repo.Owner, repo.Name, meta.numbers)

	fetchedNumbers := []int{}
	fetchedNumbersMap := map[int]struct{}{}
	for _, issue := range issues {
		fetchedNumbers = append(fetchedNumbers, issue.Number)
		fetchedNumbersMap[issue.Number] = struct{}{}
	}

	log.Debugf("Fetched %d out of %d issues.", len(fetchedNumbers), len(meta.numbers))

	deleted := []int{}
	for _, number := range meta.numbers {
		if _, ok := fetchedNumbersMap[number]; !ok {
			deleted = append(deleted, number)
		}
	}

	if len(issues) > 0 {
		repo.AddIssues(issues)
	}

	// only delete not found issues from our local cache if the request was a success, otherwise
	// we would remove all issues if e.g. GitHub is unavailable
	if err == nil && len(deleted) > 0 {
		repo.DeleteIssues(deleted)
	}

	f.removeJob(repo, job)
	f.dequeueIssues(repo, meta.numbers)

	return err
}

// processFindUpdatedIssuesJob fetches the 100 most recently updated
// issues in the given repository and updates repo. The job will be removed
// from the job queue afterwards and all fetched issues will be removed from
// the priority/regular issue queues.
func (f *Fetcher) processFindUpdatedIssuesJob(repo *github.Repository, log logrus.FieldLogger, job string) error {
	fetchedNumbers := []int{}

	issues, _, err := f.client.ListIssues(repo.Owner, repo.Name, nil, "")
	for _, issue := range issues {
		fetchedNumbers = append(fetchedNumbers, issue.Number)
	}

	log.Debugf("Fetched %d recently updated issues.", len(fetchedNumbers))

	repo.AddIssues(issues)

	f.removeJob(repo, job)
	f.dequeueIssues(repo, fetchedNumbers)

	return err
}

type scanIssuesJobMeta struct {
	max     int
	fetched int
	cursor  string
}

// processScanIssuesJob is the initial job for every repository.
// It lists all existing issues and adds them to repo.
//
// Because the initial scan is vital for proper functioning of every
// other job, this job must succeed before anything else can happen
// with a repository. For this reason a failed scan job is re-queued
// a few seconds later.
func (f *Fetcher) processScanIssuesJob(repo *github.Repository, log logrus.FieldLogger, job string, data interface{}) error {
	meta := data.(scanIssuesJobMeta)
	fullName := repo.FullName()
	fetchedNumbers := []int{}

	issues, cursor, err := f.client.ListIssues(repo.Owner, repo.Name, nil, meta.cursor)

	// if a max limit was set, enforce it (using ">=" here makes
	// it so that we stop cleanly when the list of issues is exactly
	// the right amount that was left to fetch)
	if meta.max > 0 && len(issues)+meta.fetched >= meta.max {
		issues = issues[:meta.max-meta.fetched]
		cursor = ""
	}

	for _, issue := range issues {
		fetchedNumbers = append(fetchedNumbers, issue.Number)
	}

	repo.AddIssues(issues)
	f.dequeueIssues(repo, fetchedNumbers)

	// always delete the job, no matter the outcome
	f.lock.Lock()
	delete(f.jobQueues[fullName], job)
	f.lock.Unlock()

	// batch query was successful
	if err == nil {
		log.WithField("new-cursor", cursor).Debugf("Fetched %d issues.", len(issues))

		// queue the query for the next page
		if cursor != "" {
			f.enqueueJob(repo, job, scanIssuesJobMeta{
				max:     meta.max,
				fetched: meta.fetched + len(issues),
				cursor:  cursor,
			})
		}

		return nil
	}

	retryAfter := 30 * time.Second
	log.Errorf("Failed to list issues, will retry in %s: %v", retryAfter.String(), err)

	// query failed, re-try later
	go func() {
		time.Sleep(retryAfter)
		f.enqueueJob(repo, job, data)
	}()

	return err
}
