package tagging

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	tag "github.com/aws/aws-sdk-go-v2/service/resourcegroupstaggingapi"
	tagtypes "github.com/aws/aws-sdk-go-v2/service/resourcegroupstaggingapi/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockTagAPI struct {
	out *tag.GetResourcesOutput
	err error
}

func (m *mockTagAPI) GetResources(_ context.Context, _ *tag.GetResourcesInput, _ ...func(*tag.Options)) (*tag.GetResourcesOutput, error) {
	return m.out, m.err
}

func TestResourceTags(t *testing.T) {
	arn := "arn:aws:ec2:us-east-1:123456789012:instance/i-1234"
	mock := &mockTagAPI{
		out: &tag.GetResourcesOutput{
			ResourceTagMappingList: []tagtypes.ResourceTagMapping{
				{
					ResourceARN: aws.String(arn),
					Tags: []tagtypes.Tag{
						{Key: aws.String("env"), Value: aws.String("prod")},
						{Key: aws.String("team"), Value: aws.String("platform")},
					},
				},
			},
		},
	}

	client := NewFromAPI(mock)
	tags, err := client.ResourceTags(arn)
	require.NoError(t, err)
	assert.Equal(t, "prod", tags["env"])
	assert.Equal(t, "platform", tags["team"])
}

func TestResourceTags_NotFound(t *testing.T) {
	mock := &mockTagAPI{
		out: &tag.GetResourcesOutput{
			ResourceTagMappingList: nil,
		},
	}

	client := NewFromAPI(mock)
	tags, err := client.ResourceTags("arn:aws:ec2:us-east-1:123456789012:instance/i-missing")
	require.NoError(t, err)
	assert.Empty(t, tags)
}
