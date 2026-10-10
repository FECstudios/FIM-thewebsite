package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const basePage = "<main>\n<h1>FIM</h1>\n<p>Made for your toughest tasks.</p>\n<footer>Nort Labs</footer>\n</main>"

func TestParseEdits(t *testing.T) {
	reply := "<<<<<<< FIND\n<h1>FIM</h1>\n=======\n<h1>FIM 2</h1>\n>>>>>>> REPLACE\n" +
		"<<<<<<< FIND\r\n<footer>Nort Labs</footer>\r\n=======\r\n>>>>>>> REPLACE"
	edits, found := parseEdits(reply)
	if !found || len(edits) != 2 {
		t.Fatalf("expected 2 edits, got %d (found=%v)", len(edits), found)
	}
	if edits[0].Find != "<h1>FIM</h1>" || edits[0].Replace != "<h1>FIM 2</h1>" {
		t.Fatalf("bad first edit: %+v", edits[0])
	}
	if edits[1].Replace != "" {
		t.Fatalf("expected empty replace for delete, got %q", edits[1].Replace)
	}

	if _, found := parseEdits("<main>full page</main>"); found {
		t.Fatal("full page must not be treated as edits")
	}
}

func TestApplyEdits(t *testing.T) {
	t.Run("replace, insert and delete", func(t *testing.T) {
		got, err := applyEdits(basePage, []edit{
			{Find: "<h1>FIM</h1>", Replace: "<h1>Nort Labs FIM</h1>"},
			{Find: "<p>Made for your toughest tasks.</p>", Replace: "<p>Made for your toughest tasks.</p>\n<section>Pricing</section>"},
			{Find: "<footer>Nort Labs</footer>\n", Replace: ""},
		})
		if err != nil {
			t.Fatal(err)
		}
		want := "<main>\n<h1>Nort Labs FIM</h1>\n<p>Made for your toughest tasks.</p>\n<section>Pricing</section>\n</main>"
		if got != want {
			t.Fatalf("got:\n%s\nwant:\n%s", got, want)
		}
	})
	t.Run("tolerates surrounding whitespace", func(t *testing.T) {
		if _, err := applyEdits(basePage, []edit{{Find: "  <h1>FIM</h1>  ", Replace: "<h1>X</h1>"}}); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("rejects bad edits", func(t *testing.T) {
		cases := map[string][]edit{
			"no edits":    nil,
			"not found":   {{Find: "<h2>nope</h2>", Replace: "x"}},
			"ambiguous":   {{Find: "FIM", Replace: "x"}},
			"empty find":  {{Find: "  ", Replace: "x"}},
			"breaks main": {{Find: "</main>", Replace: ""}},
		}
		for name, edits := range cases {
			p := basePage
			if name == "ambiguous" {
				// basePage has "FIM" once; add a second one so FIND is ambiguous.
				p = strings.Replace(basePage, "Nort Labs", "FIM Labs", 1)
			}
			if _, err := applyEdits(p, edits); err == nil {
				t.Errorf("%s: expected error", name)
			}
		}
	})
}

// fakeModel points chatURL at a server that returns the given replies in order
// and records the user messages it received.
func fakeModel(t *testing.T, replies ...string) *[]string {
	t.Helper()
	var got []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req chatRequest
		json.NewDecoder(r.Body).Decode(&req)
		got = append(got, req.Messages[len(req.Messages)-1].Content)
		reply := replies[len(got)-1]
		json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{{"message": map[string]string{"role": "assistant", "content": reply}}},
		})
	}))
	t.Cleanup(srv.Close)
	old := chatURL
	chatURL = srv.URL
	t.Cleanup(func() { chatURL = old })
	t.Setenv("HACKCLUB_AI_KEY", "test-key")
	return &got
}

func TestGenerateSectionAppliesEdits(t *testing.T) {
	calls := fakeModel(t, "<<<<<<< FIND\n<h1>FIM</h1>\n=======\n<h1>FIM 2</h1>\n>>>>>>> REPLACE")
	page, err := generateSection(context.Background(), "rename", basePage)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(page, "<h1>FIM 2</h1>") || !strings.Contains(page, "<footer>Nort Labs</footer>") {
		t.Fatalf("edit not applied or page lost: %s", page)
	}
	if len(*calls) != 1 {
		t.Fatalf("expected 1 model call, got %d", len(*calls))
	}
}

func TestGenerateSectionFallsBackToFullPage(t *testing.T) {
	calls := fakeModel(t,
		"<<<<<<< FIND\n<h1>does not exist</h1>\n=======\n<h1>x</h1>\n>>>>>>> REPLACE",
		"<main><h1>Rewritten</h1></main>",
	)
	page, err := generateSection(context.Background(), "rename", basePage)
	if err != nil {
		t.Fatal(err)
	}
	if page != "<main><h1>Rewritten</h1></main>" {
		t.Fatalf("expected full rewrite, got %s", page)
	}
	if len(*calls) != 2 || !strings.Contains((*calls)[1], "complete updated page") {
		t.Fatalf("expected a second call asking for the full page, got %d calls", len(*calls))
	}
}

func TestGenerateSectionEmptyPageWantsFullPage(t *testing.T) {
	fakeModel(t, "<main><h1>Hero</h1></main>")
	page, err := generateSection(context.Background(), "start", "")
	if err != nil || page != "<main><h1>Hero</h1></main>" {
		t.Fatalf("got %q %v", page, err)
	}
}
