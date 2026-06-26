package cli

// schemaVersion is the top-level version stamped on every list envelope (spec: Output schema).
// Bump only for breaking output changes; field additions are append-only and don't bump it.
const schemaVersion = 1

// listEnvelope wraps items in the stable list envelope:
// { schemaVersion, items, count, nextCursor }. nextCursor is nil at end-of-results.
func listEnvelope(items any, count int, nextCursor any) map[string]any {
	if items == nil {
		items = []any{}
	}
	return map[string]any{
		"schemaVersion": schemaVersion,
		"items":         items,
		"count":         count,
		"nextCursor":    nextCursor,
	}
}
