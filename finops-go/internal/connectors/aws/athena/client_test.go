package athena

import (
	"context"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	ath "github.com/aws/aws-sdk-go-v2/service/athena"
	athtypes "github.com/aws/aws-sdk-go-v2/service/athena/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockAthenaAPI struct {
	startOut *ath.StartQueryExecutionOutput
	startErr error
	execOut  *ath.GetQueryExecutionOutput
	execErr  error
	resOut   *ath.GetQueryResultsOutput
	resErr   error
}

func (m *mockAthenaAPI) StartQueryExecution(_ context.Context, _ *ath.StartQueryExecutionInput, _ ...func(*ath.Options)) (*ath.StartQueryExecutionOutput, error) {
	return m.startOut, m.startErr
}

func (m *mockAthenaAPI) GetQueryExecution(_ context.Context, _ *ath.GetQueryExecutionInput, _ ...func(*ath.Options)) (*ath.GetQueryExecutionOutput, error) {
	return m.execOut, m.execErr
}

func (m *mockAthenaAPI) GetQueryResults(_ context.Context, _ *ath.GetQueryResultsInput, _ ...func(*ath.Options)) (*ath.GetQueryResultsOutput, error) {
	return m.resOut, m.resErr
}

func TestBuildCURQuery_Valid(t *testing.T) {
	sql, err := buildCURQuery("my_cur_table", "123456789012", "2024-01-01", "2024-01-31", "EC2")
	require.NoError(t, err)
	assert.Contains(t, sql, "my_cur_table")
	assert.Contains(t, sql, "123456789012")
	assert.Contains(t, sql, "2024-01-01")
}

func TestBuildCURQuery_InvalidAccountID(t *testing.T) {
	_, err := buildCURQuery("t", "bad", "2024-01-01", "2024-01-31", "EC2")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid account ID")
}

func TestBuildCURQuery_InvalidDate(t *testing.T) {
	_, err := buildCURQuery("t", "123456789012", "bad-date", "2024-01-31", "EC2")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid start date")
}

func TestBuildCURQuery_InvalidService(t *testing.T) {
	_, err := buildCURQuery("t", "123456789012", "2024-01-01", "2024-01-31", "EC2'; DROP TABLE--")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid service")
}

func TestGetCURLineItems(t *testing.T) {
	mock := &mockAthenaAPI{
		startOut: &ath.StartQueryExecutionOutput{
			QueryExecutionId: aws.String("query-123"),
		},
		execOut: &ath.GetQueryExecutionOutput{
			QueryExecution: &athtypes.QueryExecution{
				Status: &athtypes.QueryExecutionStatus{
					State: athtypes.QueryExecutionStateSucceeded,
				},
			},
		},
		resOut: &ath.GetQueryResultsOutput{
			ResultSet: &athtypes.ResultSet{
				Rows: []athtypes.Row{
					{Data: []athtypes.Datum{
						{VarCharValue: aws.String("line_item_line_item_type")},
						{VarCharValue: aws.String("line_item_product_code")},
						{VarCharValue: aws.String("line_item_usage_type")},
						{VarCharValue: aws.String("product_product_name")},
						{VarCharValue: aws.String("line_item_unblended_cost")},
					}},
					{Data: []athtypes.Datum{
						{VarCharValue: aws.String("Usage")},
						{VarCharValue: aws.String("AmazonEC2")},
						{VarCharValue: aws.String("BoxUsage:m5.xlarge")},
						{VarCharValue: aws.String("Amazon Elastic Compute Cloud")},
						{VarCharValue: aws.String("150.75")},
					}},
				},
			},
		},
	}

	q := NewFromAPI(mock, "cur_db", "cur_table", "primary", "s3://output")
	items, err := q.GetCURLineItems("123456789012", "2024-01-01", "2024-01-31", "EC2")
	require.NoError(t, err)
	require.Len(t, items, 1)

	assert.Equal(t, "Usage", items[0]["line_item_line_item_type"])
	assert.Equal(t, "AmazonEC2", items[0]["line_item_product_code"])
	assert.InDelta(t, 150.75, items[0]["unblended_cost"].(float64), 0.01)
}

func TestGetCURLineItems_QueryFailed(t *testing.T) {
	mock := &mockAthenaAPI{
		startOut: &ath.StartQueryExecutionOutput{
			QueryExecutionId: aws.String("query-fail"),
		},
		execOut: &ath.GetQueryExecutionOutput{
			QueryExecution: &athtypes.QueryExecution{
				Status: &athtypes.QueryExecutionStatus{
					State:             athtypes.QueryExecutionStateFailed,
					StateChangeReason: aws.String("syntax error"),
				},
			},
		},
	}

	q := NewFromAPI(mock, "db", "tbl", "primary", "s3://out")
	_, err := q.GetCURLineItems("123456789012", "2024-01-01", "2024-01-31", "EC2")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "query failed")
}

func TestGetCURLineItems_StartError(t *testing.T) {
	mock := &mockAthenaAPI{
		startErr: fmt.Errorf("access denied"),
	}

	q := NewFromAPI(mock, "db", "tbl", "primary", "s3://out")
	_, err := q.GetCURLineItems("123456789012", "2024-01-01", "2024-01-31", "EC2")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "start query")
}
