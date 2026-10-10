package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestResolveParentUsesLatestWhenRequestedParentIsZero(t *testing.T) {
	store := &Store{
		sections: []Section{
			{ID: 1, ParentID: 0, Prompt: "first", HTML: "first-html"},
			{ID: 2, ParentID: 1, Prompt: "second", HTML: "second-html"},
		},
		nextID: 3,
	}

	parentID, previous, err := resolveParent(store, 0)
	if err != nil {
		t.Fatalf("resolveParent returned unexpected error: %v", err)
	}
	if parentID != 2 {
		t.Fatalf("expected latest section as parent, got %d", parentID)
	}
	if previous != "second-html" {
		t.Fatalf("expected previous HTML from latest section, got %q", previous)
	}
}

func TestResolveParentUsesExplicitParent(t *testing.T) {
	store := &Store{
		sections: []Section{
			{ID: 1, ParentID: 0, Prompt: "first", HTML: "first-html"},
			{ID: 2, ParentID: 1, Prompt: "second", HTML: "second-html"},
		},
		nextID: 3,
	}

	parentID, previous, err := resolveParent(store, 1)
	if err != nil {
		t.Fatalf("resolveParent returned unexpected error: %v", err)
	}
	if parentID != 1 {
		t.Fatalf("expected requested parent id 1, got %d", parentID)
	}
	if previous != "first-html" {
		t.Fatalf("expected previous HTML from explicit parent, got %q", previous)
	}
}

func TestResolveParentRejectsUnknownParent(t *testing.T) {
	store := &Store{
		sections: []Section{{ID: 1, ParentID: 0, Prompt: "first", HTML: "first-html"}},
		nextID:   2,
	}

	if _, _, err := resolveParent(store, 99); err == nil {
		t.Fatal("expected error for missing parent")
	}
}

func TestResourcesAllowNullOriginFonts(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	serveResources(r)

	req := httptest.NewRequest(http.MethodGet, "/resources/fonts/geom.ttf", nil)
	req.Header.Set("Origin", "null")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Fatalf("expected CORS header *, got %q", got)
	}
}
