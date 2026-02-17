// Package codedeploy wraps the AWS CodeDeploy API to satisfy
// the RecentDeploys portion of triage.InfraQuerier.
package codedeploy

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	cd "github.com/aws/aws-sdk-go-v2/service/codedeploy"
	cdtypes "github.com/aws/aws-sdk-go-v2/service/codedeploy/types"
)

// API is the subset of the CodeDeploy client used by this package.
type API interface {
	ListDeployments(ctx context.Context, params *cd.ListDeploymentsInput, optFns ...func(*cd.Options)) (*cd.ListDeploymentsOutput, error)
}

// Client wraps the CodeDeploy API.
type Client struct {
	api API
}

// New creates a CodeDeploy client from an AWS config.
func New(cfg aws.Config) *Client {
	return &Client{api: cd.NewFromConfig(cfg)}
}

// NewFromAPI creates a Client from an explicit API implementation (for testing).
func NewFromAPI(api API) *Client {
	return &Client{api: api}
}

// RecentDeploys returns deployments from the past 7 days in the fixture-compatible shape:
// []map[string]any with "id" field.
func (c *Client) RecentDeploys(service string) ([]map[string]any, error) {
	now := time.Now().UTC()
	sevenDaysAgo := now.Add(-7 * 24 * time.Hour)

	out, err := c.api.ListDeployments(context.TODO(), &cd.ListDeploymentsInput{
		ApplicationName: aws.String(service),
		CreateTimeRange: &cdtypes.TimeRange{
			Start: aws.Time(sevenDaysAgo),
			End:   aws.Time(now),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("codedeploy: list deployments: %w", err)
	}

	deploys := make([]map[string]any, 0, len(out.Deployments))
	for _, id := range out.Deployments {
		deploys = append(deploys, map[string]any{"id": id})
	}
	return deploys, nil
}
