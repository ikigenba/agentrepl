package repl

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
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
	if got := usage.String(); !strings.Contains(got, "flag provided but not defined") || !strings.Contains(got, "usage: agentrepl") {
		t.Fatalf("unknown flag usage = %q, want flag error and usage", got)
	}
}

func TestParseArgsHelpWritesCatalogAndReturnsErrHelpCredentialBlind(t *testing.T) {
	// R-FT8W-Z2U4
	constructed := false
	originalCatalog := defaultCatalog
	defaultCatalog = func() []catalog.Provider {
		return []catalog.Provider{{
			Name:   "test",
			EnvKey: "TEST_API_KEY",
			Models: []string{"test-model"},
			New: func(string, catalog.Options) agentkit.Provider {
				constructed = true
				return nil
			},
			Reasoning: staticReasoning{"test-model": {
				Term: "effort", Kind: agentkit.ReasoningEnum,
				Levels: []string{"low", "high"}, Default: agentkit.Level("high"),
			}},
		}}
	}
	t.Cleanup(func() {
		defaultCatalog = originalCatalog
	})

	var out bytes.Buffer
	_, err := ParseArgs("agentrepl", []string{"-h"}, &out)
	if !errors.Is(err, flag.ErrHelp) {
		t.Fatalf("ParseArgs(-h) error = %v, want flag.ErrHelp", err)
	}
	for _, want := range []string{"usage: agentrepl", "test        (TEST_API_KEY)", "test-model", "effort={low|*high}"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("help output = %q, want %q", out.String(), want)
		}
	}
	if constructed {
		t.Fatal("help constructed a provider")
	}
}

func TestWriteHelpListsDefaultCatalogInOrder(t *testing.T) {
	// R-FUGT-CUKT
	cat := catalog.Default()
	var out bytes.Buffer
	WriteHelp(&out, "agentrepl", cat)
	help := out.String()

	last := -1
	for _, provider := range cat {
		providerLine := fmt.Sprintf("  %-10s  (%s)", provider.Name, provider.EnvKey)
		index := strings.Index(help, providerLine)
		if index <= last {
			t.Fatalf("provider line %q index = %d after %d in help:\n%s", providerLine, index, last, help)
		}
		last = index
	}

	modelsIndex := strings.Index(help, "models:\n")
	if modelsIndex < 0 {
		t.Fatalf("help = %q, want models section", help)
	}
	last = modelsIndex
	for _, provider := range cat {
		providerIndex := strings.Index(help[last:], "  "+provider.Name+"\n")
		if providerIndex < 0 {
			t.Fatalf("help = %q, want models group %s", help, provider.Name)
		}
		last += providerIndex
		for _, model := range provider.Models {
			modelIndex := strings.Index(help[last:], "    "+model)
			if modelIndex < 0 {
				t.Fatalf("help = %q, want model %s under %s", help, model, provider.Name)
			}
			last += modelIndex
		}
	}
}

func TestWriteHelpGoldenReasoningClausesByKind(t *testing.T) {
	// R-FVOP-QMBI
	// R-6DEO-9TXQ
	// R-ODOF-XOTJ
	// R-OEWC-BGK8
	var out bytes.Buffer
	WriteHelp(&out, "agentrepl-test", catalog.Default())
	want, err := os.ReadFile(filepath.Join("testdata", "help_reasoning.golden"))
	if err != nil {
		t.Fatalf("reading golden: %v", err)
	}
	if out.String() != string(want) {
		t.Fatalf("help output mismatch\nwant:\n%s\ngot:\n%s", want, out.String())
	}
}

func TestWriteHelpMarksEnumeratedDefaultsAndRetainsOnlyTermResidue(t *testing.T) {
	// R-ODOF-XOTJ
	// R-OEWC-BGK8
	cat := []catalog.Provider{{
		Name:   "test",
		EnvKey: "TEST_API_KEY",
		Models: []string{"enum", "enum-with-residue", "toggle", "range"},
		Reasoning: staticReasoning{
			"enum": {
				Term: "effort", Kind: agentkit.ReasoningEnum,
				Levels: []string{"low", "medium", "high"}, Default: agentkit.Level("medium"),
			},
			"enum-with-residue": {
				Term: "effort (+ toggle)", Kind: agentkit.ReasoningEnum,
				Levels: []string{"high", "max"}, Default: agentkit.Level("max"),
			},
			"toggle": {
				Term: "thinking", Kind: agentkit.ReasoningToggle,
			},
			"range": {
				Term: "thinking budget", Kind: agentkit.ReasoningRange,
				Min: 0, Max: 24576, Default: agentkit.Budget(-1),
				Sentinels: []agentkit.Sentinel{{Value: -1, Meaning: "dynamic"}},
			},
		},
	}}

	var out bytes.Buffer
	WriteHelp(&out, "agentrepl", cat)
	help := out.String()
	for _, want := range []string{
		"effort={low|*medium|high}\n",
		"effort={high|*max}  (+ toggle)\n",
		"thinking={*on|off}\n",
		"thinking_budget=<0–24576>  (thinking budget; -1=dynamic; default dynamic)\n",
	} {
		if !strings.Contains(help, want) {
			t.Errorf("help output = %q, want row ending %q", help, want)
		}
	}
	for _, unwanted := range []string{"(effort;", "(thinking;", "default medium", "default max", "default on"} {
		if strings.Contains(help, unwanted) {
			t.Errorf("help output = %q, do not want %q", help, unwanted)
		}
	}
}

