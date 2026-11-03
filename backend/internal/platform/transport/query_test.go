package transport

import (
	"net/url"
	"testing"
)

func TestBoundedListQuery(t *testing.T) {
	for _, raw := range []string{"limit=101", "limit=0", "sort=password", "direction=sideways", "unknown=1", "limit=1&limit=2"} {
		values, _ := url.ParseQuery(raw)
		if _, err := ParseListQuery(values, []string{"name"}, []string{"q"}); err == nil {
			t.Fatalf("accepted %s", raw)
		}
	}
	values, _ := url.ParseQuery("limit=100&sort=name&direction=desc&q=site")
	q, err := ParseListQuery(values, []string{"name"}, []string{"q"})
	if err != nil || q.Limit != 100 || q.Filters["q"] != "site" {
		t.Fatalf("query %v: %v", q, err)
	}
}
