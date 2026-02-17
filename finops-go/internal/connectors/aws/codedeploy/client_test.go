package codedeploy

import (
	"context"
	"testing"

	cd "github.com/aws/aws-sdk-go-v2/service/codedeploy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockCDAPI struct {
	out *cd.ListDeploymentsOutput
	err error
}

func (m *mockCDAPI) ListDeployments(_ context.Context, _ *cd.ListDeploymentsInput, _ ...func(*cd.Options)) (*cd.ListDeploymentsOutput, error) {
	return m.out, m.err
}

func TestRecentDeploys(t *testing.T) {
	mock := &mockCDAPI{
		out: &cd.ListDeploymentsOutput{
			Deployments: []string{"d-ABC123", "d-DEF456"},
		},
	}

	client := NewFromAPI(mock)
	deploys, err := client.RecentDeploys("my-app")
	require.NoError(t, err)
	require.Len(t, deploys, 2)
	assert.Equal(t, "d-ABC123", deploys[0]["id"])
	assert.Equal(t, "d-DEF456", deploys[1]["id"])
}

func TestRecentDeploys_Empty(t *testing.T) {
	mock := &mockCDAPI{
		out: &cd.ListDeploymentsOutput{
			Deployments: nil,
		},
	}

	client := NewFromAPI(mock)
	deploys, err := client.RecentDeploys("my-app")
	require.NoError(t, err)
	assert.Empty(t, deploys)
}
