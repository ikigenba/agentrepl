package repl

import (
	"bytes"
	"context"
	"errors"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/ikigenba/agentkit"
	"github.com/ikigenba/agentrepl/internal/catalog"
	"github.com/ikigenba/agentrepl/internal/config"
	"github.com/ikigenba/agentrepl/internal/render"
)

func TestParseArgsCollectsRepeatedConfigInOrder(t *testing.T) {
	// R-EU69-75V4
	var usage bytes.Buffer
	opts, err := ParseArgs("agentrepl", []string{"-c", "system=first", "-raw", "-c", "system=second"}, &usage)
	if err != nil {
		t.Fatalf("ParseArgs returned error: %v", err)
	}
	want := []string{"system=first", "system=second"}
	if !slices.Equal(opts.Config, want) {
		t.Fatalf("Options.Config = %#v, want %#v", opts.Config, want)
	}
}

func TestParseArgsRawDefaultAndUnknownFlagUsage(t *testing.T) {
	// R-EWM1-YPCI
	var usage bytes.Buffer
	opts, err := ParseArgs("agentrepl", nil, &usage)
	if err != nil {
		t.Fatalf("ParseArgs returned error: %v", err)
	}
	if opts.Raw {
		t.Fatal("Options.Raw default = true, want false")
	}

	opts, err = ParseArgs("agentrepl", []string{"-raw"}, &usage)
	if err != nil {
		t.Fatalf("ParseArgs(-raw) returned error: %v", err)
	}
	if !opts.Raw {
		t.Fatal("Options.Raw = false, want true")
	}

	usage.Reset()
	if _, err := ParseArgs("agentrepl", []string{"-provider", "anthropic"}, &usage); err == nil {
		t.Fatal("ParseArgs unknown flag returned nil error")
	}
	if got := usage.String(); !strings.Contains(got, "flag provided but not defined") || !strings.Contains(got, "Usage of agentrepl") {
		t.Fatalf("unknown flag usage = %q, want flag error and usage", got)
	}
}

func TestRunAppliesStartupConfigInOrder(t *testing.T) {
	// R-EXTY-CH37
	out, errOut, code := runScript(t, "/get system\n/exit\n", Options{
		Config: []string{"system=first", "system=second"},
	})
	if code != 0 {
		t.Fatalf("Run exit code = %d, stderr %q", code, errOut)
	}
	if !strings.Contains(out, "system=second") {
		t.Fatalf("stdout = %q, want later config value", out)
	}
	if strings.Contains(out, "system=first") {
		t.Fatalf("stdout = %q, contains overridden config value", out)
	}
}

func TestRunBadStartupConfigIsFatalBeforeLoop(t *testing.T) {
	// R-EZ1U-Q8TW
	for _, tc := range []struct {
		name string
		pair string
		want string
	}{
		{name: "format", pair: "missing", want: "expected key=value"},
		{name: "key", pair: "nope=value", want: "unknown config key"},
		{name: "value", pair: "gen.max_tokens=not-int", want: "invalid value"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out, errOut, code := runScript(t, "/help\n", Options{Config: []string{tc.pair}})
			if code != 1 {
				t.Fatalf("Run exit code = %d, want 1", code)
			}
			if out != "" {
				t.Fatalf("stdout = %q, want empty because loop never started", out)
			}
			if !strings.Contains(errOut, "startup: config") || !strings.Contains(errOut, tc.want) {
				t.Fatalf("stderr = %q, want clear startup config error containing %q", errOut, tc.want)
			}
		})
	}
}

func TestSlashCommandDispatchUnknownIsNonFatal(t *testing.T) {
	// R-BI0J-TIHX
	out, errOut, code := runScript(t, "/does-not-exist\n/help\n/exit\n", Options{})
	if code != 0 {
		t.Fatalf("Run exit code = %d, stderr %q", code, errOut)
	}
	if !strings.Contains(out, "unknown command: /does-not-exist") {
		t.Fatalf("stdout = %q, want unknown command error", out)
	}
	if !strings.Contains(out, "/help") || !strings.Contains(out, "config keys:") {
		t.Fatalf("stdout = %q, want loop to continue to /help", out)
	}
}