func TestWriteHelpReasoningTermsMapToRegisteredConfigKeys(t *testing.T) {
	// R-6DEO-KEYS
	keys := map[string]bool{}
	for _, key := range config.Keys() {
		keys[key] = true
	}
	for _, provider := range catalog.Default() {
		for _, model := range provider.Models {
			spec, ok := provider.Reasoning.ReasoningSpec(model)
			if !ok {
				continue
			}
			key := termToKey(spec.Term)
			if !keys[key] {
				t.Fatalf("%s %s termToKey(%q) = %q, want registered config key", provider.Name, model, spec.Term, key)
			}
			switch key {
			case "effort", "thinking_budget", "thinking_level", "thinking":
			default:
				t.Fatalf("%s %s termToKey(%q) = %q, want native reasoning key", provider.Name, model, spec.Term, key)
			}
		}
	}
}

func TestWriteHelpIncludesModelsWithoutReasoningSpec(t *testing.T) {
	// R-FWWM-4E27
	cat := []catalog.Provider{{
		Name:      "plain",
		EnvKey:    "PLAIN_KEY",
		Models:    []string{"plain-model"},
		Reasoning: staticReasoning{},
	}}
	var out bytes.Buffer
	WriteHelp(&out, "agentrepl", cat)
	help := out.String()
	if !strings.Contains(help, "plain-model") || !strings.Contains(help, "(no reasoning control)") {
		t.Fatalf("help = %q, want plain model with no-reasoning clause", help)
	}
}

