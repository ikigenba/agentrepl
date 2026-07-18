package tools

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ikigenba/agentkit"
)

func TestAllReturnsFourNamedToolsWithValidSchemas(t *testing.T) {
	// R-NHBW-446N
	got := All()
	if len(got) != 4 {
		t.Fatalf("len(All()) = %d, want 4", len(got))
	}

	wantNames := []string{"bash", "read", "write", "edit"}
	for i, tool := range got {
		if tool.Name() != wantNames[i] {
			t.Fatalf("All()[%d].Name() = %q, want %q", i, tool.Name(), wantNames[i])
		}
		if tool.Description() == "" {
			t.Fatalf("%s Description() is empty", tool.Name())
		}
		assertObjectSchema(t, tool)
	}
}

func TestBashReturnsCombinedOutputAndExitStatusWithoutGoError(t *testing.T) {
	// R-NIJS-HVXC
	tool := toolByName(t, "bash")
	output, err := tool.Call(context.Background(), json.RawMessage(`{"command":"printf 'out\n'; printf 'err\n' >&2; exit 7"}`))
	if err != nil {
		t.Fatalf("bash Call() error = %v, want nil", err)
	}
	if !strings.Contains(output, "out\n") {
		t.Fatalf("bash output = %q, want stdout preserved", output)
	}
	if !strings.Contains(output, "err\n") {
		t.Fatalf("bash output = %q, want stderr preserved", output)
	}
	if !strings.Contains(output, "[exit status 7]") {
		t.Fatalf("bash output = %q, want exit status noted", output)
	}
}

func TestReadReturnsContentsAndMissingFileFeedsBackInBand(t *testing.T) {
	// R-NKZL-9FEQ
	inDir(t, t.TempDir())
	if err := os.WriteFile("present.txt", []byte("hello"), 0o644); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	tool := toolByName(t, "read")
	output, err := tool.Call(context.Background(), json.RawMessage(`{"path":"present.txt"}`))
	if err != nil {
		t.Fatalf("read existing file error = %v", err)
	}
	if output != "hello" {
		t.Fatalf("read output = %q, want file contents", output)
	}

	result := runToolUse(t, tool, json.RawMessage(`{"path":"missing.txt"}`))
	if !result.IsError {
		t.Fatalf("ToolResult.IsError = false, want true for missing file")
	}
	if !strings.Contains(result.Output, "missing.txt") {
		t.Fatalf("ToolResult.Output = %q, want missing path in error", result.Output)
	}
}

func TestWriteCreatesAndOverwritesFile(t *testing.T) {
	// R-NM7H-N75F
	inDir(t, t.TempDir())

	tool := toolByName(t, "write")
	if _, err := tool.Call(context.Background(), json.RawMessage(`{"path":"note.txt","content":"first"}`)); err != nil {
		t.Fatalf("write create error = %v", err)
	}
	assertFileContent(t, "note.txt", "first")

	if _, err := tool.Call(context.Background(), json.RawMessage(`{"path":"note.txt","content":"second"}`)); err != nil {
		t.Fatalf("write overwrite error = %v", err)
	}
	assertFileContent(t, "note.txt", "second")
}

func TestEditReplacesAllAndAbsentOldFeedsBackInBand(t *testing.T) {
	// R-NNFE-0YW4
	inDir(t, t.TempDir())
	if err := os.WriteFile("story.txt", []byte("red blue red"), 0o644); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	tool := toolByName(t, "edit")
	output, err := tool.Call(context.Background(), json.RawMessage(`{"path":"story.txt","old":"red","new":"green"}`))
	if err != nil {
		t.Fatalf("edit existing text error = %v", err)
	}
	if !strings.Contains(output, "2") {
		t.Fatalf("edit output = %q, want replacement count", output)
	}
	assertFileContent(t, "story.txt", "green blue green")

	result := runToolUse(t, tool, json.RawMessage(`{"path":"story.txt","old":"red","new":"gold"}`))
	if !result.IsError {
		t.Fatalf("ToolResult.IsError = false, want true for absent old text")
	}
	if !strings.Contains(result.Output, "old text not found") {
		t.Fatalf("ToolResult.Output = %q, want absent-old error", result.Output)
	}
}

func TestToolsResolveRelativeToProcessWorkingDirectory(t *testing.T) {
	// R-NONA-EQMT
	dir := t.TempDir()
	inDir(t, dir)
	if err := os.WriteFile("read.txt", []byte("from cwd"), 0o644); err != nil {
		t.Fatalf("os.WriteFile(read.txt) error = %v", err)
	}
	if err := os.WriteFile("edit.txt", []byte("alpha alpha"), 0o644); err != nil {
		t.Fatalf("os.WriteFile(edit.txt) error = %v", err)
	}

	read := toolByName(t, "read")
	if output, err := read.Call(context.Background(), json.RawMessage(`{"path":"read.txt"}`)); err != nil || output != "from cwd" {
		t.Fatalf("read relative output/error = %q/%v, want cwd file contents", output, err)
	}

	write := toolByName(t, "write")
	if _, err := write.Call(context.Background(), json.RawMessage(`{"path":"write.txt","content":"written in cwd"}`)); err != nil {
		t.Fatalf("write relative error = %v", err)
	}
	assertFileContent(t, filepath.Join(dir, "write.txt"), "written in cwd")

	edit := toolByName(t, "edit")
	if _, err := edit.Call(context.Background(), json.RawMessage(`{"path":"edit.txt","old":"alpha","new":"beta"}`)); err != nil {
		t.Fatalf("edit relative error = %v", err)
	}
	assertFileContent(t, filepath.Join(dir, "edit.txt"), "beta beta")

	bash := toolByName(t, "bash")
	output, err := bash.Call(context.Background(), json.RawMessage(`{"command":"printf '%s' \"$(pwd)\"; test -f read.txt"}`))
	if err != nil {
		t.Fatalf("bash relative cwd error = %v", err)
	}
	if output != dir {
		t.Fatalf("bash pwd output = %q, want %q", output, dir)
	}
}

