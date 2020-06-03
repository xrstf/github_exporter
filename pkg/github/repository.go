package github

import (
	"fmt"
	"sync"

	"github.com/shurcooL/githubv4"
)

type Repository struct {
	Owner string
	Name  string

	PullRequests map[int]PullRequest
	Issues       map[int]Issue
	Labels       []string
	lock         sync.RWMutex
}

func NewRepository(owner string, name string) *Repository {
	return &Repository{
		Owner:        owner,
		Name:         name,
		PullRequests: map[int]PullRequest{},
		Issues:       map[int]Issue{},
		Labels:       []string{},
		lock:         sync.RWMutex{},
	}
}

func (d *Repository) FullName() string {
	return fmt.Sprintf("%s/%s", d.Owner, d.Name)
}

func (d *Repository) SetLabels(Labels []string) {
	d.lock.Lock()
	defer d.lock.Unlock()

	d.Labels = Labels
}

func (d *Repository) AddPullRequests(prs []PullRequest) {
	d.lock.Lock()
	defer d.lock.Unlock()

	for _, pr := range prs {
		d.PullRequests[pr.Number] = pr
	}
}

func (d *Repository) DeletePullRequests(numbers []int) {
	d.lock.Lock()
	defer d.lock.Unlock()

	for _, number := range numbers {
		delete(d.PullRequests, number)
	}
}

func (d *Repository) GetPullRequests(states ...githubv4.PullRequestState) []PullRequest {
	d.lock.RLock()
	defer d.lock.RUnlock()

	numbers := []PullRequest{}
	for _, pr := range d.PullRequests {
		include := false

		if len(states) == 0 {
			include = true
		} else {
			for _, state := range states {
				if pr.State == state {
					include = true
					break
				}
			}
		}

		if include {
			numbers = append(numbers, pr)
		}
	}

	return numbers
}

func (d *Repository) AddIssues(prs []Issue) {
	d.lock.Lock()
	defer d.lock.Unlock()

	for _, pr := range prs {
		d.Issues[pr.Number] = pr
	}
}

func (d *Repository) DeleteIssues(numbers []int) {
	d.lock.Lock()
	defer d.lock.Unlock()

	for _, number := range numbers {
		delete(d.Issues, number)
	}
}

func (d *Repository) GetIssues(states ...githubv4.IssueState) []Issue {
	d.lock.RLock()
	defer d.lock.RUnlock()

	numbers := []Issue{}
	for _, pr := range d.Issues {
		include := false

		if len(states) == 0 {
			include = true
		} else {
			for _, state := range states {
				if pr.State == state {
					include = true
					break
				}
			}
		}

		if include {
			numbers = append(numbers, pr)
		}
	}

	return numbers
}

func (d *Repository) Locked(callback func(*Repository) error) error {
	d.lock.Lock()
	defer d.lock.Unlock()

	return callback(d)
}

func (d *Repository) RLocked(callback func(*Repository) error) error {
	d.lock.RLock()
	defer d.lock.RUnlock()

	return callback(d)
}
