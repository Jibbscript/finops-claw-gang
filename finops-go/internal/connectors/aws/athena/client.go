// Package athena wraps the AWS Athena API to query CUR data,
// satisfying the GetCURLineItems portion of triage.CostFetcher.
package athena

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	ath "github.com/aws/aws-sdk-go-v2/service/athena"
	athtypes "github.com/aws/aws-sdk-go-v2/service/athena/types"
)

const (
	pollInterval = 2 * time.Second
	pollTimeout  = 120 * time.Second
)

// API is the subset of the Athena client used by this package.
type API interface {
	StartQueryExecution(ctx context.Context, params *ath.StartQueryExecutionInput, optFns ...func(*ath.Options)) (*ath.StartQueryExecutionOutput, error)
	GetQueryExecution(ctx context.Context, params *ath.GetQueryExecutionInput, optFns ...func(*ath.Options)) (*ath.GetQueryExecutionOutput, error)
	GetQueryResults(ctx context.Context, params *ath.GetQueryResultsInput, optFns ...func(*ath.Options)) (*ath.GetQueryResultsOutput, error)
}

// Querier queries CUR data via Athena.
type Querier struct {
	api       API
	database  string
	table     string
	workgroup string
	outputLoc string
}

// New creates a Querier from an AWS config and CUR table configuration.
func New(cfg aws.Config, database, table, workgroup, outputLoc string) *Querier {
	return &Querier{
		api:       ath.NewFromConfig(cfg),
		database:  database,
		table:     table,
		workgroup: workgroup,
		outputLoc: outputLoc,
	}
}

// NewFromAPI creates a Querier from an explicit API implementation (for testing).
func NewFromAPI(api API, database, table, workgroup, outputLoc string) *Querier {
	return &Querier{
		api:       api,
		database:  database,
		table:     table,
		workgroup: workgroup,
		outputLoc: outputLoc,
	}
}

// GetCURLineItems queries the CUR table and returns line items as []map[string]any
// matching the fixture shape with keys: line_item_line_item_type, line_item_product_code,
// line_item_usage_type, product_product_name, unblended_cost.
func (q *Querier) GetCURLineItems(accountID, startDate, endDate, service string) ([]map[string]any, error) {
	sql, err := buildCURQuery(q.table, accountID, startDate, endDate, service)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.TODO(), pollTimeout)
	defer cancel()

	// Start the query.
	startOut, err := q.api.StartQueryExecution(ctx, &ath.StartQueryExecutionInput{
		QueryString: aws.String(sql),
		QueryExecutionContext: &athtypes.QueryExecutionContext{
			Database: aws.String(q.database),
		},
		WorkGroup: aws.String(q.workgroup),
		ResultConfiguration: &athtypes.ResultConfiguration{
			OutputLocation: aws.String(q.outputLoc),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("athena: start query: %w", err)
	}

	queryID := startOut.QueryExecutionId

	// Poll until complete, respecting context cancellation.
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		execOut, err := q.api.GetQueryExecution(ctx, &ath.GetQueryExecutionInput{
			QueryExecutionId: queryID,
		})
		if err != nil {
			return nil, fmt.Errorf("athena: get query execution: %w", err)
		}

		state := execOut.QueryExecution.Status.State
		switch state {
		case athtypes.QueryExecutionStateSucceeded:
			// Proceed to get results.
		case athtypes.QueryExecutionStateFailed:
			reason := ""
			if execOut.QueryExecution.Status.StateChangeReason != nil {
				reason = *execOut.QueryExecution.Status.StateChangeReason
			}
			return nil, fmt.Errorf("athena: query failed: %s", reason)
		case athtypes.QueryExecutionStateCancelled:
			return nil, fmt.Errorf("athena: query was cancelled")
		default:
			select {
			case <-ctx.Done():
				return nil, fmt.Errorf("athena: query %s: %w", *queryID, ctx.Err())
			case <-ticker.C:
				continue
			}
		}

		// Fetch results.
		resultsOut, err := q.api.GetQueryResults(ctx, &ath.GetQueryResultsInput{
			QueryExecutionId: queryID,
		})
		if err != nil {
			return nil, fmt.Errorf("athena: get query results: %w", err)
		}

		return transformResults(resultsOut), nil
	}
}

// transformResults converts Athena ResultSet rows to []map[string]any.
// The first row is the header; remaining rows are data.
func transformResults(out *ath.GetQueryResultsOutput) []map[string]any {
	if out.ResultSet == nil || len(out.ResultSet.Rows) < 2 {
		return nil
	}

	rows := out.ResultSet.Rows
	headers := make([]string, len(rows[0].Data))
	for i, d := range rows[0].Data {
		if d.VarCharValue != nil {
			headers[i] = *d.VarCharValue
		}
	}

	// Map CUR column names to fixture-compatible keys.
	keyMap := map[string]string{
		"line_item_line_item_type": "line_item_line_item_type",
		"line_item_product_code":   "line_item_product_code",
		"line_item_usage_type":     "line_item_usage_type",
		"product_product_name":     "product_product_name",
		"line_item_unblended_cost": "unblended_cost",
	}

	items := make([]map[string]any, 0, len(rows)-1)
	for _, row := range rows[1:] {
		item := make(map[string]any)
		for i, d := range row.Data {
			if i >= len(headers) {
				break
			}
			key := headers[i]
			if mapped, ok := keyMap[key]; ok {
				key = mapped
			}

			val := ""
			if d.VarCharValue != nil {
				val = *d.VarCharValue
			}

			// Try to parse numeric values for unblended_cost.
			if key == "unblended_cost" {
				if f, err := strconv.ParseFloat(val, 64); err == nil {
					item[key] = f
					continue
				}
			}
			item[key] = val
		}
		items = append(items, item)
	}

	return items
}
