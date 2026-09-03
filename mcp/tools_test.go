package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
)

// relationsFixture is the TF-438 repro: two epics claim the same child, so the
// relation type alone does not identify which link to remove.
var relationsFixture = []Relation{
	{
		ID:              "0198f1a0-1111-7000-8000-000000000001",
		SourceDisplayID: "TF-423",
		SourceTitle:     "Old epic",
		TargetDisplayID: "TF-141",
		TargetTitle:     "Child",
		RelationType:    "parent_of",
	},
	{
		ID:              "0198f1a0-2222-7000-8000-000000000002",
		SourceDisplayID: "TF-493",
		SourceTitle:     "New epic",
		TargetDisplayID: "TF-141",
		TargetTitle:     "Child",
		RelationType:    "parent_of",
	},
}

// relationServer serves the relation endpoints for one work item and records the
// path of every DELETE it receives.
type relationServer struct {
	*httptest.Server
	deleted []string
	listed  int
}

func newRelationServer(t *testing.T, projectKey string, itemNumber int, relations []Relation) *relationServer {
	t.Helper()
	rs := &relationServer{}
	prefix := "/api/v1/default/projects/" + projectKey + "/items/" + strconv.Itoa(itemNumber) + "/relations"
	mux := http.NewServeMux()
	mux.HandleFunc(prefix, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		rs.listed++
		writeData(t, w, relations)
	})
	mux.HandleFunc(prefix+"/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		rs.deleted = append(rs.deleted, strings.TrimPrefix(r.URL.Path, prefix+"/"))
		writeData(t, w, map[string]string{})
	})
	rs.Server = httptest.NewServer(mux)
	t.Cleanup(rs.Close)
	return rs
}

func writeData(t *testing.T, w http.ResponseWriter, payload any) {
	t.Helper()
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	w.Header().Set("Content-Type", "application/json")
	if _, err := w.Write([]byte(`{"data":` + string(data) + `}`)); err != nil {
		t.Fatalf("write payload: %v", err)
	}
}

// useServer points the tools at srv and pins the namespace so getClientForDisplayID
// resolves without a search round trip.
func useServer(t *testing.T, srv *relationServer) {
	t.Helper()
	t.Setenv("TASKWONDO_URL", srv.URL)
	t.Setenv("TASKWONDO_API_KEY", "twk_test")
	previous := activeNamespace
	activeNamespace = "default"
	t.Cleanup(func() { activeNamespace = previous })
}

func toolRequest(args map[string]any) mcp.CallToolRequest {
	var req mcp.CallToolRequest
	req.Params.Arguments = args
	return req
}

func resultText(t *testing.T, result *mcp.CallToolResult) string {
	t.Helper()
	var sb strings.Builder
	for _, c := range result.Content {
		text, ok := c.(mcp.TextContent)
		if !ok {
			t.Fatalf("unexpected content type %T", c)
		}
		sb.WriteString(text.Text)
	}
	return sb.String()
}

func TestListRelationsShowsRelationIDs(t *testing.T) {
	srv := newRelationServer(t, "TF", 141, relationsFixture)
	useServer(t, srv)

	result, err := handleListRelations(context.Background(), toolRequest(map[string]any{"display_id": "TF-141"}))
	if err != nil {
		t.Fatalf("handleListRelations: %v", err)
	}
	text := resultText(t, result)
	for _, r := range relationsFixture {
		if !strings.Contains(text, r.ID) {
			t.Errorf("relation %s missing its id in output:\n%s", r.SourceDisplayID, text)
		}
	}
}

func TestDeleteRelationByID(t *testing.T) {
	srv := newRelationServer(t, "TF", 141, relationsFixture)
	useServer(t, srv)

	result, err := handleDeleteRelation(context.Background(), toolRequest(map[string]any{
		"display_id":  "TF-141",
		"relation_id": relationsFixture[0].ID,
	}))
	if err != nil {
		t.Fatalf("handleDeleteRelation: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected error result: %s", resultText(t, result))
	}
	if len(srv.deleted) != 1 || srv.deleted[0] != relationsFixture[0].ID {
		t.Fatalf("deleted %v, want [%s]", srv.deleted, relationsFixture[0].ID)
	}
	if srv.listed != 0 {
		t.Errorf("listed relations %d times, want 0 when an id is given", srv.listed)
	}
}

func TestDeleteRelationByTriple(t *testing.T) {
	srv := newRelationServer(t, "TF", 423, relationsFixture)
	useServer(t, srv)

	result, err := handleDeleteRelation(context.Background(), toolRequest(map[string]any{
		"display_id":        "TF-423",
		"relation_type":     "parent_of",
		"target_display_id": "TF-141",
	}))
	if err != nil {
		t.Fatalf("handleDeleteRelation: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected error result: %s", resultText(t, result))
	}
	if len(srv.deleted) != 1 || srv.deleted[0] != relationsFixture[0].ID {
		t.Fatalf("deleted %v, want the TF-423 link [%s]", srv.deleted, relationsFixture[0].ID)
	}
}

func TestDeleteRelationTripleIgnoresReverseDirection(t *testing.T) {
	srv := newRelationServer(t, "TF", 141, relationsFixture)
	useServer(t, srv)

	// TF-141 is the target of both parent_of links, never the source.
	result, err := handleDeleteRelation(context.Background(), toolRequest(map[string]any{
		"display_id":        "TF-141",
		"relation_type":     "parent_of",
		"target_display_id": "TF-423",
	}))
	if err != nil {
		t.Fatalf("handleDeleteRelation: %v", err)
	}
	if !result.IsError {
		t.Fatalf("expected an error result, got: %s", resultText(t, result))
	}
	text := resultText(t, result)
	if !strings.Contains(text, relationsFixture[0].ID) {
		t.Errorf("error should list the relations that do exist, got:\n%s", text)
	}
	if len(srv.deleted) != 0 {
		t.Fatalf("deleted %v, want nothing", srv.deleted)
	}
}

func TestDeleteRelationTripleUnrelatedTarget(t *testing.T) {
	srv := newRelationServer(t, "TF", 423, relationsFixture)
	useServer(t, srv)

	result, err := handleDeleteRelation(context.Background(), toolRequest(map[string]any{
		"display_id":        "TF-423",
		"relation_type":     "parent_of",
		"target_display_id": "TF-999",
	}))
	if err != nil {
		t.Fatalf("handleDeleteRelation: %v", err)
	}
	if !result.IsError {
		t.Fatalf("expected an error result, got: %s", resultText(t, result))
	}
	if !strings.Contains(resultText(t, result), "no relation between TF-423 and TF-999") {
		t.Errorf("unexpected error text:\n%s", resultText(t, result))
	}
	if len(srv.deleted) != 0 {
		t.Fatalf("deleted %v, want nothing", srv.deleted)
	}
}

func TestDeleteRelationNeedsAnIdentifier(t *testing.T) {
	srv := newRelationServer(t, "TF", 141, relationsFixture)
	useServer(t, srv)

	result, err := handleDeleteRelation(context.Background(), toolRequest(map[string]any{
		"display_id":    "TF-141",
		"relation_type": "parent_of",
	}))
	if err != nil {
		t.Fatalf("handleDeleteRelation: %v", err)
	}
	if !result.IsError {
		t.Fatalf("expected an error result, got: %s", resultText(t, result))
	}
	if srv.listed != 0 || len(srv.deleted) != 0 {
		t.Fatalf("expected no API calls, listed=%d deleted=%v", srv.listed, srv.deleted)
	}
}
