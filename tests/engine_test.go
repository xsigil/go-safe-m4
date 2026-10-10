package tests

import (
	"fmt"
	"strings"
	"testing"

	"github.com/xsigil/go-safe-m4/pkg/engine"
)

func formatDiff(got, want string) string {
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

func TestEngine_BasicMacros(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple define and expand",
			input:    "define(`FOO', `bar')FOO",
			expected: "bar",
		},
		{
			name:     "macro with arguments",
			input:    "define(`GREET', `Hello, $1!')GREET(`World')",
			expected: "Hello, World!",
		},
		{
			name:     "dnl removes trailing newline",
			input:    "define(`FOO', `bar')dnl\nFOO",
			expected: "bar",
		},
		{
			name:     "ifdef defined",
			input:    "define(`FLAG', `1')ifdef(`FLAG', `YES', `NO')",
			expected: "YES",
		},
		{
			name:     "ifdef undefined",
			input:    "ifdef(`FLAG', `YES', `NO')",
			expected: "NO",
		},
		{
			name:     "ifelse match",
			input:    "ifelse(`a', `a', `match', `mismatch')",
			expected: "match",
		},
		{
			name:     "ifelse mismatch",
			input:    "ifelse(`a', `b', `match', `mismatch')",
			expected: "mismatch",
		},
		{
			name:     "nested quotes preservation",
			input:    "define(`WRAP', `[$1]')WRAP(```inner quote''')",
			expected: "[`inner quote']",
		},
		{
			name:     "html escape via h()",
			input:    "h(`<script>alert(\"XSS\") & 'foo'</script>')",
			expected: "&lt;script&gt;alert(&#34;XSS&#34;) &amp; &#39;foo&#39;&lt;/script&gt;",
		},
		{
			name:     "json escape",
			input:    "json_escape(`hello \"world\" \n test')",
			expected: `hello \"world\" \n test`,
		},
		{
			name:     "shell quote",
			input:    "sh_quote(`echo \"hello\"')",
			expected: "'echo \"hello\"'",
		},
		{
			name:     "shell quote with single quotes",
			input:    "sh_quote(`a'b')",
			expected: "'a'\\''b'",
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
				t.Errorf("\n=== FAILURE: %s ===\nINPUT   : %s\nGOT     : %s\nWANT    : %s%s",
					tt.name, tt.input, got, tt.expected, formatDiff(got, tt.expected))
			}
		})
	}
}

func TestEngine_Security(t *testing.T) {
	t.Run("traversal include blocked", func(t *testing.T) {
		eng := engine.NewEngine(engine.WithSandboxRoot("/tmp/sandbox"))
		_, err := eng.Expand("include(`../../etc/passwd')")
		if err == nil {
			t.Fatalf("expected error on path traversal, got nil")
		}
		if !strings.Contains(err.Error(), "security violation") {
			t.Errorf("expected security violation error, got %v", err)
		}
	})

	t.Run("syscmd disabled by default", func(t *testing.T) {
		eng := engine.NewEngine()
		got, err := eng.Expand("syscmd(`id')")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != "syscmd(id)" {
			t.Errorf("expected 'syscmd(id)', got %q", got)
		}
	})
}
