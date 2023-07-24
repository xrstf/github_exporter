// SPDX-FileCopyrightText: 2023 Christoph Mewes
// SPDX-License-Identifier: MIT

package github

import (
	"time"

	"github.com/shurcooL/githubv4"
)

type Milestone struct {
	Number             int
	Title              string
	State              githubv4.MilestoneState
	CreatedAt          time.Time
	UpdatedAt          time.Time
	ClosedAt           *time.Time
	DueOn              *time.Time
	FetchedAt          time.Time
	OpenIssues         int
	ClosedIssues       int
	OpenPullRequests   int
	ClosedPullRequests int
}