func TestWriteHelpDoesNotConstructProviders(t *testing.T) {
	// R-FY4I-I5SW
	constructed := false
	cat := []catalog.Provider{{
		Name:   "credential-blind",
		EnvKey: "SECRET_KEY",
		Models: []string{"model"},
		New: func(string, catalog.Options) agentkit.Provider {
			constructed = true
			return nil
		},
		Reasoning: staticReasoning{},
	}}
	var out bytes.Buffer
	WriteHelp(&out, "agentrepl", cat)
	if constructed {
		t.Fatal("WriteHelp constructed a provider")
	}
	if !strings.Contains(out.String(), "SECRET_KEY") || !strings.Contains(out.String(), "model") {
		t.Fatalf("help = %q, want env-key documentation and model without credentials", out.String())
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
		{name: "value", pair: "max_tokens=not-int", want: "invalid value"},
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
		rend: render.NewDecorated(&out, false, false),
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
	provider := newScriptedProvider(successRound("raw ok", usageOne()), successRound("decorated ok", usageTwo()))
	script := strings.Join([]string{
		"/set provider test",
		"/set model test-model",
		"/render raw",
		"hello raw",
		"/render decorated",
		"hello decorated",
		"/exit",
	}, "\n") + "\n"
	out, errOut, _, code := runScriptWithProvider(t, script, Options{}, provider)
	if code != 0 {
		t.Fatalf("Run exit code = %d, stderr %q", code, errOut)
	}
	if !strings.Contains(out, `{"type":"prompt","text":"hello raw"}`) ||
		!strings.Contains(out, `{"type":"usage"`) {
		t.Fatalf("stdout = %q, want raw JSON turn after /render raw", out)
	}
	if !strings.Contains(out, "assistant › decorated ok") ||
		strings.Contains(out, "hello decorated") {
		t.Fatalf("stdout = %q, want decorated turn after /render decorated", out)
	}
}

func TestHelpListsCommandsAndConfigKeys(t *testing.T) {
	// R-BO41-QD7E
	out, errOut, code := runScript(t, "/help\n/exit\n", Options{})
	if code != 0 {
		t.Fatalf("Run exit code = %d, stderr %q", code, errOut)
	}
	for _, want := range []string{"/set", "/get", "/dump", "/clear", "/render", "/providers", "/help", "/exit", "/quit", "temperature", "tool_loop_limit"} {
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

func TestTurnMessageDrivesConversationAndEmptyLineIsIgnored(t *testing.T) {
	// R-BJ8G-7A8M
	provider := newScriptedProvider(successRound("hi", usageOne()))
	out, errOut, log, code := runScriptWithProvider(t, "\nhello\n/exit\n", Options{}, provider)
	if code != 0 {
		t.Fatalf("Run exit code = %d, stderr %q", code, errOut)
	}
	if len(provider.requests) != 1 {
		t.Fatalf("provider request count = %d, want 1", len(provider.requests))
	}
	if !strings.Contains(out, "assistant › hi") || strings.Contains(out, "you › hello") {
		t.Fatalf("stdout = %q, want completed turn for non-command input", out)
	}
	assertLogTypes(t, log, []string{"turn_start", "message", "usage", "turn_end", "summary"})
}

func TestTTYPromptPrecedesEachInputReadAndDoesNotEchoInput(t *testing.T) {
	// R-JFBW-TYU8
	provider := newScriptedProvider(successRound("hi", usageOne()))
	script := strings.Join([]string{
		"/help",
		"",
		"/set provider test",
		"/set model test-model",
		"hello",
		"/exit",
	}, "\n") + "\n"
	result := runScriptWithProviderContextAndIO(t, context.Background(), script, Options{}, provider, IO{IsTTY: true})
	if result.code != 0 {
		t.Fatalf("Run exit code = %d, stderr %q", result.code, result.stderr)
	}
	if got := strings.Count(result.stdout, "you › "); got < 6 {
		t.Fatalf("stdout = %q, prompt count = %d, want prompt before command, empty line, turn, and exit reads", result.stdout, got)
	}
	if !strings.Contains(result.stdout, "you › \nnotice › /clear") {
		t.Fatalf("stdout = %q, want first prompt to precede /help output", result.stdout)
	}
	if !strings.Contains(result.stdout, "you › \nassistant › hi") {
		t.Fatalf("stdout = %q, want turn prompt to precede assistant output", result.stdout)
	}
	if strings.Contains(result.stdout, "you › hello") {
		t.Fatalf("stdout = %q, decorated Input echoed entered turn", result.stdout)
	}
}

func TestWaiterStopsOnFirstOutputAndAgainByDefer(t *testing.T) {
	// R-6F74-SX99
	var calls []string
	waiter := &recordingWaiter{calls: &calls}
	rend := &recordingRenderer{calls: &calls}
	provider := newScriptedProvider(successRound("hi", usageOne()))
	conv := &agentkit.Conversation{Provider: provider, Model: "test-model"}
	state := &state{conv: conv, rend: rend, waiter: waiter}

	handleTurn(context.Background(), state, "hello")

	if state.quit {
		t.Fatalf("state.quit = true, want false")
	}
	if len(calls) < 4 {
		t.Fatalf("calls = %#v, want waiter start, first-output stop, render event, defer stop", calls)
	}
	if calls[0] != "waiter:start:test-model" {
		t.Fatalf("calls = %#v, want Start with model before output", calls)
	}
	if calls[1] != "waiter:stop" {
		t.Fatalf("calls = %#v, want Stop before first rendered event", calls)
	}
	if !strings.HasPrefix(calls[2], "render:event:") {
		t.Fatalf("calls = %#v, want rendered event after first Stop", calls)
	}
	if calls[len(calls)-1] != "waiter:stop" {
		t.Fatalf("calls = %#v, want defer Stop at end of turn", calls)
	}
}

func TestWaiterStopsByDeferForErrorEmptyAndInterrupt(t *testing.T) {
	// R-6F74-SX99
	tests := []struct {
		name      string
		run       func(t *testing.T, calls *[]string) *state
		wantQuit  bool
		wantCode  int
		wantCalls []string
	}{
		{
			name: "error",
			run: func(t *testing.T, calls *[]string) *state {
				t.Helper()
				state := newWaiterTestState(calls, newScriptedProvider(errorRound("provider failed")))
				handleTurn(context.Background(), state, "hello")
				return state
			},
			wantCalls: []string{"waiter:start:test-model", "waiter:stop", "render:error:provider failed", "waiter:stop"},
		},
		{
			name: "empty",
			run: func(t *testing.T, calls *[]string) *state {
				t.Helper()
				empty := agentkit.NewRoundTrip(agentkit.Message{}, agentkit.FinishStop, agentkit.Usage{}, nil, nil)
				state := newWaiterTestState(calls, newScriptedProvider(empty))
				handleTurn(context.Background(), state, "hello")
				return state
			},
			wantCalls: []string{"waiter:start:test-model", "waiter:stop", "render:event:agentkit.MessageDone", "render:usage", "waiter:stop"},
		},
		{
			name: "interrupt",
			run: func(t *testing.T, calls *[]string) *state {
				t.Helper()
				provider := newBlockingProvider()
				state := newWaiterTestState(calls, provider)
				ctx, cancel := context.WithCancel(context.Background())
				done := make(chan struct{})
				go func() {
					defer close(done)
					handleTurn(ctx, state, "hello")
				}()
				<-provider.started
				cancel()
				select {
				case <-done:
				case <-time.After(time.Second):
					t.Fatal("handleTurn did not return after interrupt")
				}
				return state
			},
			wantQuit:  true,
			wantCode:  130,
			wantCalls: []string{"waiter:start:test-model", "waiter:stop", "render:notice:interrupted", "waiter:stop"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var calls []string
			state := tc.run(t, &calls)
			if !slices.Equal(calls, tc.wantCalls) {
				t.Fatalf("calls = %#v, want %#v", calls, tc.wantCalls)
			}
			if state.quit != tc.wantQuit {
				t.Fatalf("state.quit = %v, want %v", state.quit, tc.wantQuit)
			}
			if state.exitCode != tc.wantCode {
				t.Fatalf("state.exitCode = %d, want %d", state.exitCode, tc.wantCode)
			}
		})
	}
}

func TestWaiterIsNopForRawAndNonTTYRuns(t *testing.T) {
	// R-6F74-SX99
	tests := []struct {
		name   string
		script string
		ioDeps IO
		opts   Options
	}{
		{
			name:   "render raw command",
			script: "/render raw\nhello\n/exit\n",
			ioDeps: IO{IsTTY: true},
		},
		{
			name:   "non tty decorated",
			script: "hello\n/exit\n",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			waiter := &recordingWaiter{}
			result := runScriptWithProviderContextIOAndWaiter(t, context.Background(), tc.script, tc.opts, newScriptedProvider(successRound("hi", usageOne())), tc.ioDeps, waiter)
			if result.code != 0 {
				t.Fatalf("Run exit code = %d, stderr %q", result.code, result.stderr)
			}
			if len(waiter.ownedCalls) != 0 {
				t.Fatalf("waiter calls = %#v, want none", waiter.ownedCalls)
			}
			if strings.Contains(result.stdout, "waiting for test-model") {
				t.Fatalf("stdout = %q, want no wait status line", result.stdout)
			}
		})
	}
}

func TestFailedTurnRendersErrorSkipsUsageAndContinues(t *testing.T) {
	// R-LSKZ-36TW
	// R-OPZQ-Y90U
	// R-H7HT-LNRE
	provider := newScriptedProvider(errorRound("provider failed"), successRound("after error", usageOne()))
	out, errOut, log, code := runScriptWithProvider(t, "fail once\nrecover\n/exit\n", Options{}, provider)
	if code != 0 {
		t.Fatalf("Run exit code = %d, stderr %q", code, errOut)
	}
	if !strings.Contains(out, "error › provider failed") {
		t.Fatalf("stdout = %q, want rendered turn error", out)
	}
	if !strings.Contains(out, "assistant › after error") || strings.Contains(out, "you › recover") {
		t.Fatalf("stdout = %q, want loop to accept next input", out)
	}
	if strings.Contains(out, "· cost     $0.002000 turn") {
		t.Fatalf("stdout = %q, decorated output should suppress per-turn usage", out)
	}
	if !strings.Contains(out, "· cost     $0.002000 session") {
		t.Fatalf("stdout = %q, want errored turn excluded from session cumulative", out)
	}
	assertLogTypes(t, log, []string{"turn_start", "error", "turn_end", "turn_start", "message", "usage", "turn_end", "summary"})
}

func TestTurnWarningsRelayBeforeUsageAndError(t *testing.T) {
	// R-G480-F0ID
	warnings := []agentkit.Warning{
		{Setting: "reasoning", Code: agentkit.WarnReasoningUnsupported, Detail: "xhigh is not supported"},
		{Setting: "tool_schema", Code: agentkit.WarnToolSchemaLossy, Detail: "dropped keyword"},
	}
	provider := newScriptedProvider(toolUseRoundWithWarnings(warnings), errorRound("provider failed"), successRound("after warning", usageOne()))
	out, errOut, _, code := runScriptWithProvider(t, "warn then fail\nno warning\n/exit\n", Options{Raw: true}, provider)
	if code != 0 {
		t.Fatalf("Run exit code = %d, stderr %q", code, errOut)
	}
	records := decodeLogRecords(t, out)
	gotTypes := recordTypes(t, records)
	wantTypes := []string{"prompt", "message_done", "tool_use", "tool_result", "warning", "warning", "error", "prompt", "message_done", "usage", "summary"}
	if !slices.Equal(gotTypes, wantTypes) {
		t.Fatalf("stdout record types = %#v, want %#v\nstdout:\n%s", gotTypes, wantTypes, out)
	}
	first := records[4]
	if first["Setting"] != warnings[0].Setting || first["Code"] != float64(warnings[0].Code) || first["Detail"] != warnings[0].Detail {
		t.Fatalf("first warning = %#v, want verbatim %#v", first, warnings[0])
	}
	second := records[5]
	if second["Setting"] != warnings[1].Setting || second["Code"] != float64(warnings[1].Code) || second["Detail"] != warnings[1].Detail {
		t.Fatalf("second warning = %#v, want verbatim %#v", second, warnings[1])
	}
}

func TestNonNativeReasoningWarningRelayedAndTurnCompletesWithDefault(t *testing.T) {
	// R-G6NT-6JZR
	provider := &reasoningWarningProvider{}
	result := runScriptWithProviderContext(t, context.Background(), "hello\n/exit\n", Options{
		Raw:    true,
		Config: []string{"effort=xhigh"},
	}, provider)
	if result.code != 0 {
		t.Fatalf("Run exit code = %d, stderr %q", result.code, result.stderr)
	}
	if len(provider.requests) != 1 {
		t.Fatalf("provider request count = %d, want 1", len(provider.requests))
	}
	if level, ok := provider.requests[0].Gen.Reasoning.Level(); !ok || level != "xhigh" {
		t.Fatalf("request reasoning = %#v, want non-native level xhigh from config carve-out", provider.requests[0].Gen.Reasoning)
	}
	records := decodeLogRecords(t, result.stdout)
	gotTypes := recordTypes(t, records)
	wantTypes := []string{"prompt", "message_done", "warning", "usage", "summary"}
	if !slices.Equal(gotTypes, wantTypes) {
		t.Fatalf("stdout record types = %#v, want %#v\nstdout:\n%s", gotTypes, wantTypes, result.stdout)
	}
	warning := records[2]
	if warning["Setting"] != "reasoning" || warning["Code"] != float64(agentkit.WarnReasoningUnsupported) {
		t.Fatalf("warning record = %#v, want reasoning unsupported warning", warning)
	}
	if !strings.Contains(result.stdout, `"type":"usage"`) || strings.Contains(result.stdout, `"type":"error"`) {
		t.Fatalf("stdout = %q, want completed turn with usage and no error", result.stdout)
	}
}

func TestUsageCommandRendersAgentkitCumulativeSummary(t *testing.T) {
	// R-OSFJ-PSI8
	provider := newScriptedProvider(successRound("first", usageOne()), successRound("second", usageTwo()))
	out, errOut, _, code := runScriptWithProvider(t, "one\n/usage\ntwo\n/exit\n", Options{}, provider)
	if code != 0 {
		t.Fatalf("Run exit code = %d, stderr %q", code, errOut)
	}
	firstSummary := "summary\n· tokens  in=100 cache(r=0 w=0) out=50 reasoning=0 total=150\n· cost     $0.002000 session"
	if !strings.Contains(out, firstSummary) {
		t.Fatalf("stdout = %q, want /usage summary sourced from first completed turn", out)
	}
	finalSummary := "summary\n· tokens  in=300 cache(r=0 w=0) out=150 reasoning=0 total=450\n· cost     $0.006000 session"
	if !strings.Contains(out, finalSummary) {
		t.Fatalf("stdout = %q, want exit summary sourced from both completed turns", out)
	}
}

func TestExitQuitAndEOFReturnCleanly(t *testing.T) {
	// R-LW8O-8I1Z
	// R-OUVC-HBZM
	for _, tc := range []struct {
		name   string
		script string
	}{
		{name: "exit", script: "/exit\n"},
		{name: "quit", script: "/quit\n"},
		{name: "eof", script: ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out, errOut, log, code := runScriptWithProvider(t, tc.script, Options{}, newScriptedProvider())
			if code != 0 {
				t.Fatalf("Run exit code = %d, stderr %q, stdout %q", code, errOut, out)
			}
			if !strings.HasSuffix(out, "summary\n· tokens  in=0 cache(r=0 w=0) out=0 reasoning=0 total=0\n· cost     $0.000000 session\n") {
				t.Fatalf("stdout = %q, want summary as final output", out)
			}
			assertLogTypes(t, log, []string{"summary"})
		})
	}
}

func TestIdleInterruptExitsThroughGracefulCleanup(t *testing.T) {
	// R-LXGK-M9SO
	// R-M149-RL0R
	reader, writer := io.Pipe()
	t.Cleanup(func() {
		_ = reader.Close()
		_ = writer.Close()
	})
	ctx, cancel := context.WithCancel(context.Background())
	var out, errOut bytes.Buffer
	logDir := t.TempDir()
	done := make(chan int, 1)

	go func() {
		done <- Run(ctx, Deps{
			IO: IO{
				In:  reader,
				Out: &out,
				Err: &errOut,
			},
			Getenv: func(string) string { return "" },
			Now: func() time.Time {
				return time.Date(2026, 6, 18, 12, 0, 0, 123456000, time.UTC)
			},
			LogDir: logDir,
		}, Options{})
	}()

	cancel()
	code := awaitRun(t, done)
	if code != 130 {
		t.Fatalf("Run exit code = %d, want 130", code)
	}
	if errOut.String() != "" {
		t.Fatalf("stderr = %q, want empty for interrupt", errOut.String())
	}
	if !strings.Contains(out.String(), "notice › interrupted") {
		t.Fatalf("stdout = %q, want interrupt notice", out.String())
	}
	if !strings.HasSuffix(out.String(), "summary\n· tokens  in=0 cache(r=0 w=0) out=0 reasoning=0 total=0\n· cost     $0.000000 session\n") {
		t.Fatalf("stdout = %q, want summary as final output", out.String())
	}
	log := readOnlyLog(t, logDir)
	assertLogTypes(t, log, []string{"summary"})
}

func TestStreamingInterruptLeavesValidLogAndDoesNotAccumulateUsage(t *testing.T) {
	// R-LYOH-01JD
	// R-OPZQ-Y90U
	// R-OUVC-HBZM
	provider := newInterruptProvider()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan runResult, 1)

	go func() {
		done <- runScriptWithProviderContext(t, ctx, "before\ninterrupt\n", Options{}, provider)
	}()

	<-provider.interruptStarted
	cancel()
	result := awaitRunResult(t, done)
	if result.code != 130 {
		t.Fatalf("Run exit code = %d, stderr %q", result.code, result.stderr)
	}
	if strings.Contains(result.stdout, "error ›") {
		t.Fatalf("stdout = %q, want interrupt notice instead of rendered error", result.stdout)
	}
	if !strings.Contains(result.stdout, "· cost     $0.002000 session") {
		t.Fatalf("stdout = %q, want final summary to keep pre-interrupt cumulative cost only", result.stdout)
	}
	if !strings.HasSuffix(result.stdout, "summary\n· tokens  in=100 cache(r=0 w=0) out=50 reasoning=0 total=150\n· cost     $0.002000 session\n") {
		t.Fatalf("stdout = %q, want summary as final output", result.stdout)
	}
	records := decodeLogRecords(t, result.log)
	if len(records) < 3 {
		t.Fatalf("log records = %#v, want turn_end then summary", records)
	}
	if records[len(records)-2]["type"] != "turn_end" || records[len(records)-1]["type"] != "summary" {
		t.Fatalf("last log records = %#v, want turn_end then summary\nlog:\n%s", records[len(records)-2:], result.log)
	}
	for i, record := range records {
		if record["type"] == nil {
			t.Fatalf("log record %d missing type after JSON decode: %#v", i, record)
		}
	}
}

