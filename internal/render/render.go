package render

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/ikigenba/agentkit"
)

// Renderer presents the input prompt, streamed events, outcome, and spend.
type Renderer interface {
	Prompt()
	Input(text string)
	Event(ev agentkit.Event)
	Usage(turn agentkit.Usage, turnCost, total agentkit.Cost)
	Summary(total agentkit.Usage, totalCost agentkit.Cost)
	Warning(w agentkit.Warning)
	Error(err error)
	Notice(line string)
}

// NewDecorated returns the operator-facing renderer.
func NewDecorated(out io.Writer, color, tty bool) Renderer {
	return &decoratedRenderer{out: out, color: color, tty: tty}
}

// NewRaw returns the machine-readable JSONL renderer.
func NewRaw(out io.Writer) Renderer {
	return rawRenderer{out: out}
}

type decoratedRenderer struct {
	out       io.Writer
	color     bool
	tty       bool
	streaming bool
}

func (r *decoratedRenderer) Prompt() {
	r.finishStream()
	if !r.tty {
		return
	}
	fmt.Fprintf(r.out, "%syou ›%s ", r.paint(ansiBold), r.paint(ansiReset))
}

func (r *decoratedRenderer) Input(string) {
}

func (r *decoratedRenderer) Event(ev agentkit.Event) {
	switch ev := ev.(type) {
	case agentkit.TextDelta:
		if !r.streaming {
			fmt.Fprintf(r.out, "%sassistant ›%s ", r.paint(ansiGreen), r.paint(ansiReset))
			r.streaming = true
		}
		fmt.Fprint(r.out, ev.Text)
	case agentkit.ReasoningDelta:
		if !r.streaming {
			fmt.Fprintf(r.out, "%sreasoning ›%s ", r.paint(ansiDim), r.paint(ansiReset))
			r.streaming = true
		}
		fmt.Fprint(r.out, ev.Text)
	case agentkit.MessageDone:
		r.finishStream()
	case agentkit.ToolUse:
		r.finishStream()
		fmt.Fprintf(r.out, "%stool call ›%s %s %s\n", r.paint(ansiCyan), r.paint(ansiReset), ev.Name, string(ev.Input))
	case agentkit.ToolResult:
		r.finishStream()
		label := "tool result ›"
		if ev.IsError {
			label = "tool error ›"
		}
		fmt.Fprintf(r.out, "%s%s%s %s: %s\n", r.resultColor(ev.IsError), label, r.paint(ansiReset), ev.Name, ev.Output)
	}
}

func (r *decoratedRenderer) Usage(turn agentkit.Usage, turnCost, total agentkit.Cost) {
}

func (r *decoratedRenderer) Summary(total agentkit.Usage, totalCost agentkit.Cost) {
	r.finishStream()
	fmt.Fprintln(r.out, "summary")
	fmt.Fprintf(r.out, "· tokens  in=%d cache(r=%d w=%d) out=%d reasoning=%d total=%d\n",
		total.InputUncached,
		total.CacheReadInput,
		total.CacheWriteInput,
		total.Output,
		total.ReasoningOutput,
		total.Total,
	)
	fmt.Fprintf(r.out, "· cost     $%.6f session\n", totalCost.USD())
}

func (r *decoratedRenderer) Warning(w agentkit.Warning) {
	r.finishStream()
	fmt.Fprintf(r.out, "%swarning ›%s %s: %s\n", r.paint(ansiYellow), r.paint(ansiReset), w.Setting, w.Detail)
}

func (r *decoratedRenderer) Error(err error) {
	r.finishStream()
	fmt.Fprintf(r.out, "%serror ›%s %v\n", r.paint(ansiRed), r.paint(ansiReset), err)
}

func (r *decoratedRenderer) Notice(line string) {
	r.finishStream()
	fmt.Fprintf(r.out, "notice › %s\n", line)
}

func (r *decoratedRenderer) finishStream() {
	if r.streaming {
		fmt.Fprintln(r.out)
		r.streaming = false
	}
}

func (r *decoratedRenderer) resultColor(isError bool) string {
	if isError {
		return r.paint(ansiRed)
	}
	return r.paint(ansiYellow)
}

func (r *decoratedRenderer) paint(code string) string {
	if !r.color {
		return ""
	}
	return code
}

type rawRenderer struct {
	out io.Writer
}

func (r rawRenderer) Prompt() {
}

func (r rawRenderer) Input(text string) {
	r.write(rawPrompt{Type: "prompt", Text: text})
}

func (r rawRenderer) Event(ev agentkit.Event) {
	switch ev := ev.(type) {
	case agentkit.MessageDone:
		r.write(rawMessageDone{Type: "message_done", Message: ev.Message})
	case agentkit.ToolUse:
		r.write(rawToolUse{Type: "tool_use", ToolUse: ev})
	case agentkit.ToolResult:
		r.write(rawToolResult{Type: "tool_result", ToolResult: ev})
	}
}

func (r rawRenderer) Usage(turn agentkit.Usage, turnCost, total agentkit.Cost) {
	r.write(rawUsage{
		Type:           "usage",
		Usage:          turn,
		TurnCostUSD:    formatUSD(turnCost),
		SessionCostUSD: formatUSD(total),
	})
}

func (r rawRenderer) Summary(total agentkit.Usage, totalCost agentkit.Cost) {
	r.write(rawSummary{
		Type:           "summary",
		Usage:          total,
		SessionCostUSD: formatUSD(totalCost),
	})
}

func (r rawRenderer) Warning(w agentkit.Warning) {
	r.write(rawWarning{Type: "warning", Warning: w})
}

func (r rawRenderer) Error(err error) {
	r.write(rawError{Type: "error", Error: err.Error()})
}

func (r rawRenderer) Notice(line string) {
	r.write(rawNotice{Type: "notice", Text: line})
}

func (r rawRenderer) write(v any) {
	_ = json.NewEncoder(r.out).Encode(v)
}

type rawPrompt struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type rawMessageDone struct {
	Type    string           `json:"type"`
	Message agentkit.Message `json:"message"`
}

type rawToolUse struct {
	Type    string           `json:"type"`
	ToolUse agentkit.ToolUse `json:"tool_use"`
}

type rawToolResult struct {
	Type       string              `json:"type"`
	ToolResult agentkit.ToolResult `json:"tool_result"`
}

type rawUsage struct {
	Type           string         `json:"type"`
	Usage          agentkit.Usage `json:"usage"`
	TurnCostUSD    string         `json:"turn_cost_usd"`
	SessionCostUSD string         `json:"session_cost_usd"`
}

type rawSummary struct {
	Type           string         `json:"type"`
	Usage          agentkit.Usage `json:"usage"`
	SessionCostUSD string         `json:"session_cost_usd"`
}

type rawWarning struct {
	Type string `json:"type"`
	agentkit.Warning
}

type rawError struct {
	Type  string `json:"type"`
	Error string `json:"error"`
}

type rawNotice struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

func formatUSD(cost agentkit.Cost) string {
	return fmt.Sprintf("%.6f", cost.USD())
}

const (
	ansiReset  = "\x1b[0m"
	ansiBold   = "\x1b[1m"
	ansiDim    = "\x1b[2m"
	ansiGreen  = "\x1b[32m"
	ansiYellow = "\x1b[33m"
	ansiCyan   = "\x1b[36m"
	ansiRed    = "\x1b[31m"
)