func TestRuntimeConfigCommandsReachConfigAndSurviveErrors(t *testing.T) {
	// R-BKGC-L1ZB
	// R-H8PP-ZFI3
	out, errOut, code := runScript(t, strings.Join([]string{
		"/set system You are helpful",
		"/get system",
		"/dump",
		"/set provider anthropic",
		"/get provider",
		"/exit",
	}, "\n")+"\n", Options{})
	if code != 0 {
		t.Fatalf("Run exit code = %d, stderr %q", code, errOut)
	}
	for _, want := range []string{
		"system=You are helpful",
		"tool_loop_limit=default",
		"missing API key",
		"provider=default",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("stdout = %q, want %q", out, want)
		}
	}
}

func TestClearEmptiesHistoryAndLeavesSpend(t *testing.T) {
	// R-BLO8-YTQ0
	var out bytes.Buffer
	conv := &agentkit.Conversation{
		History: []agentkit.Message{{
			Role:   agentkit.RoleUser,
			Blocks: []agentkit.Block{agentkit.TextBlock{Text: "prior"}},
		}},
	}
	s := &state{
		conv: conv,
		rend: render.NewDecorated(&out, false),
	}
	before := conv.TotalCost()
	if err := commands["clear"].run(s, ""); err != nil {
		t.Fatalf("/clear returned error: %v", err)
	}
	if len(conv.History) != 0 {
		t.Fatalf("History length = %d, want 0", len(conv.History))
	}
	if after := conv.TotalCost(); after != before {
		t.Fatalf("TotalCost changed from %v to %v", before, after)
	}
}

func TestRenderCommandSwitchesRenderer(t *testing.T) {
	// R-BMW5-CLGP
	cat := catalog.Default()
	openai, ok := catalog.Lookup(cat, "openai")
	if !ok {
		t.Fatal("openai provider missing from catalog")
	}
	script := strings.Join([]string{
		"/set provider openai",
		"/set model " + openai.Models[0],
		"/render raw",
		"hello raw",
		"/render decorated",
		"hello decorated",
		"/exit",
	}, "\n") + "\n"
	out, errOut, code := runScriptWithEnv(t, script, Options{}, func(key string) string {
		if key == "OPENAI_API_KEY" {
			return "secret"
		}
		return ""
	})
	if code != 0 {
		t.Fatalf("Run exit code = %d, stderr %q", code, errOut)
	}
	if !strings.Contains(out, `{"type":"notice","text":"turn execution is not available in this build"}`) {
		t.Fatalf("stdout = %q, want raw JSON turn-stub notice after /render raw", out)
	}
	if !strings.Contains(out, "notice › turn execution is not available in this build") {
		t.Fatalf("stdout = %q, want decorated turn-stub notice after /render decorated", out)
	}
}

func TestHelpListsCommandsAndConfigKeys(t *testing.T) {
	// R-BO41-QD7E
	out, errOut, code := runScript(t, "/help\n/exit\n", Options{})
	if code != 0 {
		t.Fatalf("Run exit code = %d, stderr %q", code, errOut)
	}
	for _, want := range []string{"/set", "/get", "/dump", "/clear", "/render", "/providers", "/help", "/exit", "/quit", "gen.temperature", "tool_loop_limit"} {
		if !strings.Contains(out, want) {
			t.Fatalf("stdout = %q, want %q", out, want)
		}
	}
}

func TestProvidersListsEnvPresenceAndModels(t *testing.T) {
	// R-BPBY-44Y3
	out, errOut, code := runScriptWithEnv(t, "/providers\n/exit\n", Options{}, func(key string) string {
		if key == "OPENAI_API_KEY" {
			return "secret"
		}
		return ""
	})
	if code != 0 {
		t.Fatalf("Run exit code = %d, stderr %q", code, errOut)
	}
	for _, provider := range catalog.Default() {
		if !strings.Contains(out, provider.Name) || !strings.Contains(out, provider.EnvKey) || !strings.Contains(out, provider.Models[0]) {
			t.Fatalf("stdout = %q, want provider %s with env and models", out, provider.Name)
		}
	}
	if !strings.Contains(out, "OPENAI_API_KEY=present") || !strings.Contains(out, "ANTHROPIC_API_KEY=missing") {
		t.Fatalf("stdout = %q, want env presence markers", out)
	}
}

