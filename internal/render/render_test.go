package render

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/ikigenba/agentkit"
)

var update = flag.Bool("update", false, "update golden files")

func TestDecoratedGoldenRendersKindsUsageCostsAndToolErrors(t *testing.T) {
	// R-LL9K-SKDQ
	// R-LRD2-PF37
	// R-ONJY-6PJG
	// R-OORU-KHA5
	var buf bytes.Buffer
	render := NewDecorated(&buf, false)

	render.Prompt("hello")
	render.Event(agentkit.ReasoningDelta{Text: "checking"})
	render.Event(agentkit.MessageDone{})
	render.Event(agentkit.TextDelta{Text: "Hi there"})
	render.Event(agentkit.MessageDone{})
	render.Event(agentkit.ToolUse{ID: "toolu_1", Name: "read", Input: json.RawMessage(`{"path":"missing.txt"}`)})
	render.Event(agentkit.ToolResult{ID: "toolu_1", Name: "read", Output: "contents"})
	render.Event(agentkit.ToolResult{ID: "toolu_2", Name: "read", Output: "open missing.txt: no such file", IsError: true})
	render.Usage(turnUsage(), agentkit.Cost(1_234_000), agentkit.Cost(5_678_000))
	render.Summary(summaryUsage(), agentkit.Cost(6_789_000))

	got := buf.String()
	assertGolden(t, "decorated.golden", got)
	for _, want := range []string{
		"you › hello",
		"reasoning › checking",
		"assistant › Hi there",
		`tool call › read {"path":"missing.txt"}`,
		"tool result › read: contents",
		"tool error › read: open missing.txt: no such file",
		"in=123 cache(r=10 w=5) out=456 reasoning=78 total=999",
		"$0.001234 turn   $0.005678 session",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("decorated output missing %q:\n%s", want, got)
		}
	}
}

func TestDecoratedStreamsDeltasIncrementally(t *testing.T) {
	// R-LMHH-6C4F
	var buf bytes.Buffer
	render := NewDecorated(&buf, false)

	render.Event(agentkit.TextDelta{Text: "Hel"})
	if got := buf.String(); got != "assistant › Hel" {
		t.Fatalf("after first TextDelta = %q, want bytes written immediately", got)
	}
	render.Event(agentkit.TextDelta{Text: "lo"})
	if got := buf.String(); got != "assistant › Hello" {
		t.Fatalf("after second TextDelta = %q, want appended bytes", got)
	}

	buf.Reset()
	render = NewDecorated(&buf, false)
	render.Event(agentkit.ReasoningDelta{Text: "check"})
	if got := buf.String(); got != "reasoning › check" {
		t.Fatalf("after ReasoningDelta = %q, want bytes written immediately", got)
	}
}

func TestDecoratedColorIsControlledByConstructorFlag(t *testing.T) {
	// R-LNPD-K3V4
	var color bytes.Buffer
	colorRender := NewDecorated(&color, true)
	colorRender.Prompt("hello")
	colorRender.Event(agentkit.TextDelta{Text: "Hi"})
	colorRender.Event(agentkit.ToolResult{Name: "read", Output: "missing", IsError: true})
	colorRender.Error(assertErr("boom"))

	gotColor := color.String()
	if !strings.Contains(gotColor, "\x1b[") {
		t.Fatalf("color output = %q, want ANSI escape sequence", gotColor)
	}
	assertGolden(t, "decorated_color.golden", visibleANSI(gotColor))

	var plain bytes.Buffer
	plainRender := NewDecorated(&plain, false)
	plainRender.Prompt("hello")
	plainRender.Event(agentkit.TextDelta{Text: "Hi"})
	plainRender.Event(agentkit.ToolResult{Name: "read", Output: "missing", IsError: true})
	plainRender.Error(assertErr("boom"))

	if strings.Contains(plain.String(), "\x1b[") {
		t.Fatalf("plain output = %q, want no ANSI escape sequence", plain.String())
	}
	assertGolden(t, "decorated_colorless.golden", plain.String())
}

