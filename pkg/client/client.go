package client

import (
	"context"
	"errors"
	"fmt"

	"github.com/shurcooL/githubv4"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
)

type rateLimit struct {
	Cost      int
	Remaining int
}

var stopFetching = errors.New("stop fetching data pls")

type Client struct {
	ctx             context.Context
	client          *githubv4.Client
	log             logrus.FieldLogger
	requests        map[string]int
	remainingPoints int
}

func NewClient(ctx context.Context, log logrus.FieldLogger, token string) (*Client, error) {
	if token == "" {
		return nil, errors.New("token cannot be empty")
	}

	src := oauth2.StaticTokenSource(
		&oauth2.Token{
			AccessToken: token,
		},
	)
	httpClient := oauth2.NewClient(ctx, src)
	client := githubv4.NewClient(httpClient)

	return &Client{
		ctx:             ctx,
		client:          client,
		log:             log,
		requests:        map[string]int{},
		remainingPoints: 0,
	}, nil
}

func (c *Client) GetRemainingPoints() int {
	return c.remainingPoints
}

func (c *Client) GetRequestCounts() map[string]int {
	return c.requests
}

func (c *Client) countRequest(owner string, name string, rateLimit rateLimit) {
	key := fmt.Sprintf("%s/%s", owner, name)
	val := c.requests[key]

	c.requests[key] = val + 1
	c.remainingPoints = rateLimit.Remaining
}

func getNumberedQueryVariables(numbers []int, max int) map[string]interface{} {
	if len(numbers) > max {
		panic(fmt.Sprintf("List contains more (%d) than possible (%d) PR numbers.", len(numbers), max))
	}

	variables := map[string]interface{}{}

	for i := 0; i < max; i++ {
		number := 0
		has := false

		if i < len(numbers) {
			number = numbers[i]
			has = true
		}

		variables[fmt.Sprintf("number%d", i)] = githubv4.Int(number)
		variables[fmt.Sprintf("has%d", i)] = githubv4.Boolean(has)
	}

	return variables
}
