package main

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

type repository struct {
	owner string
	name  string
}

func (r *repository) String() string {
	return fmt.Sprintf("%s/%s", r.owner, r.name)
}

type repositoryList []repository

func (l *repositoryList) String() string {
	return fmt.Sprint(*l)
}

func (l *repositoryList) Set(value string) error {
	parts := strings.Split(value, "/")

	if len(parts) != 2 {
		return errors.New(`not a valid repository name, must be "owner/name"`)
	}

	*l = append(*l, repository{
		owner: parts[0],
		name:  parts[1],
	})

	return nil
}

type stringSlice []string

func (l *stringSlice) String() string {
	return fmt.Sprint(*l)
}

func (l *stringSlice) Set(value string) error {
	*l = append(*l, value)
	return nil
}

func getOptions(ctx context.Context) options {
	return ctx.Value(OptionsCtxKey).(options)
}
