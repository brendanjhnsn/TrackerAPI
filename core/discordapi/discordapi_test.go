package discordapi_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sort"
	"testing"

	"github.com/brendanjhnsn/TrackerAPI/core/discordapi"
)

func makeServer(t *testing.T, pages [][]map[string]any) *httptest.Server {
	t.Helper()
	pageIdx := 0
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") == "" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if pageIdx >= len(pages) {
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode([]map[string]any{})
			return
		}
		_ = json.NewEncoder(w).Encode(pages[pageIdx])
		pageIdx++
	}))
}

func member(id, roleID string) map[string]any {
	return map[string]any{
		"user":  map[string]any{"id": id},
		"roles": []string{roleID},
	}
}

func memberNoRole(id string) map[string]any {
	return map[string]any{
		"user":  map[string]any{"id": id},
		"roles": []string{"other-role"},
	}
}

func TestListMembersWithRole_ReturnsOnlyMatchingRole(t *testing.T) {
	page := []map[string]any{
		member("111", "target-role"),
		memberNoRole("222"),
		member("333", "target-role"),
	}
	srv := makeServer(t, [][]map[string]any{page})
	defer srv.Close()

	got, err := discordapi.ListMembersWithRole(
		context.Background(),
		srv.Client(),
		srv.URL,
		"bot-token",
		"guild123",
		"target-role",
		100,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	sort.Strings(got)
	want := []string{"111", "333"}
	if len(got) != len(want) {
		t.Fatalf("want %v, got %v", want, got)
	}
	for i, v := range want {
		if got[i] != v {
			t.Errorf("want %v, got %v", want, got)
		}
	}
}

func TestListMembersWithRole_PaginatesUntilShortPage(t *testing.T) {
	// pageSize=2: first page returns exactly 2 (full page) → fetch again
	// second page returns 1 (short) → stop
	page1 := []map[string]any{member("aaa", "r"), member("bbb", "r")}
	page2 := []map[string]any{member("ccc", "r")}
	srv := makeServer(t, [][]map[string]any{page1, page2})
	defer srv.Close()

	got, err := discordapi.ListMembersWithRole(
		context.Background(),
		srv.Client(),
		srv.URL,
		"tok",
		"g1",
		"r",
		2, // pageSize triggers pagination at 2
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 3 {
		t.Errorf("want 3 results, got %d: %v", len(got), got)
	}
}

func TestListMembersWithRole_EmptyGuild_ReturnsEmpty(t *testing.T) {
	srv := makeServer(t, [][]map[string]any{{}})
	defer srv.Close()

	got, err := discordapi.ListMembersWithRole(
		context.Background(),
		srv.Client(),
		srv.URL,
		"tok",
		"g1",
		"r",
		1000,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("want empty, got %v", got)
	}
}