func TestCompletedRunWritesConversationRecordsAndSummaryToLog(t *testing.T) {
	// R-8IUX-DBG8
	provider := newScriptedProvider(toolUseRound(), successRound("done", usageTwo()))
	_, errOut, log, code := runScriptWithProvider(t, "use tool\n/exit\n", Options{}, provider)
	if code != 0 {
		t.Fatalf("Run exit code = %d, stderr %q", code, errOut)
	}
	assertLogTypes(t, log, []string{"turn_start", "message", "tool_use", "tool_result", "message", "usage", "turn_end", "summary"})
	records := decodeLogRecords(t, log)
	if records[0]["provider"] != "test" || records[0]["model"] != "test-model" {
		t.Fatalf("turn_start = %#v, want provider/model", records[0])
	}
	if records[len(records)-2]["status"] != "ok" {
		t.Fatalf("turn_end = %#v, want ok status", records[len(records)-2])
	}
}

func TestSessionLogIsIndependentOfRenderMode(t *testing.T) {
	// R-8K2T-R36X
	decoratedProvider := newScriptedProvider(successRound("same", usageOne()))
	_, errOut, decoratedLog, code := runScriptWithProvider(t, "hello\n/exit\n", Options{}, decoratedProvider)
	if code != 0 {
		t.Fatalf("decorated Run exit code = %d, stderr %q", code, errOut)
	}

	rawProvider := newScriptedProvider(successRound("same", usageOne()))
	_, errOut, rawLog, code := runScriptWithProvider(t, "hello\n/exit\n", Options{Raw: true}, rawProvider)
	if code != 0 {
		t.Fatalf("raw Run exit code = %d, stderr %q", code, errOut)
	}

	decorated := normalizedLogRecords(t, decoratedLog)
	raw := normalizedLogRecords(t, rawLog)
	if !reflect.DeepEqual(raw, decorated) {
		t.Fatalf("raw log = %#v, want decorated log %#v", raw, decorated)
	}
}

