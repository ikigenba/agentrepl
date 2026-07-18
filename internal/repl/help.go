package repl

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/ikigenba/agentkit"
	"github.com/ikigenba/agentrepl/internal/catalog"
)

// WriteHelp renders the static launch-time catalog.
func WriteHelp(out io.Writer, name string, cat []catalog.Provider) {
	fmt.Fprintf(out, "usage: %s [-c key=value ...] [-raw] [-h]\n\n", name)
	fmt.Fprintln(out, "flags:")
	fmt.Fprintln(out, "  -c key=value   set an agentkit config value (repeatable); see config keys via /help at runtime")
	fmt.Fprintln(out, "  -raw           emit the raw, undecorated message stream")
	fmt.Fprintln(out, "  -h, -help      show this catalog and exit")
	fmt.Fprintln(out)

	fmt.Fprintln(out, "providers:")
	for _, provider := range cat {
		fmt.Fprintf(out, "  %-10s  (%s)\n", provider.Name, provider.EnvKey)
	}
	fmt.Fprintln(out)

	fmt.Fprintln(out, "models:")
	for _, provider := range cat {
		fmt.Fprintf(out, "  %s\n", provider.Name)
		for _, entry := range catalog.Models(provider.Name) {
			fmt.Fprintf(out, "    %-24s %s\n", entry.Model, reasoningClause(entry.Reasoning))
		}
	}
}

func reasoningClause(spec *agentkit.ReasoningSpec) string {
	if spec == nil {
		return "(no reasoning control)"
	}
	key := termToKey(spec.Term)
	switch spec.Kind {
	case agentkit.ReasoningEnum:
		clause := fmt.Sprintf("%s={%s}", key, strings.Join(markReasoningDefault(spec.Levels, formatReasoningDefault(*spec)), "|"))
		if residue := reasoningTermResidue(spec.Term, key); residue != "" {
			clause += "  (" + residue + ")"
		}
		return clause
	case agentkit.ReasoningRange:
		clause := fmt.Sprintf("%s=<%d–%d>", key, spec.Min, spec.Max)
		parts := make([]string, 0, len(spec.Sentinels))
		defaultBudget, hasDefaultBudget := spec.Default.Budget()
		markedDefault := false
		for _, sentinel := range spec.Sentinels {
			meaning := sentinel.Meaning
			if hasDefaultBudget && !markedDefault && defaultBudget == sentinel.Value {
				meaning = "*" + meaning
				markedDefault = true
			}
			parts = append(parts, fmt.Sprintf("%d=%s", sentinel.Value, meaning))
		}
		details := strings.Join(parts, ", ")
		if markedDefault {
			return clause + "  (" + details + ")"
		}
		if details != "" {
			details += "; "
		}
		return clause + "  (" + details + "default " + formatReasoningDefault(*spec) + ")"
	case agentkit.ReasoningToggle:
		return fmt.Sprintf("%s={%s}", key, strings.Join(markReasoningDefault([]string{"on", "off"}, formatReasoningDefault(*spec)), "|"))
	default:
		return "(no reasoning control)"
	}
}

func markReasoningDefault(values []string, defaultValue string) []string {
	marked := make([]string, len(values))
	for i, value := range values {
		marked[i] = value
		if value == defaultValue {
			marked[i] = "*" + value
		}
	}
	return marked
}

func reasoningTermResidue(term, key string) string {
	base := strings.ReplaceAll(key, "_", " ")
	if len(term) < len(base) || !strings.EqualFold(term[:len(base)], base) {
		return ""
	}
	residue := strings.TrimSpace(term[len(base):])
	residue = strings.TrimPrefix(residue, "(")
	return strings.TrimSpace(strings.TrimSuffix(residue, ")"))
}

func termToKey(term string) string {
	key := strings.ToLower(term)
	key = strings.TrimSuffix(key, " (+ toggle)")
	return strings.ReplaceAll(key, " ", "_")
}

func formatReasoningDefault(spec agentkit.ReasoningSpec) string {
	value := spec.Default
	if value.Disabled() {
		return "off"
	}
	if level, ok := value.Level(); ok {
		return level
	}
	if budget, ok := value.Budget(); ok {
		for _, sentinel := range spec.Sentinels {
			if budget == sentinel.Value {
				return sentinel.Meaning
			}
		}
		return strconv.Itoa(budget)
	}
	if spec.Kind == agentkit.ReasoningToggle && value.IsUnset() {
		return "on"
	}
	return "default"
}
