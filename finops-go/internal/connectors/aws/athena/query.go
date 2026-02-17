package athena

import (
	"fmt"
	"regexp"
)

var (
	accountIDPattern = regexp.MustCompile(`^\d{12}$`)
	datePattern      = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
	servicePattern   = regexp.MustCompile(`^[a-zA-Z0-9 _-]+$`)
	tablePattern     = regexp.MustCompile(`^[a-zA-Z0-9_.]+$`)
)

// buildCURQuery constructs a SQL query for the CUR table with validated inputs.
// Returns an error if any input fails validation (preventing SQL injection).
func buildCURQuery(table, accountID, startDate, endDate, service string) (string, error) {
	if !accountIDPattern.MatchString(accountID) {
		return "", fmt.Errorf("athena query: invalid account ID %q (must be 12 digits)", accountID)
	}
	if !datePattern.MatchString(startDate) {
		return "", fmt.Errorf("athena query: invalid start date %q (must be YYYY-MM-DD)", startDate)
	}
	if !datePattern.MatchString(endDate) {
		return "", fmt.Errorf("athena query: invalid end date %q (must be YYYY-MM-DD)", endDate)
	}
	if !servicePattern.MatchString(service) {
		return "", fmt.Errorf("athena query: invalid service %q (must be alphanumeric)", service)
	}
	if !tablePattern.MatchString(table) {
		return "", fmt.Errorf("athena query: invalid table name %q (must be alphanumeric, dots, underscores)", table)
	}

	query := fmt.Sprintf(
		`SELECT line_item_line_item_type, line_item_product_code, line_item_usage_type, product_product_name, line_item_unblended_cost
FROM %s
WHERE line_item_usage_account_id = '%s'
  AND line_item_usage_start_date >= TIMESTAMP '%s'
  AND line_item_usage_start_date < TIMESTAMP '%s'
  AND product_product_name = '%s'
ORDER BY line_item_unblended_cost DESC
LIMIT 1000`,
		table, accountID, startDate, endDate, service,
	)
	return query, nil
}
