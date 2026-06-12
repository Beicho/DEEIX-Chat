package conversation

import "testing"

func TestNormalizeProjectDocumentFileIDsDeduplicatesAndRejectsEmpty(t *testing.T) {
	got, err := normalizeProjectDocumentFileIDs([]string{" file_a ", "file_b", "file_a"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 || got[0] != "file_a" || got[1] != "file_b" {
		t.Fatalf("unexpected normalized ids: %#v", got)
	}

	if _, err = normalizeProjectDocumentFileIDs([]string{" ", ""}); err == nil {
		t.Fatal("expected empty input to be rejected")
	}
}