func TestRuntimeSelectionErrorWritesStdoutAndStartupFatalWritesStderr(t *testing.T) {
	// R-HB5I-QYZH
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

func TestExpectedFailuresRenderAndDoNotEndLoop(t *testing.T) {
	// R-H9XM-D78S
	// R-HCDF-4QQ6
	provider := newScriptedProvider(toolUseRound(), errorRound("provider failed"), successRound("after failures", usageOne()))
	out, errOut, _, code := runScriptWithProvider(t, strings.Join([]string{
		"/does-not-exist",
		"/set max_tokens nope",
		"use missing file tool",
		"provider failure",
		"still alive",
		"/exit",
	}, "\n")+"\n", Options{}, provider)
	if code != 0 {
		t.Fatalf("Run exit code = %d, stderr %q", code, errOut)
	}
	for _, want := range []string{
		"unknown command",
		"invalid value",
		"tool error › read",
		"provider failed",
		"assistant › after failures",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("stdout = %q, want expected failure or recovery marker %q", out, want)
		}
	}
	if errOut != "" {
		t.Fatalf("stderr = %q, want empty for in-loop expected failures", errOut)
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
				New: func(string, catalog.Options) agentkit.Provider {
					t.Fatal("constructor should not be called for unknown provider")
					return nil
				},
			}},
			Getenv: func(string) string { return "" },
		},
		rend: render.NewDecorated(&out, false, false),
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

