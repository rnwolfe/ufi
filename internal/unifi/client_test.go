package unifi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"

	"github.com/rnwolfe/ufi/internal/errs"
)

func TestToSnake(t *testing.T) {
	cases := map[string]string{
		"macAddress": "mac_address",
		"ipAddress":  "ip_address",
		"vlanId":     "vlan_id",
		"id":         "id",
		"uptimeSec":  "uptime_sec",
		"name":       "name",
	}
	for in, want := range cases {
		if got := toSnake(in); got != want {
			t.Errorf("toSnake(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestSnakeKeysNested(t *testing.T) {
	in := map[string]any{"macAddress": "x", "nested": map[string]any{"vlanId": 5}, "arr": []any{map[string]any{"ipAddress": "y"}}}
	out := snakeKeys(in).(map[string]any)
	if _, ok := out["mac_address"]; !ok {
		t.Fatalf("top-level key not snaked: %v", out)
	}
	if _, ok := out["nested"].(map[string]any)["vlan_id"]; !ok {
		t.Fatalf("nested key not snaked: %v", out["nested"])
	}
	if _, ok := out["arr"].([]any)[0].(map[string]any)["ip_address"]; !ok {
		t.Fatalf("array item key not snaked: %v", out["arr"])
	}
}

func TestCursorRoundTrip(t *testing.T) {
	for _, n := range []int{0, 1, 25, 999} {
		c := encodeCursor(n)
		got, err := decodeCursor(c)
		if err != nil || got != n {
			t.Fatalf("cursor %d round-trip: got %d err %v", n, got, err)
		}
	}
	if _, err := decodeCursor("!!!notbase64"); err == nil {
		t.Fatalf("expected error on bad cursor")
	}
}

func TestListPaginationWithCursor(t *testing.T) {
	all := []map[string]any{{"id": "1"}, {"id": "2"}, {"id": "3"}}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		end := offset + limit
		if end > len(all) {
			end = len(all)
		}
		var data []map[string]any
		if offset < len(all) {
			data = all[offset:end]
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"offset": offset, "limit": limit, "count": len(data), "totalCount": len(all), "data": data,
		})
	}))
	defer srv.Close()

	c := newClient(mustURL(srv.URL), "k", Options{})
	// First page: limit 2 → 2 items + a cursor.
	res, err := c.List(context.Background(), "/things", ListOpts{Limit: 2})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if res.Count != 2 || res.NextCursor == nil {
		t.Fatalf("page1: count=%d cursor=%v", res.Count, res.NextCursor)
	}
	// Second page via the cursor: 1 item, no further cursor.
	res2, err := c.List(context.Background(), "/things", ListOpts{Limit: 2, Cursor: res.NextCursor.(string)})
	if err != nil {
		t.Fatalf("list2: %v", err)
	}
	if res2.Count != 1 || res2.NextCursor != nil {
		t.Fatalf("page2: count=%d cursor=%v", res2.Count, res2.NextCursor)
	}
}

func TestClassifyZoneBasedFirewall(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"statusCode":400,"code":"api.firewall.zone-based-firewall-not-configured","message":"Zone Based Firewall is not configured"}`))
	}))
	defer srv.Close()

	c := newClient(mustURL(srv.URL), "k", Options{})
	_, err := c.GetObject(context.Background(), "/sites/x/firewall/policies")
	var ce *errs.CLIError
	if !errors.As(err, &ce) {
		t.Fatalf("want *errs.CLIError, got %T", err)
	}
	if ce.Exit != errs.ExitUnsupported || ce.Code != "UNSUPPORTED" {
		t.Fatalf("ZBF should map to UNSUPPORTED/11, got %s/%d", ce.Code, ce.Exit)
	}
}

func TestClassifyAuth(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"statusCode":401,"message":"unauthorized"}`))
	}))
	defer srv.Close()
	c := newClient(mustURL(srv.URL), "bad", Options{})
	_, err := c.GetObject(context.Background(), "/info")
	var ce *errs.CLIError
	if !errors.As(err, &ce) || ce.Exit != errs.ExitAuth {
		t.Fatalf("401 should map to AUTH (exit 4), got %v", err)
	}
}

func mustURL(s string) *url.URL {
	u, err := url.Parse(s)
	if err != nil {
		panic(err)
	}
	return u
}
