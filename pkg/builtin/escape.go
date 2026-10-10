package builtin

import (
	"encoding/json"
	"html"
	"strings"
)

func HTMLEscape(s string) string {
	return html.EscapeString(s)
}

func JSONEscape(s string) (string, error) {
	b, err := json.Marshal(s)
	if err != nil {
		return "", err
	}
	res := string(b)
	if len(res) >= 2 && res[0] == '"' && res[len(res)-1] == '"' {
		return res[1 : len(res)-1], nil
	}
	return res, nil
}

func ShellQuote(s string) string {
	if s == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}