func runScriptWithProvider(t *testing.T, script string, opts Options, provider *scriptedProvider) (stdout, stderr, log string, code int) {
	t.Helper()
	originalCatalog := defaultCatalog
	defaultCatalog = func() []catalog.Provider {
		return []catalog.Provider{{
			Name:   "test",
			EnvKey: "TEST_API_KEY",
			Models: []string{"test-model"},
			New: func(string, catalog.Options) agentkit.Provider {
				return provider
			},
		}}
	}
	t.Cleanup(func() {
		defaultCatalog = originalCatalog
	})

	var out, errOut bytes.Buffer
	logDir := t.TempDir()
	code = Run(context.Background(), Deps{
		IO: IO{
			In:  strings.NewReader(script),
			Out: &out,
			Err: &errOut,
		},
		Getenv: func(key string) string {
			if key == "TEST_API_KEY" {
				return "secret"
			}
			return ""
		},
		Now: func() time.Time {
			return time.Date(2026, 6, 18, 12, 0, 0, 123456000, time.UTC)
		},
		LogDir: logDir,
	}, Options{
		Config: append([]string{"provider=test", "model=test-model"}, opts.Config...),
		Raw:    opts.Raw,
	})
	matches, err := filepath.Glob(filepath.Join(logDir, "*.jsonl"))
	if err != nil {
		t.Fatalf("checking log dir: %v", err)
	}
	if len(matches) != 1 {
		t.Fatalf("log files = %v, want exactly one", matches)
	}
	content, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatalf("reading log file: %v", err)
	}
	return out.String(), errOut.String(), string(content), code
}

type runResult struct {
	stdout string
	stderr string
	log    string
	code   int
}

func runScriptWithProviderContext(t *testing.T, ctx context.Context, script string, opts Options, provider agentkit.Provider) runResult {
	t.Helper()
	return runScriptWithProviderContextAndIO(t, ctx, script, opts, provider, IO{})
}

func runScriptWithProviderContextAndIO(t *testing.T, ctx context.Context, script string, opts Options, provider agentkit.Provider, ioDeps IO) runResult {
	t.Helper()
	return runScriptWithProviderContextIOAndWaiter(t, ctx, script, opts, provider, ioDeps, nil)
}

