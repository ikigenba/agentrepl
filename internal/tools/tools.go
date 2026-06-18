package tools

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/ikigenba/agentkit"
)

type bashInput struct {
	Command string `json:"command" jsonschema:"required" jsonschema_description:"Shell command to run with bash -lc."`
}

type readInput struct {
	Path string `json:"path" jsonschema:"required" jsonschema_description:"Path to read, resolved relative to the current working directory."`
}

type writeInput struct {
	Path    string `json:"path" jsonschema:"required" jsonschema_description:"Path to write, resolved relative to the current working directory."`
	Content string `json:"content" jsonschema:"required" jsonschema_description:"Content to write to the file."`
}

type editInput struct {
	Path string `json:"path" jsonschema:"required" jsonschema_description:"Path to edit, resolved relative to the current working directory."`
	Old  string `json:"old" jsonschema:"required" jsonschema_description:"Text to replace. All occurrences are replaced."`
	New  string `json:"new" jsonschema:"required" jsonschema_description:"Replacement text."`
}

// All returns the four built-in tools, operating relative to the process
// working directory.
func All() []agentkit.Tool {
	return []agentkit.Tool{
		agentkit.NewTool("bash", "Run a shell command with bash -lc and return combined stdout and stderr.", runBash),
		agentkit.NewTool("read", "Read a local file.", readFile),
		agentkit.NewTool("write", "Write a local file, creating or truncating it.", writeFile),
		agentkit.NewTool("edit", "Replace all occurrences of text in a local file.", editFile),
	}
}

func runBash(ctx context.Context, in bashInput) (string, error) {
	cmd := exec.CommandContext(ctx, "bash", "-lc", in.Command)
	output, err := cmd.CombinedOutput()
	if err == nil {
		return string(output), nil
	}

	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		return string(output), err
	}

	text := string(output)
	if text != "" && !strings.HasSuffix(text, "\n") {
		text += "\n"
	}
	return fmt.Sprintf("%s[exit status %d]", text, exitErr.ExitCode()), nil
}

func readFile(_ context.Context, in readInput) (string, error) {
	content, err := os.ReadFile(in.Path)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

func writeFile(_ context.Context, in writeInput) (string, error) {
	if err := os.WriteFile(in.Path, []byte(in.Content), 0o644); err != nil {
		return "", err
	}
	return fmt.Sprintf("wrote %s", in.Path), nil
}

func editFile(_ context.Context, in editInput) (string, error) {
	content, err := os.ReadFile(in.Path)
	if err != nil {
		return "", err
	}

	text := string(content)
	count := strings.Count(text, in.Old)
	if count == 0 {
		return "", fmt.Errorf("old text not found in %s", in.Path)
	}

	updated := strings.ReplaceAll(text, in.Old, in.New)
	if err := os.WriteFile(in.Path, []byte(updated), 0o644); err != nil {
		return "", err
	}
	return fmt.Sprintf("replaced %d occurrence(s) in %s", count, in.Path), nil
}
