package transport

import (
	"fmt"
	"net/url"
	"strconv"
)

// ListQuery bounds transport inputs. Domain repositories must bind cursor/filter
// values to trusted scope and use parameterized SQL, never concatenate user SQL.
type ListQuery struct {
	Limit     int
	Cursor    string
	Sort      string
	Direction string
	Filters   map[string]string
}

func ParseListQuery(values url.Values, sorts, filters []string) (ListQuery, error) {
	result := ListQuery{Limit: 25, Direction: "asc", Filters: map[string]string{}}
	allowed := map[string]bool{"limit": true, "cursor": true, "sort": true, "direction": true}
	for _, key := range filters {
		allowed[key] = true
	}
	for key, items := range values {
		if !allowed[key] || len(items) != 1 {
			return result, fmt.Errorf("unsupported or repeated query field")
		}
	}
	if value := values.Get("limit"); value != "" {
		n, err := strconv.Atoi(value)
		if err != nil || n < 1 || n > 100 {
			return result, fmt.Errorf("limit must be between 1 and 100")
		}
		result.Limit = n
	}
	result.Cursor = values.Get("cursor")
	if len(result.Cursor) > 4096 {
		return result, fmt.Errorf("invalid cursor")
	}
	result.Sort = values.Get("sort")
	if result.Sort != "" {
		ok := false
		for _, key := range sorts {
			if key == result.Sort {
				ok = true
			}
		}
		if !ok {
			return result, fmt.Errorf("unsupported sort")
		}
	}
	if value := values.Get("direction"); value != "" {
		if value != "asc" && value != "desc" {
			return result, fmt.Errorf("unsupported direction")
		}
		result.Direction = value
	}
	for _, key := range filters {
		if values.Has(key) {
			result.Filters[key] = values.Get(key)
		}
	}
	return result, nil
}