func assertObjectSchema(t *testing.T, tool agentkit.Tool) {
	t.Helper()

	var schema struct {
		Type       string `json:"type"`
		Properties map[string]struct {
			Type        string `json:"type"`
			Description string `json:"description"`
		} `json:"properties"`
	}
	if err := json.Unmarshal(tool.JSONSchema(), &schema); err != nil {
		t.Fatalf("%s JSONSchema() did not unmarshal: %v", tool.Name(), err)
	}
	if schema.Type != "object" {
		t.Fatalf("%s schema type = %q, want object", tool.Name(), schema.Type)
	}
	if len(schema.Properties) == 0 {
		t.Fatalf("%s schema properties empty", tool.Name())
	}
	for name, property := range schema.Properties {
		if property.Type != "string" {
			t.Fatalf("%s schema property %s type = %q, want string", tool.Name(), name, property.Type)
		}
		if property.Description == "" {
			t.Fatalf("%s schema property %s description is empty", tool.Name(), name)
		}
	}
}

func toolByName(t *testing.T, name string) agentkit.Tool {
	t.Helper()
	for _, tool := range All() {
		if tool.Name() == name {
			return tool
		}
	}
	t.Fatalf("tool %q not found", name)
	return nil
}

func inDir(t *testing.T, dir string) {
	t.Helper()

	previous, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd() error = %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("os.Chdir(%q) error = %v", dir, err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(previous); err != nil {
			t.Fatalf("restore cwd to %q: %v", previous, err)
		}
	})
}

func assertFileContent(t *testing.T, path, want string) {
	t.Helper()

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("os.ReadFile(%q) error = %v", path, err)
	}
	if string(content) != want {
		t.Fatalf("content of %s = %q, want %q", path, string(content), want)
	}
}

func runToolUse(t *testing.T, tool agentkit.Tool, input json.RawMessage) agentkit.ToolResult {
	t.Helper()

	provider := &toolResultProvider{name: tool.Name(), input: input}
	conv := &agentkit.Conversation{
		Provider: provider,
		Model:    "fake-model",
		Tools:    []agentkit.Tool{tool},
	}
	stream := conv.Send(context.Background(), "use tool")

	var result *agentkit.ToolResult
	for event := range stream.Events() {
		if toolResult, ok := event.(agentkit.ToolResult); ok {
			result = &toolResult
		}
	}
	if err := stream.Err(); err != nil {
		t.Fatalf("stream.Err() = %v, want nil", err)
	}
	if result == nil {
		t.Fatalf("stream did not emit ToolResult")
	}
	if len(provider.requests) != 2 {
		t.Fatalf("provider requests = %d, want 2", len(provider.requests))
	}
	return *result
}

type toolResultProvider struct {
	name     string
	input    json.RawMessage
	requests []*agentkit.Request
}

func (p *toolResultProvider) Name() string {
	return "fake"
}

func (p *toolResultProvider) Pricing(model string) (agentkit.Pricing, bool) {
	return agentkit.Pricing{Tiers: []agentkit.RateTier{{}}}, model == "fake-model"
}

func (p *toolResultProvider) RoundTrip(_ context.Context, req *agentkit.Request) *agentkit.RoundTrip {
	p.requests = append(p.requests, req)
	switch len(p.requests) {
	case 1:
		return agentkit.NewRoundTrip(agentkit.Message{
			Role: agentkit.RoleAssistant,
			Blocks: []agentkit.Block{agentkit.ToolUseBlock{
				ID:    "toolu_test",
				Name:  p.name,
				Input: p.input,
			}},
		}, agentkit.FinishToolUse, agentkit.Usage{}, nil, nil, 0, true)
	case 2:
		result := lastToolResult(req.Messages)
		if result == nil {
			return agentkit.NewRoundTrip(agentkit.Message{}, agentkit.FinishOther, agentkit.Usage{}, nil, errors.New("missing tool result"), 0, false)
		}
		return agentkit.NewRoundTrip(agentkit.Message{
			Role:   agentkit.RoleAssistant,
			Blocks: []agentkit.Block{agentkit.TextBlock{Text: "done"}},
		}, agentkit.FinishStop, agentkit.Usage{}, nil, nil, 0, true)
	default:
		return agentkit.NewRoundTrip(agentkit.Message{}, agentkit.FinishOther, agentkit.Usage{}, nil, errors.New("unexpected extra request"), 0, false)
	}
}

func lastToolResult(messages []agentkit.Message) *agentkit.ToolResultBlock {
	for i := len(messages) - 1; i >= 0; i-- {
		for _, block := range messages[i].Blocks {
			if result, ok := block.(agentkit.ToolResultBlock); ok {
				return &result
			}
		}
	}
	return nil
}