func TestRawJSONLGoldenSkipsDeltasAndCarriesUsageSummaryAndToolErrors(t *testing.T) {
	// R-LOX9-XVLT
	// R-LRD2-PF37
	// R-OR7N-C0RJ
	// R-OW38-V3QB
	var buf bytes.Buffer
	render := NewRaw(&buf)

	render.Prompt("hello")
	render.Event(agentkit.TextDelta{Text: "ignored"})
	render.Event(agentkit.ReasoningDelta{Text: "ignored"})
	render.Event(agentkit.MessageDone{Message: agentkit.Message{
		Role: agentkit.RoleAssistant,
		Blocks: []agentkit.Block{
			agentkit.TextBlock{Text: "Hi there"},
			agentkit.ReasoningBlock{
				Opaque:    json.RawMessage(`{"signature":"opaque"}`),
				Summary:   "checking",
				BoundToID: "toolu_1",
			},
		},
	}})
	render.Event(agentkit.ToolUse{ID: "toolu_1", Name: "read", Input: json.RawMessage(`{"path":"missing.txt"}`)})
	render.Event(agentkit.ToolResult{ID: "toolu_1", Name: "read", Output: "open missing.txt: no such file", IsError: true})
	render.Usage(turnUsage(), agentkit.Cost(1_234_000), agentkit.Cost(5_678_000))
	render.Summary(summaryUsage(), agentkit.Cost(6_789_000))

	got := buf.String()
	if strings.Contains(got, "\x1b[") {
		t.Fatalf("raw output contains ANSI escape sequence: %q", got)
	}
	assertJSONLines(t, got)
	assertGolden(t, "raw.golden", got)

	records := decodeRawRecords(t, got)
	gotTypes := make([]string, 0, len(records))
	for _, record := range records {
		gotTypes = append(gotTypes, record["type"].(string))
	}
	wantTypes := []string{"prompt", "message_done", "tool_use", "tool_result", "usage", "summary"}
	if !reflect.DeepEqual(gotTypes, wantTypes) {
		t.Fatalf("raw record types = %v, want %v", gotTypes, wantTypes)
	}
	result := records[3]["tool_result"].(map[string]any)
	if result["IsError"] != true {
		t.Fatalf("raw tool_result IsError = %v, want true", result["IsError"])
	}
	usage := records[4]
	if usage["turn_cost_usd"] != "0.001234" || usage["session_cost_usd"] != "0.005678" {
		t.Fatalf("raw usage costs = %#v, want turn/session USD", usage)
	}
	summary := records[5]
	if summary["session_cost_usd"] != "0.006789" {
		t.Fatalf("raw summary cost = %#v, want session USD", summary)
	}
}

func TestWarningGoldenRendersDistinctTreatmentAndRawCarriesFields(t *testing.T) {
	// R-G5FW-SS92
	warning := agentkit.Warning{
		Setting: "reasoning",
		Code:    agentkit.WarnReasoningUnsupported,
		Detail:  "xhigh is not supported by test-model; using high",
	}

	var decorated bytes.Buffer
	decoratedRender := NewDecorated(&decorated, false)
	decoratedRender.Warning(warning)
	decoratedRender.Error(assertErr("turn failed"))
	gotDecorated := decorated.String()
	assertGolden(t, "warning_decorated.golden", gotDecorated)
	if !strings.Contains(gotDecorated, "warning › reasoning: xhigh is not supported") {
		t.Fatalf("decorated warning = %q, want warning treatment with setting and detail", gotDecorated)
	}
	if !strings.Contains(gotDecorated, "error › turn failed") {
		t.Fatalf("decorated output = %q, want error treatment for comparison", gotDecorated)
	}
	if strings.Index(gotDecorated, "warning ›") == strings.Index(gotDecorated, "error ›") {
		t.Fatalf("decorated output = %q, warning and error treatments were not distinct", gotDecorated)
	}

	var raw bytes.Buffer
	rawRender := NewRaw(&raw)
	rawRender.Warning(warning)
	gotRaw := raw.String()
	assertJSONLines(t, gotRaw)
	assertGolden(t, "warning_raw.golden", gotRaw)

	records := decodeRawRecords(t, gotRaw)
	if len(records) != 1 {
		t.Fatalf("raw warning record count = %d, want 1", len(records))
	}
	record := records[0]
	if record["type"] != "warning" || record["Setting"] != "reasoning" || record["Code"] != float64(agentkit.WarnReasoningUnsupported) || record["Detail"] != warning.Detail {
		t.Fatalf("raw warning record = %#v, want verbatim Setting/Code/Detail", record)
	}
}

func turnUsage() agentkit.Usage {
	return agentkit.Usage{
		InputUncached:   123,
		CacheReadInput:  10,
		CacheWriteInput: 5,
		Output:          456,
		ReasoningOutput: 78,
		Total:           999,
	}
}

func summaryUsage() agentkit.Usage {
	return agentkit.Usage{
		InputUncached:   223,
		CacheReadInput:  20,
		CacheWriteInput: 15,
		Output:          556,
		ReasoningOutput: 88,
		Total:           1111,
	}
}

func assertGolden(t *testing.T, name string, got string) {
	t.Helper()
	path := filepath.Join("testdata", name)
	if *update {
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatalf("update golden %s: %v", name, err)
		}
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden %s: %v", name, err)
	}
	if string(want) != got {
		t.Fatalf("%s mismatch\nwant:\n%s\ngot:\n%s", name, string(want), got)
	}
}

func assertJSONLines(t *testing.T, got string) {
	t.Helper()
	scanner := bufio.NewScanner(strings.NewReader(got))
	for scanner.Scan() {
		var record map[string]any
		if err := json.Unmarshal(scanner.Bytes(), &record); err != nil {
			t.Fatalf("invalid JSONL line %q: %v", scanner.Text(), err)
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("scan JSONL: %v", err)
	}
}

func decodeRawRecords(t *testing.T, got string) []map[string]any {
	t.Helper()
	var records []map[string]any
	scanner := bufio.NewScanner(strings.NewReader(got))
	for scanner.Scan() {
		var record map[string]any
		if err := json.Unmarshal(scanner.Bytes(), &record); err != nil {
			t.Fatalf("decode JSONL line %q: %v", scanner.Text(), err)
		}
		records = append(records, record)
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("scan JSONL: %v", err)
	}
	return records
}

func visibleANSI(s string) string {
	return strings.ReplaceAll(s, "\x1b", "<ESC>")
}

type assertErr string

func (e assertErr) Error() string { return string(e) }