func runScriptWithProviderContextIOAndWaiter(t *testing.T, ctx context.Context, script string, opts Options, provider agentkit.Provider, ioDeps IO, waiter Waiter) runResult {
	t.Helper()
	originalCatalog := defaultCatalog
	defaultCatalog = func() []catalog.Provider {
		return []catalog.Provider{{
			Name:   "test",
			EnvKey: "TEST_API_KEY",
			Models: []string{"test-model"},
			New: func(string, catalog.Options) agentkit.Provider {
				return provider
			},
		}}
	}
	t.Cleanup(func() {
		defaultCatalog = originalCatalog
	})

	var out, errOut bytes.Buffer
	ioDeps.In = strings.NewReader(script)
	ioDeps.Out = &out
	ioDeps.Err = &errOut
	logDir := t.TempDir()
	code := Run(ctx, Deps{
		IO: ioDeps,
		Getenv: func(key string) string {
			if key == "TEST_API_KEY" {
				return "secret"
			}
			if key == "NO_COLOR" && ioDeps.IsTTY {
				return "1"
			}
			return ""
		},
		Now: func() time.Time {
			return time.Date(2026, 6, 18, 12, 0, 0, 123456000, time.UTC)
		},
		Waiter: waiter,
		LogDir: logDir,
	}, Options{
		Config: append([]string{"provider=test", "model=test-model"}, opts.Config...),
		Raw:    opts.Raw,
	})
	return runResult{
		stdout: out.String(),
		stderr: errOut.String(),
		log:    readOnlyLog(t, logDir),
		code:   code,
	}
}

type recordingWaiter struct {
	calls      *[]string
	ownedCalls []string
}

func (w *recordingWaiter) Start(model string) {
	w.append("waiter:start:" + model)
}

func (w *recordingWaiter) Stop() {
	w.append("waiter:stop")
}

func (w *recordingWaiter) append(call string) {
	if w.calls != nil {
		*w.calls = append(*w.calls, call)
		return
	}
	w.ownedCalls = append(w.ownedCalls, call)
}

type recordingRenderer struct {
	calls *[]string
}

func (r *recordingRenderer) Prompt() {}

func (r *recordingRenderer) Input(string) {}

func (r *recordingRenderer) Event(ev agentkit.Event) {
	r.append(fmt.Sprintf("render:event:%T", ev))
}

func (r *recordingRenderer) Usage(agentkit.Usage, agentkit.Cost, agentkit.Cost) {
	r.append("render:usage")
}

func (r *recordingRenderer) Summary(agentkit.Usage, agentkit.Cost) {}

func (r *recordingRenderer) Warning(w agentkit.Warning) {
	r.append("render:warning:" + w.Detail)
}

func (r *recordingRenderer) Error(err error) {
	r.append("render:error:" + err.Error())
}

func (r *recordingRenderer) Notice(line string) {
	r.append("render:notice:" + line)
}

func (r *recordingRenderer) append(call string) {
	*r.calls = append(*r.calls, call)
}

func newWaiterTestState(calls *[]string, provider agentkit.Provider) *state {
	return &state{
		conv:   &agentkit.Conversation{Provider: provider, Model: "test-model"},
		rend:   &recordingRenderer{calls: calls},
		waiter: &recordingWaiter{calls: calls},
	}
}

type scriptedProvider struct {
	rounds   []*agentkit.RoundTrip
	requests []agentkit.Request
}

type staticReasoning map[string]agentkit.ReasoningSpec

func (s staticReasoning) ReasoningSpec(model string) (agentkit.ReasoningSpec, bool) {
	spec, ok := s[model]
	return spec, ok
}

func (s staticReasoning) SupportedReasoning() map[string]agentkit.ReasoningSpec {
	out := make(map[string]agentkit.ReasoningSpec, len(s))
	for model, spec := range s {
		out[model] = spec
	}
	return out
}

func newScriptedProvider(rounds ...*agentkit.RoundTrip) *scriptedProvider {
	return &scriptedProvider{rounds: rounds}
}

func (p *scriptedProvider) RoundTrip(_ context.Context, req *agentkit.Request) *agentkit.RoundTrip {
	p.requests = append(p.requests, *req)
	if len(p.rounds) == 0 {
		return errorRound("unexpected provider call")
	}
	round := p.rounds[0]
	p.rounds = p.rounds[1:]
	return round
}

func (p *scriptedProvider) Name() string {
	return "test"
}

func (p *scriptedProvider) Pricing(model string) (agentkit.Pricing, bool) {
	if model != "test-model" {
		return agentkit.Pricing{}, false
	}
	return agentkit.Pricing{Tiers: []agentkit.RateTier{{
		InputUncached: 10_000,
		Output:        20_000,
	}}}, true
}

type interruptProvider struct {
	interruptStarted chan struct{}
	requests         []agentkit.Request
}

func newInterruptProvider() *interruptProvider {
	return &interruptProvider{interruptStarted: make(chan struct{})}
}

func (p *interruptProvider) RoundTrip(ctx context.Context, req *agentkit.Request) *agentkit.RoundTrip {
	p.requests = append(p.requests, *req)
	if len(p.requests) == 1 {
		return successRound("before interrupt", usageOne())
	}
	close(p.interruptStarted)
	<-ctx.Done()
	return agentkit.NewRoundTrip(agentkit.Message{}, agentkit.FinishStop, agentkit.Usage{
		InputUncached: 999,
		Output:        999,
		Total:         1998,
	}, nil, ctx.Err())
}

func (p *interruptProvider) Name() string {
	return "test"
}

func (p *interruptProvider) Pricing(model string) (agentkit.Pricing, bool) {
	if model != "test-model" {
		return agentkit.Pricing{}, false
	}
	return agentkit.Pricing{Tiers: []agentkit.RateTier{{
		InputUncached: 10_000,
		Output:        20_000,
	}}}, true
}

type blockingProvider struct {
	started chan struct{}
}

func newBlockingProvider() *blockingProvider {
	return &blockingProvider{started: make(chan struct{})}
}

