// Package connectors provides composite adapters that compose individual AWS
// service clients to satisfy the union interfaces consumed by activities.
package connectors

import (
	"github.com/aws/aws-sdk-go-v2/aws"

	"github.com/finops-claw-gang/finops-go/internal/connectors/aws/athena"
	"github.com/finops-claw-gang/finops-go/internal/connectors/aws/cloudwatch"
	"github.com/finops-claw-gang/finops-go/internal/connectors/aws/codedeploy"
	"github.com/finops-claw-gang/finops-go/internal/connectors/aws/costexplorer"
	"github.com/finops-claw-gang/finops-go/internal/connectors/aws/tagging"
)

// AWSCostClient satisfies activities.CostDeps by composing Cost Explorer and Athena clients.
// Methods:
//   - GetRICoverage, GetSPCoverage, GetCostTimeseries -> Cost Explorer
//   - GetCURLineItems -> Athena
type AWSCostClient struct {
	ce  *costexplorer.Client
	ath *athena.Querier
}

// NewAWSCostClient creates an AWSCostClient from an AWS config and Athena CUR configuration.
func NewAWSCostClient(cfg aws.Config, curDatabase, curTable, curWorkgroup, curOutputBucket string) *AWSCostClient {
	return &AWSCostClient{
		ce:  costexplorer.New(cfg),
		ath: athena.New(cfg, curDatabase, curTable, curWorkgroup, curOutputBucket),
	}
}

func (c *AWSCostClient) GetRICoverage(accountID, startDate, endDate string) (map[string]any, error) {
	return c.ce.GetRICoverage(accountID, startDate, endDate)
}

func (c *AWSCostClient) GetSPCoverage(accountID, startDate, endDate string) (map[string]any, error) {
	return c.ce.GetSPCoverage(accountID, startDate, endDate)
}

func (c *AWSCostClient) GetCostTimeseries(service, accountID, startDate, endDate string) (map[string]any, error) {
	return c.ce.GetCostTimeseries(service, accountID, startDate, endDate)
}

func (c *AWSCostClient) GetCURLineItems(accountID, startDate, endDate, service string) ([]map[string]any, error) {
	return c.ath.GetCURLineItems(accountID, startDate, endDate, service)
}

// AWSInfraClient satisfies activities.InfraDeps by composing CloudWatch, Tagging, and CodeDeploy clients.
// Methods:
//   - CloudWatchMetrics -> CloudWatch
//   - ResourceTags -> Tagging
//   - RecentDeploys -> CodeDeploy
type AWSInfraClient struct {
	cw *cloudwatch.Client
	tg *tagging.Client
	cd *codedeploy.Client
}

// NewAWSInfraClient creates an AWSInfraClient from an AWS config.
func NewAWSInfraClient(cfg aws.Config) *AWSInfraClient {
	return &AWSInfraClient{
		cw: cloudwatch.New(cfg),
		tg: tagging.New(cfg),
		cd: codedeploy.New(cfg),
	}
}

func (c *AWSInfraClient) RecentDeploys(service string) ([]map[string]any, error) {
	return c.cd.RecentDeploys(service)
}

func (c *AWSInfraClient) CloudWatchMetrics(resourceID, metricName, namespace string) (map[string]any, error) {
	return c.cw.CloudWatchMetrics(resourceID, metricName, namespace)
}

func (c *AWSInfraClient) ResourceTags(resourceARN string) (map[string]string, error) {
	return c.tg.ResourceTags(resourceARN)
}
