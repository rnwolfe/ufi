package unifi

import "strings"

// snakeKeys recursively rewrites every JSON object key from the UniFi API's camelCase to
// snake_case, so ufi's output is a stable, fleet-consistent snake_case contract regardless of
// upstream casing (spec: "Wire vs. output naming"). Values are untouched; arrays recurse.
func snakeKeys(v any) any {
	switch t := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(t))
		for k, val := range t {
			out[toSnake(k)] = snakeKeys(val)
		}
		return out
	case []any:
		for i := range t {
			t[i] = snakeKeys(t[i])
		}
		return t
	default:
		return v
	}
}

// toSnake converts a camelCase/PascalCase identifier to snake_case, inserting a separator at
// lower→upper boundaries and around acronym runs (ipAddress→ip_address, vlanId→vlan_id).
func toSnake(s string) string {
	var b strings.Builder
	rs := []rune(s)
	for i, r := range rs {
		if r >= 'A' && r <= 'Z' {
			prevLower := i > 0 && isLowerOrDigit(rs[i-1])
			nextLower := i+1 < len(rs) && rs[i+1] >= 'a' && rs[i+1] <= 'z'
			if i > 0 && (prevLower || nextLower) {
				b.WriteByte('_')
			}
			b.WriteRune(r - 'A' + 'a')
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func isLowerOrDigit(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
}
