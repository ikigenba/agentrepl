package repl

import (
	"bytes"
	"testing"
	"time"
)

func TestSkeletonSeamsCompile(t *testing.T) {
	var in bytes.Buffer
	var out bytes.Buffer
	var errOut bytes.Buffer

	io := IO{
		In:    &in,
		Out:   &out,
		Err:   &errOut,
		IsTTY: true,
	}
	if io.In == nil || io.Out == nil || io.Err == nil || !io.IsTTY {
		t.Fatal("IO seam was not populated")
	}

	getenv := Getenv(func(key string) string { return "value:" + key })
	if got := getenv("AGENTREPL_TEST"); got != "value:AGENTREPL_TEST" {
		t.Fatalf("Getenv seam returned %q", got)
	}

	fixed := time.Date(2026, 6, 18, 12, 0, 0, 0, time.UTC)
	now := Now(func() time.Time { return fixed })
	if got := now(); !got.Equal(fixed) {
		t.Fatalf("Now seam returned %v", got)
	}
}
