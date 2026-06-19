package render

import (
	"encoding/json"
	"fmt"
	"io"
	"sync"
	"time"

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

// NewLiveWaiter returns the TTY-only wait status line driver.
func NewLiveWaiter(out io.Writer, color bool) *LiveWaiter {
	return &LiveWaiter{out: out, color: color}
}

// LiveWaiter paints and erases the ephemeral wait status line.
type LiveWaiter struct {
	out   io.Writer
	color bool

	mu      sync.Mutex
	stop    chan struct{}
	done    chan struct{}
	running bool
	painted bool
}

func (w *LiveWaiter) Start(model string) {
	w.mu.Lock()
	if w.running {
		w.mu.Unlock()
		return
	}
	w.stop = make(chan struct{})
	w.done = make(chan struct{})
	w.running = true
	w.painted = false
	stop := w.stop
	done := w.done
	start := time.Now()
	w.mu.Unlock()

	go w.run(model, start, stop, done)
}

func (w *LiveWaiter) Stop() {
	w.mu.Lock()
	if !w.running {
		w.mu.Unlock()
		return
	}
	stop := w.stop
	done := w.done
	w.mu.Unlock()

	close(stop)
	<-done

	w.mu.Lock()
	if w.painted {
		fmt.Fprint(w.out, "\r\x1b[2K")
	}
	w.stop = nil
	w.done = nil
	w.running = false
	w.painted = false
	w.mu.Unlock()
}

func (w *LiveWaiter) run(model string, start time.Time, stop <-chan struct{}, done chan<- struct{}) {
	defer close(done)

	timer := time.NewTimer(2 * time.Second)
	defer timer.Stop()

	select {
	case <-stop:
		return
	case <-timer.C:
		w.paint(model, time.Since(start))
	}

	ticker := time.NewTicker(80 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			w.paint(model, time.Since(start))
		}
	}
}

func (w *LiveWaiter) paint(model string, elapsed time.Duration) {
	w.mu.Lock()
	fmt.Fprint(w.out, "\r\x1b[2K", waitLine(model, elapsed, w.color))
	w.painted = true
	w.mu.Unlock()
}

func waitLine(model string, elapsed time.Duration, color bool) string {
	line := fmt.Sprintf("waiting for %s (%s)", model, formatElapsed(elapsed))
	if !color {
		return line
	}
	return ansiBrightBlack + line + ansiReset
}

func formatElapsed(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	total := int(d / time.Second)
	hours := total / 3600
	minutes := (total % 3600) / 60
	seconds := total % 60
	if hours > 0 {
		return fmt.Sprintf("%dh%dm%ds", hours, minutes, seconds)
	}
	if minutes > 0 {
		return fmt.Sprintf("%dm%ds", minutes, seconds)
	}
	return fmt.Sprintf("%ds", seconds)
}

type decoratedRenderer struct {
	out        io.Writer
	color      bool
	tty        bool
	streaming  bool
	drewBlock  bool
	lastPrompt bool
}

func (r *decoratedRenderer) Prompt() {
	r.finishStream()
	if !r.tty {
		return
	}
	r.startBlock(true)
	fmt.Fprintf(r.out, "%syou ›%s ", r.paint(ansiBold), r.paint(ansiReset))
}

func (r *decoratedRenderer) Input(string) {
}

func (r *decoratedRenderer) Event(ev agentkit.Event) {
	switch ev := ev.(type) {
	case agentkit.TextDelta:
		if !r.streaming {
			r.startBlock(false)
			fmt.Fprintf(r.out, "%s%sassistant ›%s %s", r.paint(ansiBold), r.paint(ansiBrightBlue), r.paint(ansiReset), r.paint(ansiBrightBlue))
			r.streaming = true
		}
		fmt.Fprint(r.out, ev.Text)
	case agentkit.ReasoningDelta:
		if !r.streaming {
			r.startBlock(false)
			fmt.Fprintf(r.out, "%sreasoning › %s", r.paint(ansiDim), r.paint(ansiReset))
			fmt.Fprint(r.out, r.paint(ansiDim))
			r.streaming = true
		}
		fmt.Fprint(r.out, ev.Text)
	case agentkit.MessageDone:
		r.finishStream()
	case agentkit.ToolUse:
		r.finishStream()
		r.startBlock(false)
		fmt.Fprintf(r.out, "%stool call › %s %s%s\n", r.paint(ansiBrightBlack), ev.Name, trimOneTrailingNewline(string(ev.Input)), r.paint(ansiReset))
	case agentkit.ToolResult:
		r.finishStream()
		label := "tool result ›"
		if ev.IsError {
			label = "tool error ›"
		}
		r.startBlock(false)
		fmt.Fprintf(r.out, "%s%s %s: %s%s\n", r.resultColor(ev.IsError), label, ev.Name, trimOneTrailingNewline(ev.Output), r.paint(ansiReset))
	}
}

func (r *decoratedRenderer) Usage(turn agentkit.Usage, turnCost, total agentkit.Cost) {
}

func (r *decoratedRenderer) Summary(total agentkit.Usage, totalCost agentkit.Cost) {
	r.finishStream()
	r.startBlock(false)
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
	r.startBlock(false)
	fmt.Fprintf(r.out, "%swarning ›%s %s: %s\n", r.paint(ansiYellow), r.paint(ansiReset), w.Setting, w.Detail)
}

func (r *decoratedRenderer) Error(err error) {
	r.finishStream()
	r.startBlock(false)
	fmt.Fprintf(r.out, "%serror ›%s %v\n", r.paint(ansiRed), r.paint(ansiReset), err)
}

func (r *decoratedRenderer) Notice(line string) {
	r.finishStream()
	r.startBlock(false)
	fmt.Fprintf(r.out, "notice › %s\n", line)
}

func (r *decoratedRenderer) finishStream() {
	if r.streaming {
		fmt.Fprintln(r.out, r.paint(ansiReset))
		r.streaming = false
	}
}

func (r *decoratedRenderer) startBlock(prompt bool) {
	if !r.drewBlock {
		r.drewBlock = true
		r.lastPrompt = prompt
		return
	}
	if !(prompt && r.lastPrompt) {
		fmt.Fprintln(r.out)
	}
	r.lastPrompt = prompt
}

func (r *decoratedRenderer) resultColor(isError bool) string {
	if isError {
		return r.paint(ansiRed)
	}
	return r.paint(ansiBrightBlack)
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

func trimOneTrailingNewline(s string) string {
	if len(s) == 0 || s[len(s)-1] != '\n' {
		return s
	}
	s = s[:len(s)-1]
	if len(s) > 0 && s[len(s)-1] == '\r' {
		return s[:len(s)-1]
	}
	return s
}

const (
	ansiReset       = "\x1b[0m"
	ansiBold        = "\x1b[1m"
	ansiDim         = "\x1b[2m"
	ansiYellow      = "\x1b[33m"
	ansiRed         = "\x1b[31m"
	ansiBrightBlack = "\x1b[90m"
	ansiBrightBlue  = "\x1b[94m"
)
