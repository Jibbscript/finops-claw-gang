package analysis

// CostQuerier provides cost data needed by the analysis planner.
type CostQuerier interface {
	GetCURLineItems(accountID, startDate, endDate string, service string) ([]map[string]any, error)
}
