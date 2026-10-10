package tests

import (
	"fmt"
	"strings"
	"testing"

	"github.com/xsigil/go-safe-m4/pkg/engine"
)

func formatDiffPOSIX(got, want string) string {
	var sb strings.Builder
	sb.WriteString("\n--- CHAR DIFF ---\n")
	maxLen := len(got)
	if len(want) > maxLen {
		maxLen = len(want)
	}
	for i := 0; i < maxLen; i++ {
		var gStr, wStr string
		var gByte, wByte byte
		if i < len(got) {
			gByte = got[i]
			gStr = fmt.Sprintf("%q (0x%02X)", string(gByte), gByte)
		} else {
			gStr = "<EOF>"
		}
		if i < len(want) {
			wByte = want[i]
			wStr = fmt.Sprintf("%q (0x%02X)", string(wByte), wByte)
		} else {
			wStr = "<EOF>"
		}

		status := "OK"
		if i >= len(got) || i >= len(want) || gByte != wByte {
			status = "MISMATCH"
		}
		sb.WriteString(fmt.Sprintf("[%02d] GOT: %-15s | WANT: %-15s | %s\n", i, gStr, wStr, status))
	}
	return sb.String()
}

func TestEngine_POSIXBuiltins(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "len basic",
			input:    "len(`abcdef')",
			expected: "6",
		},
		{
			name:     "len utf-8",
			input:    "len(`日本語')",
			expected: "3",
		},
		{
			name:     "index found",
			input:    "index(`hello world', `world')",
			expected: "6",
		},
		{
			name:     "index not found",
			input:    "index(`hello world', `xyz')",
			expected: "-1",
		},
		{
			name:     "substr basic",
			input:    "substr(`abcdef', 2, 3)",
			expected: "cde",
		},
		{
			name:     "substr to end",
			input:    "substr(`abcdef', 3)",
			expected: "def",
		},
		{
			name:     "translit range expansion",
			input:    "translit(`hello 123', `a-z', `A-Z')",
			expected: "HELLO 123",
		},
		{
			name:     "translit deletion",
			input:    "translit(`hello world', `lo')",
			expected: "he wrd",
		},
		{
			name:     "incr and decr",
			input:    "incr(41) decr(43)",
			expected: "42 42",
		},
		{
			name:     "eval simple math",
			input:    "eval(`(10 + 20) * 2 - 5')",
			expected: "55",
		},
		{
			name:     "eval radix hex",
			input:    "eval(255, 16)",
			expected: "ff",
		},
		{
			name:     "eval comparison with ifelse",
			input:    "ifelse(eval(`10 > 5'), 1, `YES', `NO')",
			expected: "YES",
		},
		{
			name:     "defn preserves macro definition",
			input:    "define(`A', `hello $1')define(`B', defn(`A'))B(`world')",
			expected: "hello world",
		},
		{
			name:     "shift discards first arg",
			input:    "define(`TAIL', `shift($@)')TAIL(`one', `two', `three')",
			expected: "two,three",
		},
		{
			name:     "comments pass through without expanding macros",
			input:    "define(`FOO', `bar')# FOO is not expanded\nFOO",
			expected: "# FOO is not expanded\nbar",
		},
		{
			name:     "changecom custom comment delimiters",
			input:    "define(`FOO', `bar')changecom(`/*', `*/')/* FOO */ FOO",
			expected: "/* FOO */ bar",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eng := engine.NewEngine()
			got, err := eng.Expand(tt.input)
			if err != nil {
				t.Fatalf("unexpected expansion error: %v", err)
			}
			if got != tt.expected {
				t.Errorf("\n=== FAILURE: %s ===\nINPUT   : %s\nGOT     : %q\nWANT    : %q%s",
					tt.name, tt.input, got, tt.expected, formatDiffPOSIX(got, tt.expected))
			}
		})
	}
}
