package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"pomotask/internal/task"
)

// SchemaVersion is the version of the persisted document format this build
// reads and writes. ADR 0001 requires that the stored value be read and checked
// before task records are decoded, and that an unrecognized value be a
// load-time failure rather than something to ignore.
const SchemaVersion = 1

// record is the persisted shape of one task. It is deliberately separate from
// task.Task: tagging the domain type would keep the serialized format inside
// the domain, which is what ADR 0001's prohibition exists to prevent
// (research.md R9).
type record struct {
	ID   int    `json:"id"`
	Text string `json:"text"`
	Done bool   `json:"done"`
}

// document is the persisted shape of the whole data file. ADR 0001 mandates
// both fields.
type document struct {
	SchemaVersion int      `json:"schema_version"`
	Tasks         []record `json:"tasks"`
}

// versionProbe carries only the field that must be read before anything else in
// the document is decoded.
type versionProbe struct {
	SchemaVersion int `json:"schema_version"`
}

// Load reads the tasks stored at path.
//
// An absent file yields an empty task set and no error: first use is ordinary
// and must never be confused with unreadable data. A file whose schema_version
// this build does not support, or that cannot be decoded, is reported as an
// error naming what was wrong. In every failure case the file is left exactly
// as found — nothing is moved, truncated, or rewritten (ADR 0001;
// research.md R2, R4).
func Load(path string) ([]task.Task, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading task data from %s: %w", path, err)
	}

	// The version is checked before the records are decoded, in the order
	// ADR 0001 mandates. Both passes read bytes already in memory.
	var probe versionProbe
	if err := json.Unmarshal(data, &probe); err != nil {
		return nil, fmt.Errorf("task data in %s could not be read: %w", path, err)
	}
	if probe.SchemaVersion != SchemaVersion {
		return nil, fmt.Errorf(
			"task data in %s is schema version %d; this build supports schema version %d",
			path, probe.SchemaVersion, SchemaVersion)
	}

	var doc document
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("decoding task data in %s: %w", path, err)
	}

	tasks := make([]task.Task, 0, len(doc.Tasks))
	for _, r := range doc.Tasks {
		tasks = append(tasks, task.Task{ID: r.ID, Text: r.Text, Done: r.Done})
	}
	return tasks, nil
}

// Save writes tasks to path, replacing whatever is there.
//
// ADR 0001 mandates the mechanism: serialize into a temporary file created by
// os.CreateTemp in the destination's own directory, then replace the
// destination with os.Rename. Keeping both files in one directory avoids a
// cross-filesystem rename. No failure leaves a temporary file behind.
//
// The destination's directory is created if it does not exist, so that a
// first-time user needs no setup step beyond running the command (SC-006).
func Save(path string, tasks []task.Task) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating the directory for %s: %w", path, err)
	}

	records := make([]record, 0, len(tasks))
	for _, t := range tasks {
		records = append(records, record{ID: t.ID, Text: t.Text, Done: t.Done})
	}

	data, err := json.MarshalIndent(document{SchemaVersion: SchemaVersion, Tasks: records}, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding task data for %s: %w", path, err)
	}
	data = append(data, '\n')

	tmp, err := os.CreateTemp(dir, "tasks-*.json")
	if err != nil {
		return fmt.Errorf("creating a temporary file in %s: %w", dir, err)
	}
	tmpName := tmp.Name()

	// Every failure below discards the partial temporary file, and each of
	// those cleanups is deliberately best-effort. The failure already in hand
	// is what the caller needs to hear; returning a failure to tidy up in its
	// place would replace the useful message with the less useful one, and
	// would say nothing about the write that actually went wrong.
	discardTemp := func() {
		_ = os.Remove(tmpName)
	}

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		discardTemp()
		return fmt.Errorf("writing task data to %s: %w", tmpName, err)
	}
	if err := tmp.Close(); err != nil {
		discardTemp()
		return fmt.Errorf("closing %s: %w", tmpName, err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		discardTemp()
		return fmt.Errorf("replacing %s with the updated task data: %w", path, err)
	}
	return nil
}