func (p *blockingProvider) RoundTrip(ctx context.Context, _ *agentkit.Request) *agentkit.RoundTrip {
	close(p.started)
	<-ctx.Done()
	return agentkit.NewRoundTrip(agentkit.Message{}, agentkit.FinishStop, agentkit.Usage{}, nil, ctx.Err())
}

func (p *blockingProvider) Name() string {
	return "test"
}

func (p *blockingProvider) Pricing(model string) (agentkit.Pricing, bool) {
	if model != "test-model" {
		return agentkit.Pricing{}, false
	}
	return agentkit.Pricing{Tiers: []agentkit.RateTier{{
		InputUncached: 10_000,
		Output:        20_000,
	}}}, true
}

func successRound(text string, usage agentkit.Usage) *agentkit.RoundTrip {
	message := agentkit.Message{
		Role:   agentkit.RoleAssistant,
		Blocks: []agentkit.Block{agentkit.TextBlock{Text: text}},
	}
	return agentkit.NewRoundTrip(message, agentkit.FinishStop, usage, nil, nil)
}

func toolUseRound() *agentkit.RoundTrip {
	return toolUseRoundWithWarnings(nil)
}

func toolUseRoundWithWarnings(warnings []agentkit.Warning) *agentkit.RoundTrip {
	message := agentkit.Message{
		Role: agentkit.RoleAssistant,
		Blocks: []agentkit.Block{agentkit.ToolUseBlock{
			ID:    "toolu_1",
			Name:  "read",
			Input: json.RawMessage(`{"path":"missing.txt"}`),
		}},
	}
	return agentkit.NewRoundTrip(message, agentkit.FinishToolUse, usageOne(), warnings, nil)
}

func errorRound(message string) *agentkit.RoundTrip {
	return agentkit.NewRoundTrip(agentkit.Message{}, agentkit.FinishStop, agentkit.Usage{}, nil, errors.New(message))
}

func usageOne() agentkit.Usage {
	return agentkit.Usage{
		InputUncached: 100,
		Output:        50,
		Total:         150,
	}
}

func usageTwo() agentkit.Usage {
	return agentkit.Usage{
		InputUncached: 200,
		Output:        100,
		Total:         300,
	}
}

type reasoningWarningProvider struct {
	requests []agentkit.Request
}

func (p *reasoningWarningProvider) RoundTrip(_ context.Context, req *agentkit.Request) *agentkit.RoundTrip {
	p.requests = append(p.requests, *req)
	level, ok := req.Gen.Reasoning.Level()
	if !ok || level != "xhigh" {
		return errorRound("missing non-native reasoning level")
	}
	message := agentkit.Message{
		Role:   agentkit.RoleAssistant,
		Blocks: []agentkit.Block{agentkit.TextBlock{Text: "defaulted"}},
	}
	warnings := []agentkit.Warning{{
		Setting: "reasoning",
		Code:    agentkit.WarnReasoningUnsupported,
		Detail:  "xhigh is not supported by test-model; using high",
	}}
	return agentkit.NewRoundTrip(message, agentkit.FinishStop, usageOne(), warnings, nil)
}

func (p *reasoningWarningProvider) Name() string {
	return "test"
}

func (p *reasoningWarningProvider) Pricing(model string) (agentkit.Pricing, bool) {
	if model != "test-model" {
		return agentkit.Pricing{}, false
	}
	return agentkit.Pricing{Tiers: []agentkit.RateTier{{
		InputUncached: 10_000,
		Output:        20_000,
	}}}, true
}

func assertLogTypes(t *testing.T, log string, want []string) {
	t.Helper()
	records := decodeLogRecords(t, log)
	got := recordTypes(t, records)
	if !slices.Equal(got, want) {
		t.Fatalf("log types = %#v, want %#v\nlog:\n%s", got, want, log)
	}
}

func recordTypes(t *testing.T, records []map[string]any) []string {
	t.Helper()
	got := make([]string, 0, len(records))
	for _, record := range records {
		value, ok := record["type"].(string)
		if !ok {
			t.Fatalf("record missing string type: %#v", record)
		}
		got = append(got, value)
	}
	return got
}

func awaitRun(t *testing.T, done <-chan int) int {
	t.Helper()
	select {
	case code := <-done:
		return code
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not return")
		return -1
	}
}

func awaitRunResult(t *testing.T, done <-chan runResult) runResult {
	t.Helper()
	select {
	case result := <-done:
		return result
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not return")
		return runResult{}
	}
}

func readOnlyLog(t *testing.T, logDir string) string {
	t.Helper()
	matches, err := filepath.Glob(filepath.Join(logDir, "*.jsonl"))
	if err != nil {
		t.Fatalf("checking log dir: %v", err)
	}
	if len(matches) != 1 {
		t.Fatalf("log files = %v, want exactly one", matches)
	}
	content, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatalf("reading log file: %v", err)
	}
	return string(content)
}

func decodeLogRecords(t *testing.T, log string) []map[string]any {
	t.Helper()
	lines := strings.Split(strings.TrimSpace(log), "\n")
	if len(lines) == 1 && lines[0] == "" {
		return nil
	}
	records := make([]map[string]any, 0, len(lines))
	for _, line := range lines {
		var record map[string]any
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			t.Fatalf("invalid JSONL record %q: %v", line, err)
		}
		records = append(records, record)
	}
	return records
}

func normalizedLogRecords(t *testing.T, log string) []map[string]any {
	t.Helper()
	records := decodeLogRecords(t, log)
	for _, record := range records {
		delete(record, "time")
	}
	return records
}
