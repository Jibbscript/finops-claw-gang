// Package tagging wraps the AWS Resource Groups Tagging API to satisfy
// the ResourceTags portion of executor.TagFetcher.
package tagging

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	tag "github.com/aws/aws-sdk-go-v2/service/resourcegroupstaggingapi"
)

// API is the subset of the Tagging API client used by this package.
type API interface {
	GetResources(ctx context.Context, params *tag.GetResourcesInput, optFns ...func(*tag.Options)) (*tag.GetResourcesOutput, error)
}

// Client wraps the Resource Groups Tagging API.
type Client struct {
	api API
}

// New creates a Tagging client from an AWS config.
func New(cfg aws.Config) *Client {
	return &Client{api: tag.NewFromConfig(cfg)}
}

// NewFromAPI creates a Client from an explicit API implementation (for testing).
func NewFromAPI(api API) *Client {
	return &Client{api: api}
}

// ResourceTags returns tags for the given resource ARN as map[string]string.
func (c *Client) ResourceTags(resourceARN string) (map[string]string, error) {
	out, err := c.api.GetResources(context.TODO(), &tag.GetResourcesInput{
		ResourceARNList: []string{resourceARN},
	})
	if err != nil {
		return nil, fmt.Errorf("tagging: get resources: %w", err)
	}

	tags := make(map[string]string)
	for _, mapping := range out.ResourceTagMappingList {
		if mapping.ResourceARN == nil || *mapping.ResourceARN != resourceARN {
			continue
		}
		for _, t := range mapping.Tags {
			if t.Key != nil && t.Value != nil {
				tags[*t.Key] = *t.Value
			}
		}
	}
	return tags, nil
}
