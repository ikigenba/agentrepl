package main

import (
	"bytes"
	"os"
	"os/exec"
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
	matches, err := filepath.Glob(filepath.Join(home, ".agentrepl", "logs", "*.jsonl"))
	if err != nil {
		t.Fatalf("checking log dir: %v", err)
	}
	if len(matches) != 1 {
		t.Fatalf("logs = %v, want one log under ~/.agentrepl/logs", matches)
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

	out.Reset()
	errOut.Reset()
	code = run([]string{"-h"}, strings.NewReader(""), &out, &errOut, false)
	if code != 0 {
		t.Fatalf("help exit code = %d, stderr %q", code, errOut.String())
	}
	if errOut.Len() != 0 {
		t.Fatalf("stderr = %q, want empty for help", errOut.String())
	}
	if !strings.Contains(out.String(), "usage: agentrepl") || !strings.Contains(out.String(), "providers:") {
		t.Fatalf("stdout = %q, want self-describing help", out.String())
	}
}

func TestRunReportsWorkingDirectoryFailureAsStartupFatal(t *testing.T) {
	previous, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd() before test: %v", err)
	}
	parent := t.TempDir()
	cwd := filepath.Join(parent, "removed")
	if err := os.Mkdir(cwd, 0o755); err != nil {
		t.Fatalf("os.Mkdir(%q): %v", cwd, err)
	}
	if err := os.Chdir(cwd); err != nil {
		t.Fatalf("os.Chdir(%q): %v", cwd, err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(previous); err != nil {
			t.Errorf("restore working directory to %q: %v", previous, err)
		}
	})
	if err := os.Remove(cwd); err != nil {
		t.Fatalf("os.Remove(%q): %v", cwd, err)
	}

	var out, errOut bytes.Buffer
	code := run(nil, strings.NewReader("/exit\n"), &out, &errOut, false)
	if code != 1 {
		t.Fatalf("working-directory failure exit code = %d, want 1", code)
	}
	if out.Len() != 0 {
		t.Fatalf("stdout = %q, want empty for startup fatal", out.String())
	}
	if !strings.Contains(errOut.String(), "startup: working dir:") {
		t.Fatalf("stderr = %q, want working-directory startup error", errOut.String())
	}
}

func TestRunVersionReportsUnstampedVersionAndShortCircuits(t *testing.T) {
	// R-S45L-UT0N
	if version != "dev" {
		t.Fatalf("unstamped version = %q, want dev", version)
	}

	var out, errOut bytes.Buffer
	code := run([]string{"-V", "-c", "not-a-key-value"}, strings.NewReader("this must not run\n"), &out, &errOut, false)
	if code != 0 {
		t.Fatalf("run exit code = %d, want 0; stderr = %q", code, errOut.String())
	}
	if got := out.String(); got != "dev\n" {
		t.Fatalf("stdout = %q, want %q", got, "dev\n")
	}
	if errOut.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", errOut.String())
	}
}

func TestRunVersionAliasesMatchShortFlag(t *testing.T) {
	// R-S6LE-MCI1
	for _, flag := range []string{"--version", "-version"} {
		t.Run(flag, func(t *testing.T) {
			var out, errOut bytes.Buffer
			code := run([]string{flag}, strings.NewReader("this must not run\n"), &out, &errOut, false)
			if code != 0 {
				t.Fatalf("run exit code = %d, want 0; stderr = %q", code, errOut.String())
			}
			if got := out.String(); got != "dev\n" {
				t.Fatalf("stdout = %q, want %q", got, "dev\n")
			}
			if errOut.Len() != 0 {
				t.Fatalf("stderr = %q, want empty", errOut.String())
			}
		})
	}
}

func TestVersionLinkerSeamStampsRealBinary(t *testing.T) {
	// R-S5DI-8KRC
	const sentinel = "v9.8.7-linker-test"
	binary := filepath.Join(t.TempDir(), "agentrepl")
	build := exec.Command("go", "build", "-ldflags", "-X main.version="+sentinel, "-o", binary, ".")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("building stamped binary: %v\n%s", err, output)
	}

	cmd := exec.Command(binary, "-V")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("running stamped binary: %v; stderr = %q", err, stderr.String())
	}
	if got := stdout.String(); got != sentinel+"\n" {
		t.Fatalf("stdout = %q, want %q", got, sentinel+"\n")
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}
