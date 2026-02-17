package executor

// TagFetcher provides resource tags for safety checks.
type TagFetcher interface {
	ResourceTags(resourceARN string) (map[string]string, error)
}