func TestTurnPrecheckHintsBeforeProviderAndModel(t *testing.T) {
	// R-BQJU-HWOS
	out, errOut, code := runScript(t, "hello\n/exit\n", Options{})
	if code != 0 {
		t.Fatalf("Run exit code = %d, stderr %q", code, errOut)
	}
	if !strings.Contains(out, "set a provider and model first") {
		t.Fatalf("stdout = %q, want provider/model hint", out)
	}
	if strings.Contains(out, "turn execution is not available") {
		t.Fatalf("stdout = %q, turn handler ran despite failed pre-check", out)
	}
}

func TestExitQuitAndEOFReturnCleanly(t *testing.T) {
	for _, tc := range []struct {
		name   string
		script string
	}{
		{name: "exit", script: "/exit\n"},
		{name: "quit", script: "/quit\n"},
		{name: "eof", script: ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out, errOut, code := runScript(t, tc.script, Options{})
			if code != 0 {
				t.Fatalf("Run exit code = %d, stderr %q, stdout %q", code, errOut, out)
			}
		})
	}
}

func TestRuntimeSelectionErrorWritesStdoutAndStartupFatalWritesStderr(t *testing.T) {
	out, errOut, code := runScript(t, "/set provider anthropic\n/exit\n", Options{})
	if code != 0 {
		t.Fatalf("Run exit code = %d, stderr %q", code, errOut)
	}
	if !strings.Contains(out, "missing API key") {
		t.Fatalf("stdout = %q, want runtime error", out)
	}
	if errOut != "" {
		t.Fatalf("stderr = %q, want empty for runtime error", errOut)
	}

	out, errOut, code = runScript(t, "/exit\n", Options{Config: []string{"nope=value"}})
	if code != 1 {
		t.Fatalf("Run exit code = %d, want 1", code)
	}
	if out != "" {
		t.Fatalf("stdout = %q, want empty for startup fatal", out)
	}
	if !strings.Contains(errOut, "unknown config key") {
		t.Fatalf("stderr = %q, want startup fatal", errOut)
	}
}

func TestRunReportsMissingClockAsStartupFatal(t *testing.T) {
	var out, errOut bytes.Buffer
	code := Run(context.Background(), Deps{
		IO: IO{
			In:  strings.NewReader("/help\n"),
			Out: &out,
			Err: &errOut,
		},
		LogDir: t.TempDir(),
	}, Options{})
	if code != 1 {
		t.Fatalf("Run exit code = %d, want 1", code)
	}
	if out.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", out.String())
	}
	if !strings.Contains(errOut.String(), "missing clock") {
		t.Fatalf("stderr = %q, want missing clock", errOut.String())
	}
}

func TestCommandTableSetPropagatesConfigSentinels(t *testing.T) {
	var out bytes.Buffer
	conv := &agentkit.Conversation{}
	s := &state{
		conv: conv,
		target: &config.Target{
			Conv: conv,
			Catalog: []catalog.Provider{{
				Name:   "test",
				EnvKey: "TEST_KEY",
				New: func(string) agentkit.Provider {
					t.Fatal("constructor should not be called for unknown provider")
					return nil
				},
			}},
			Getenv: func(string) string { return "" },
		},
		rend: render.NewDecorated(&out, false),
	}
	err := commands["set"].run(s, "missing value")
	if !errors.Is(err, config.ErrUnknownKey) {
		t.Fatalf("/set error = %v, want ErrUnknownKey", err)
	}
}

func runScript(t *testing.T, script string, opts Options) (stdout, stderr string, code int) {
	t.Helper()
	return runScriptWithEnv(t, script, opts, func(string) string { return "" })
}

func runScriptWithEnv(t *testing.T, script string, opts Options, getenv func(string) string) (stdout, stderr string, code int) {
	t.Helper()
	var out, errOut bytes.Buffer
	logDir := t.TempDir()
	code = Run(context.Background(), Deps{
		IO: IO{
			In:  strings.NewReader(script),
			Out: &out,
			Err: &errOut,
		},
		Getenv: getenv,
		Now: func() time.Time {
			return time.Date(2026, 6, 18, 12, 0, 0, 123456000, time.UTC)
		},
		LogDir: logDir,
	}, opts)
	if _, err := filepath.Glob(filepath.Join(logDir, "*.jsonl")); err != nil {
		t.Fatalf("checking log dir: %v", err)
	}
	return out.String(), errOut.String(), code
}
