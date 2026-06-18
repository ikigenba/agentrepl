package render

import "github.com/ikigenba/agentkit"

// Renderer presents one turn's prompt, streamed events, outcome, and spend.
type Renderer interface {
	Prompt(text string)
	Event(ev agentkit.Event)
	Usage(turn agentkit.Usage, turnCost, total agentkit.Cost)
	Summary(total agentkit.Usage, totalCost agentkit.Cost)
	Error(err error)
	Notice(line string)
}
