// SPDX-FileCopyrightText: 2023 Christoph Mewes
// SPDX-License-Identifier: MIT

package github

import (
	"strings"
	"time"

	"github.com/shurcooL/githubv4"
)

type Issue struct {
	Number    int
	Author    string
	State     githubv4.IssueState
	CreatedAt time.Time
	UpdatedAt time.Time
	FetchedAt time.Time
	Labels    []string
}

func (i *Issue) HasLabel(label string) bool {
	label = strings.ToLower(label)

	for _, l := range i.Labels {
		if label == strings.ToLower(l) {
			return true
		}
	}

	return false
}
