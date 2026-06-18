package session

import (
	"os"
	"path/filepath"
	"time"
)

const idLayout = "20060102T150405.000000"

// ID returns a session id derived from t, stable for a given t.
func ID(t time.Time) string {
	return t.Format(idLayout)
}

// Open ensures dir exists and opens dir/<ID(now)>.jsonl unbuffered for writing,
// returning the file and the id.
func Open(dir string, now time.Time) (*os.File, string, error) {
	id := ID(now)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, "", err
	}

	path := filepath.Join(dir, id+".jsonl")
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return nil, "", err
	}
	return file, id, nil
}
