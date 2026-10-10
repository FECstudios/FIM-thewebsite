package main

import "testing"

func TestExtractMainElement(t *testing.T) {
	t.Run("salvages leading and trailing text", func(t *testing.T) {
		in := "Sure — here is the page:\n<main class=\"hero\">Hi</main>\nThanks!"
		got, ok := extractMainElement(in)
		if !ok {
			t.Fatal("expected to extract a main element")
		}
		if got != "<main class=\"hero\">Hi</main>" {
			t.Fatalf("unexpected extracted output: %q", got)
		}
	})

	t.Run("rejects missing closing tag", func(t *testing.T) {
		if _, ok := extractMainElement("<main><div>oops</div>"); ok {
			t.Fatal("expected missing closing tag to be rejected")
		}
	})

	t.Run("rejects no main tag at all", func(t *testing.T) {
		if _, ok := extractMainElement("<div>nope</div>"); ok {
			t.Fatal("expected non-main output to be rejected")
		}
	})
}
