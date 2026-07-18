package repl

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ikigenba/agentkit"
	"github.com/ikigenba/agentrepl/internal/catalog"
)

func TestWriteHelpGoldenIncludesDefaultsAuthRoutesAndTwoTierFooter(t *testing.T) {
	// R-FVOP-QMBI
	// R-6DEO-9TXQ
	// R-ODOF-XOTJ
	// R-OEWC-BGK8
	// R-5873-KWGL
	// R-5AMW-CFXZ
	// R-5BUS-Q7OO
	// R-5D2P-3ZFD
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

func TestWriteHelpMarksMatchingRangeSentinelWithoutTrailingDefault(t *testing.T) {
	// R-APDX-FP3D
	specs := []*agentkit.ReasoningSpec{
		{
			Term: "thinking budget", Kind: agentkit.ReasoningRange,
			Min: 0, Max: 24576, Default: agentkit.Budget(-1),
			Sentinels: []agentkit.Sentinel{
				{Value: 0, Meaning: "off"},
				{Value: -1, Meaning: "dynamic"},
			},
		},
		{
			Term: "thinking budget", Kind: agentkit.ReasoningRange,
			Min: 1024, Max: 4096, Default: agentkit.DisableReasoning(),
		},
	}

	var out bytes.Buffer
	for _, spec := range specs {
		out.WriteString(reasoningClause(spec))
		out.WriteByte('\n')
	}
	help := out.String()
	for _, want := range []string{
		"thinking_budget=<0–24576>  (0=off, -1=*dynamic)",
		"thinking_budget=<1024–4096>  (default off)",
	} {
		if !strings.Contains(help, want) {
			t.Errorf("help output = %q, want %q", help, want)
		}
	}
	if strings.Contains(help, "default dynamic") {
		t.Fatalf("help output = %q, contains trailing default for a starred sentinel", help)
	}
}
