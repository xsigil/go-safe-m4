package builtin

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/xsigil/go-safe-m4/pkg/lexer"
)

type EngineContext interface {
	RegisterBuiltin(name string, fn MacroFunc)
	SetMacro(name, val string)
	DeleteMacro(name string)
	HasMacroOrBuiltin(name string) bool
	GetLexer() *lexer.Lexer
	SetCurrentDiv(div int)
	GetDiversions() map[int]*strings.Builder
	GetSandboxRoot() string
}

type MacroFunc func(ctx EngineContext, args []string) (string, error)

func RegisterDefaults(ctx EngineContext) {
	ctx.RegisterBuiltin("define", BuiltinDefine)
	ctx.RegisterBuiltin("undefine", BuiltinUndefine)
	ctx.RegisterBuiltin("ifdef", BuiltinIfdef)
	ctx.RegisterBuiltin("ifelse", BuiltinIfelse)
	ctx.RegisterBuiltin("changequote", BuiltinChangequote)
	ctx.RegisterBuiltin("divert", BuiltinDivert)
	ctx.RegisterBuiltin("undivert", BuiltinUndivert)
	ctx.RegisterBuiltin("include", BuiltinInclude)
	ctx.RegisterBuiltin("dnl", BuiltinDnl)

	// Modern Safe M4 Escaping Builtins
	ctx.RegisterBuiltin("h", BuiltinHTMLEscape)
	ctx.RegisterBuiltin("html_escape", BuiltinHTMLEscape)
	ctx.RegisterBuiltin("json_escape", BuiltinJSONEscape)
	ctx.RegisterBuiltin("sh_quote", BuiltinShellQuote)
}

func BuiltinDefine(ctx EngineContext, args []string) (string, error) {
	if len(args) < 1 || args[0] == "" {
		return "", nil
	}
	val := ""
	if len(args) >= 2 {
		val = args[1]
	}
	ctx.SetMacro(args[0], val)
	return "", nil
}

func BuiltinUndefine(ctx EngineContext, args []string) (string, error) {
	if len(args) > 0 {
		ctx.DeleteMacro(args[0])
	}
	return "", nil
}

func BuiltinIfdef(ctx EngineContext, args []string) (string, error) {
	if len(args) < 2 {
		return "", nil
	}
	if ctx.HasMacroOrBuiltin(args[0]) {
		return args[1], nil
	}
	if len(args) >= 3 {
		return args[2], nil
	}
	return "", nil
}

func BuiltinIfelse(ctx EngineContext, args []string) (string, error) {
	for len(args) >= 3 {
		if args[0] == args[1] {
			return args[2], nil
		}
		if len(args) == 4 {
			return args[3], nil
		}
		args = args[3:]
	}
	return "", nil
}

func BuiltinChangequote(ctx EngineContext, args []string) (string, error) {
	l := ctx.GetLexer()
	if len(args) == 0 || (args[0] == "" && len(args) == 1) {
		l.LeftQuote = "`"
		l.RightQuote = "'"
		return "", nil
	}
	l.LeftQuote = args[0]
	if len(args) >= 2 {
		l.RightQuote = args[1]
	} else {
		l.RightQuote = "'"
	}
	return "", nil
}

func BuiltinDivert(ctx EngineContext, args []string) (string, error) {
	divNum := 0
	if len(args) > 0 && args[0] != "" {
		if n, err := strconv.Atoi(args[0]); err == nil {
			divNum = n
		}
	}
	ctx.SetCurrentDiv(divNum)
	return "", nil
}

func BuiltinUndivert(ctx EngineContext, args []string) (string, error) {
	divs := ctx.GetDiversions()
	var targetDivs []int
	if len(args) == 0 {
		for d := range divs {
			if d > 0 {
				targetDivs = append(targetDivs, d)
			}
		}
	} else {
		for _, arg := range args {
			if d, err := strconv.Atoi(arg); err == nil && d > 0 {
				targetDivs = append(targetDivs, d)
			}
		}
	}

	var out strings.Builder
	for _, d := range targetDivs {
		if buf, exists := divs[d]; exists {
			out.WriteString(buf.String())
			delete(divs, d)
		}
	}
	return out.String(), nil
}

func BuiltinInclude(ctx EngineContext, args []string) (string, error) {
	if len(args) == 0 || args[0] == "" {
		return "", fmt.Errorf("include: missing file path")
	}

	root := ctx.GetSandboxRoot()
	cleanPath := filepath.Clean(args[0])

	var targetPath string
	if root != "" {
		targetPath = filepath.Join(root, cleanPath)
		rel, err := filepath.Rel(root, targetPath)
		if err != nil || strings.HasPrefix(rel, "..") || strings.HasPrefix(rel, "/") {
			return "", fmt.Errorf("security violation: path traversal detected in include(%s)", args[0])
		}
	} else {
		if filepath.IsAbs(cleanPath) {
			return "", fmt.Errorf("security violation: absolute paths not allowed in include(%s)", args[0])
		}
		targetPath = cleanPath
	}

	content, err := os.ReadFile(targetPath)
	if err != nil {
		return "", fmt.Errorf("include(%s): %w", args[0], err)
	}
	return string(content), nil
}

func BuiltinDnl(ctx EngineContext, args []string) (string, error) {
	l := ctx.GetLexer()
	for {
		r, ok := l.ReadRune()
		if !ok || r == '\n' {
			break
		}
	}
	return "", nil
}

func BuiltinHTMLEscape(ctx EngineContext, args []string) (string, error) {
	if len(args) == 0 {
		return "", nil
	}
	return HTMLEscape(args[0]), nil
}

func BuiltinJSONEscape(ctx EngineContext, args []string) (string, error) {
	if len(args) == 0 {
		return "", nil
	}
	return JSONEscape(args[0])
}

func BuiltinShellQuote(ctx EngineContext, args []string) (string, error) {
	if len(args) == 0 {
		return "", nil
	}
	return ShellQuote(args[0]), nil
}
