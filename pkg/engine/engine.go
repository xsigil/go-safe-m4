package engine

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/xsigil/go-safe-m4/pkg/builtin"
	"github.com/xsigil/go-safe-m4/pkg/lexer"
)

type Engine struct {
	lex         *lexer.Lexer
	macros      map[string]string
	builtins    map[string]builtin.MacroFunc
	diversions  map[int]*strings.Builder
	currentDiv  int
	sandboxRoot string
	maxDepth    int
}

func NewEngine(opts ...Option) *Engine {
	e := &Engine{
		lex:         lexer.NewLexer(),
		macros:      make(map[string]string),
		builtins:    make(map[string]builtin.MacroFunc),
		diversions:  make(map[int]*strings.Builder),
		currentDiv:  0,
		sandboxRoot: ".",
		maxDepth:    1024,
	}
	builtin.RegisterDefaults(e)
	for _, opt := range opts {
		opt(e)
	}
	return e
}

type Option func(*Engine)

func WithSandboxRoot(root string) Option {
	return func(e *Engine) {
		e.sandboxRoot = root
	}
}

func (e *Engine) RegisterBuiltin(name string, fn builtin.MacroFunc) {
	e.builtins[name] = fn
}

func (e *Engine) SetMacro(name, val string) {
	e.macros[name] = val
}

func (e *Engine) GetMacro(name string) (string, bool) {
	val, ok := e.macros[name]
	return val, ok
}

func (e *Engine) DeleteMacro(name string) {
	delete(e.macros, name)
	delete(e.builtins, name)
}

func (e *Engine) HasMacroOrBuiltin(name string) bool {
	if _, ok := e.builtins[name]; ok {
		return true
	}
	_, ok := e.macros[name]
	return ok
}

func (e *Engine) GetLexer() *lexer.Lexer {
	return e.lex
}

func (e *Engine) SetCurrentDiv(div int) {
	e.currentDiv = div
}

func (e *Engine) GetDiversions() map[int]*strings.Builder {
	return e.diversions
}

func (e *Engine) GetSandboxRoot() string {
	return e.sandboxRoot
}

func (e *Engine) Define(name, value string) {
	e.macros[name] = value
}

func (e *Engine) Expand(input string) (string, error) {
	e.lex.PushInput(input)
	depth := 0

	for {
		tok, ok := e.lex.NextToken()
		if !ok {
			break
		}

		switch tok.Type {
		case lexer.TokenQuoted, lexer.TokenComment, lexer.TokenOther:
			e.writeOutput(tok.Value)

		case lexer.TokenIdent:
			name := tok.Value
			if !e.HasMacroOrBuiltin(name) {
				e.writeOutput(name)
				continue
			}

			args, _, err := e.collectArgs()
			if err != nil {
				return "", err
			}

			depth++
			if depth > e.maxDepth {
				return "", fmt.Errorf("maximum macro recursion depth exceeded (%d)", e.maxDepth)
			}

			if isRawBuiltin(name) {
				res, err := e.evalMacro(name, args)
				if err != nil {
					return "", err
				}
				e.writeOutput(res)
				continue
			}

			expansion, err := e.evalMacro(name, args)
			if err != nil {
				return "", err
			}

			e.lex.PushInput(expansion)
		}
	}

	var finalOut strings.Builder
	if out0, ok := e.diversions[0]; ok {
		finalOut.WriteString(out0.String())
	}
	return finalOut.String(), nil
}

func isRawBuiltin(name string) bool {
	switch name {
	case "h", "html_escape", "json_escape", "sh_quote":
		return true
	default:
		return false
	}
}

func (e *Engine) writeOutput(s string) {
	if e.currentDiv < 0 {
		return
	}
	buf, ok := e.diversions[e.currentDiv]
	if !ok {
		buf = &strings.Builder{}
		e.diversions[e.currentDiv] = buf
	}
	buf.WriteString(s)
}

func (e *Engine) collectArgs() ([]string, bool, error) {
	r, ok := e.lex.ReadRune()
	if !ok {
		return nil, false, nil
	}
	if r != '(' {
		e.lex.UnreadRune(r)
		return nil, false, nil
	}

	var args []string
	var current strings.Builder
	parenDepth := 0
	quoteDepth := 0

	lq := e.lex.LeftQuote
	rq := e.lex.RightQuote

	processArg := func(raw string) (string, error) {
		s := strings.TrimSpace(raw)
		if strings.HasPrefix(s, lq) && strings.HasSuffix(s, rq) {
			return s[len(lq) : len(s)-len(rq)], nil
		}
		if s == "" {
			return "", nil
		}
		subEngine := &Engine{
			lex:         lexer.NewLexer(),
			macros:      e.macros,
			builtins:    e.builtins,
			diversions:  make(map[int]*strings.Builder),
			currentDiv:  0,
			sandboxRoot: e.sandboxRoot,
			maxDepth:    e.maxDepth,
		}
		return subEngine.Expand(s)
	}

	for {
		if e.lex.MatchPrefix(lq) {
			quoteDepth++
			current.WriteString(lq)
			continue
		}
		if quoteDepth > 0 && e.lex.MatchPrefix(rq) {
			quoteDepth--
			current.WriteString(rq)
			continue
		}

		r, ok := e.lex.ReadRune()
		if !ok {
			return nil, true, fmt.Errorf("unexpected EOF while reading macro arguments")
		}

		if quoteDepth > 0 {
			current.WriteRune(r)
			continue
		}

		switch r {
		case '(':
			parenDepth++
			current.WriteRune('(')
		case ')':
			if parenDepth == 0 {
				argStr, err := processArg(current.String())
				if err != nil {
					return nil, true, err
				}
				if len(args) == 0 && current.Len() == 0 {
					return args, true, nil
				}
				args = append(args, argStr)
				return args, true, nil
			}
			parenDepth--
			current.WriteRune(')')
		case ',':
			if parenDepth == 0 {
				argStr, err := processArg(current.String())
				if err != nil {
					return nil, true, err
				}
				args = append(args, argStr)
				current.Reset()
			} else {
				current.WriteRune(',')
			}
		default:
			current.WriteRune(r)
		}
	}
}

func (e *Engine) evalMacro(name string, args []string) (string, error) {
	if fn, ok := e.builtins[name]; ok {
		return fn(e, args)
	}

	body, ok := e.macros[name]
	if !ok {
		return "", nil
	}

	return substituteArgs(body, name, args), nil
}

func substituteArgs(body, macroName string, args []string) string {
	res := body
	res = strings.ReplaceAll(res, "$0", macroName)
	res = strings.ReplaceAll(res, "$#", strconv.Itoa(len(args)))
	res = strings.ReplaceAll(res, "$*", strings.Join(args, ","))
	res = strings.ReplaceAll(res, "$@", strings.Join(args, ","))

	for i, arg := range args {
		placeholder := fmt.Sprintf("$%d", i+1)
		res = strings.ReplaceAll(res, placeholder, arg)
	}
	return res
}
