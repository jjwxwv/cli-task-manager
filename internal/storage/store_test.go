package storage_test

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"pomotask/internal/storage"
	"pomotask/internal/task"
)

// dataPath returns a path inside a fresh temporary directory. Every test in
// this file works there and none can reach the user's real task data, which is
// what the Constitution requires of persistence tests.
func dataPath(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "tasks.json")
}

// T014 — what was saved is what loads back (FR-004).
func TestSaveLoadRoundTrip(t *testing.T) {
	path := dataPath(t)
	want := []task.Task{
		{ID: 1, Text: "write the report", Done: false},
		{ID: 2, Text: "review the draft", Done: true},
	}

	if err := storage.Save(path, want); err != nil {
		t.Fatalf("Save returned an unexpected error: %v", err)
	}

	got, err := storage.Load(path)
	if err != nil {
		t.Fatalf("Load returned an unexpected error: %v", err)
	}
	if len(got) != len(want) {
		t.Fatalf("Load returned %d tasks, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("task %d = %+v, want %+v", i, got[i], want[i])
		}
	}
}

// T014 — an absent file is an empty task set, not an error. First use is
// ordinary and must never be confused with unreadable data (spec first-use edge
// case, SC-006).
func TestLoadAbsentFile(t *testing.T) {
	got, err := storage.Load(dataPath(t))
	if err != nil {
		t.Fatalf("Load of an absent file returned an error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("Load of an absent file returned %d tasks, want 0", len(got))
	}
}

// T014 — a save into a directory holding no data file succeeds with no prior
// initialization step of any kind (SC-006).
func TestSaveWithNoPriorInitialization(t *testing.T) {
	path := dataPath(t)

	if err := storage.Save(path, []task.Task{{ID: 1, Text: "write the report"}}); err != nil {
		t.Fatalf("Save into a directory with no data file returned an error: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("the data file was not created: %v", err)
	}
}

// T014 — saving an empty collection is not an error and loads back empty.
func TestSaveEmptyCollection(t *testing.T) {
	path := dataPath(t)

	if err := storage.Save(path, nil); err != nil {
		t.Fatalf("Save of an empty collection returned an error: %v", err)
	}
	got, err := storage.Load(path)
	if err != nil {
		t.Fatalf("Load returned an unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("Load returned %d tasks, want 0", len(got))
	}
}

// T015 — an unrecognized schema_version is a load-time failure naming both the
// version found and the version supported, and the file is left as found
// (ADR 0001).
func TestLoadRejectsUnsupportedSchemaVersion(t *testing.T) {
	path := dataPath(t)
	original := []byte(`{"schema_version": 2, "tasks": [{"id": 1, "text": "a", "done": false}]}`)
	writeFile(t, path, original)

	_, err := storage.Load(path)
	if err == nil {
		t.Fatal("Load accepted schema_version 2, want a failure")
	}
	message := err.Error()
	for _, want := range []string{"2", strconv.Itoa(storage.SchemaVersion)} {
		if !strings.Contains(message, want) {
			t.Errorf("error %q does not name %q; it must state both the version found and the version supported", message, want)
		}
	}

	assertFileUnchanged(t, path, original)
}

// T015 — a truncated document fails and the file is left as found
// (research.md R2).
func TestLoadRejectsTruncatedDocument(t *testing.T) {
	path := dataPath(t)
	original := []byte(`{"schema_version": 1, "tasks": [{"id": 1, "text": "wri`)
	writeFile(t, path, original)

	got, err := storage.Load(path)
	if err == nil {
		t.Fatalf("Load accepted a truncated document and returned %v, want a failure", got)
	}
	if got != nil {
		t.Errorf("Load returned %v alongside its error, want nil", got)
	}

	assertFileUnchanged(t, path, original)
}

// T015 — a document that is not JSON at all fails the same way, rather than
// being mistaken for an absent file.
func TestLoadRejectsNonJSON(t *testing.T) {
	path := dataPath(t)
	original := []byte("this is not JSON\n")
	writeFile(t, path, original)

	if _, err := storage.Load(path); err == nil {
		t.Fatal("Load accepted a non-JSON file, want a failure")
	}

	assertFileUnchanged(t, path, original)
}

// T016 — the write path leaves no stray temporary file behind and the
// destination survives (ADR 0001 write mandate; research.md R5).
func TestSaveLeavesNoTemporaryFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tasks.json")

	if err := storage.Save(path, []task.Task{{ID: 1, Text: "write the report"}}); err != nil {
		t.Fatalf("Save returned an unexpected error: %v", err)
	}
	assertOnlyDataFile(t, dir)

	// A second save replaces the destination and again leaves nothing behind.
	if err := storage.Save(path, []task.Task{{ID: 1, Text: "write the report", Done: true}}); err != nil {
		t.Fatalf("second Save returned an unexpected error: %v", err)
	}
	assertOnlyDataFile(t, dir)

	got, err := storage.Load(path)
	if err != nil {
		t.Fatalf("Load after two saves returned an error: %v", err)
	}
	if len(got) != 1 || !got[0].Done {
		t.Errorf("the destination did not survive the second save: %v", got)
	}
}

// T016 — a save that cannot complete leaves no temporary file behind either.
// Renaming onto a directory fails on every platform, which is what makes this
// portable.
func TestFailedSaveLeavesNoTemporaryFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tasks.json")
	if err := os.Mkdir(path, 0o755); err != nil {
		t.Fatalf("preparing the unwritable destination: %v", err)
	}

	if err := storage.Save(path, []task.Task{{ID: 1, Text: "write the report"}}); err == nil {
		t.Fatal("Save onto a directory succeeded, want a failure")
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("reading the destination directory: %v", err)
	}
	for _, entry := range entries {
		if entry.Name() != "tasks.json" {
			t.Errorf("a stray file was left behind after a failed save: %s", entry.Name())
		}
	}
}

func writeFile(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("writing the fixture at %s: %v", path, err)
	}
}

func assertFileUnchanged(t *testing.T, path string, original []byte) {
	t.Helper()
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("the data file could not be read back: %v", err)
	}
	if string(after) != string(original) {
		t.Errorf("the data file changed:\n before: %q\n  after: %q", original, after)
	}
}

func assertOnlyDataFile(t *testing.T, dir string) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("reading the destination directory: %v", err)
	}
	if len(entries) != 1 || entries[0].Name() != "tasks.json" {
		names := make([]string, 0, len(entries))
		for _, entry := range entries {
			names = append(names, entry.Name())
		}
		t.Errorf("destination directory holds %v, want only tasks.json", names)
	}
}
