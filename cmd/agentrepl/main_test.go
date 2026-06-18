package main

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunWiresHomeLogDirAndExitCodes(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("NO_COLOR", "1")

	var out, errOut bytes.Buffer
	code := run(nil, strings.NewReader("/exit\n"), &out, &errOut, true)
	if code != 0 {
		t.Fatalf("run exit code = %d, stderr %q", code, errOut.String())
	}
	if errOut.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", errOut.String())
	}
	matches, err := filepath.Glob(filepath.Join(home, ".agentkit", "*.jsonl"))
	if err != nil {
		t.Fatalf("checking log dir: %v", err)
	}
	if len(matches) != 1 {
		t.Fatalf("logs = %v, want one log under ~/.agentkit", matches)
	}

	out.Reset()
	errOut.Reset()
	code = run([]string{"-unknown"}, strings.NewReader(""), &out, &errOut, false)
	if code != 1 {
		t.Fatalf("bad-flag exit code = %d, want 1", code)
	}
	if out.Len() != 0 {
		t.Fatalf("stdout = %q, want empty for startup fatal", out.String())
	}
	if !strings.Contains(errOut.String(), "flag provided but not defined") {
		t.Fatalf("stderr = %q, want flag error", errOut.String())
	}
}
