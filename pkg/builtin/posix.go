package builtin

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strconv"
	"strings"
	"unicode/utf8"
)

func RegisterPOSIX(ctx EngineContext) {
	ctx.RegisterBuiltin("len", BuiltinLen)
	ctx.RegisterBuiltin("index", BuiltinIndex)
	ctx.RegisterBuiltin("substr", BuiltinSubstr)
	ctx.RegisterBuiltin("translit", BuiltinTranslit)
	ctx.RegisterBuiltin("eval", BuiltinEval)
	ctx.RegisterBuiltin("incr", BuiltinIncr)
	ctx.RegisterBuiltin("decr", BuiltinDecr)
	ctx.RegisterBuiltin("shift", BuiltinShift)
	ctx.RegisterBuiltin("defn", BuiltinDefn)
	ctx.RegisterBuiltin("changecom", BuiltinChangecom)
}

func BuiltinLen(ctx EngineContext, args []string) (string, error) {
	if len(args) == 0 {
		return "0", nil
	}
	return strconv.Itoa(utf8.RuneCountInString(args[0])), nil
}

func BuiltinIndex(ctx EngineContext, args []string) (string, error) {
	if len(args) < 2 {
		return "-1", nil
	}
	s := args[0]
	sub := args[1]
	byteIdx := strings.Index(s, sub)
	if byteIdx < 0 {
		return "-1", nil
	}
	return strconv.Itoa(utf8.RuneCountInString(s[:byteIdx])), nil
}

func BuiltinSubstr(ctx EngineContext, args []string) (string, error) {
	if len(args) == 0 {
		return "", nil
	}
	runes := []rune(args[0])
	from := 0
	if len(args) >= 2 && args[1] != "" {
		if n, err := strconv.Atoi(args[1]); err == nil {
			from = n
		}
	}

	if from < 0 || from >= len(runes) {
		return "", nil
	}

	if len(args) >= 3 && args[2] != "" {
		length, err := strconv.Atoi(args[2])
		if err == nil {
			if length <= 0 {
				return "", nil
			}
			to := from + length
			if to > len(runes) {
				to = len(runes)
			}
			return string(runes[from:to]), nil
		}
	}

	return string(runes[from:]), nil
}

// BuiltinTranslit: a-z などのハイフン範囲指定に対応
func BuiltinTranslit(ctx EngineContext, args []string) (string, error) {
	if len(args) == 0 {
		return "", nil
	}
	if len(args) == 1 {
		return args[0], nil
	}

	fromRunes := expandRanges(args[1])
	toRunes := []rune("")
	if len(args) >= 3 {
		toRunes = expandRanges(args[2])
	}

	mapping := make(map[rune]rune)
	deletion := make(map[rune]bool)

	for i, f := range fromRunes {
		if i < len(toRunes) {
			mapping[f] = toRunes[i]
		} else {
			deletion[f] = true
		}
	}

	var sb strings.Builder
	for _, r := range []rune(args[0]) {
		if deletion[r] {
			continue
		}
		if repl, ok := mapping[r]; ok {
			sb.WriteRune(repl)
		} else {
			sb.WriteRune(r)
		}
	}
	return sb.String(), nil
}

func expandRanges(s string) []rune {
	runes := []rune(s)
	var res []rune
	for i := 0; i < len(runes); i++ {
		if i+2 < len(runes) && runes[i+1] == '-' && runes[i] <= runes[i+2] {
			for r := runes[i]; r <= runes[i+2]; r++ {
				res = append(res, r)
			}
			i += 2
		} else {
			res = append(res, runes[i])
		}
	}
	return res
}

func BuiltinIncr(ctx EngineContext, args []string) (string, error) {
	if len(args) == 0 || args[0] == "" {
		return "", fmt.Errorf("incr: missing argument")
	}
	n, err := strconv.Atoi(strings.TrimSpace(args[0]))
	if err != nil {
		return "", fmt.Errorf("incr: non-numeric argument %q", args[0])
	}
	return strconv.Itoa(n + 1), nil
}

func BuiltinDecr(ctx EngineContext, args []string) (string, error) {
	if len(args) == 0 || args[0] == "" {
		return "", fmt.Errorf("decr: missing argument")
	}
	n, err := strconv.Atoi(strings.TrimSpace(args[0]))
	if err != nil {
		return "", fmt.Errorf("decr: non-numeric argument %q", args[0])
	}
	return strconv.Itoa(n - 1), nil
}

// BuiltinShift: shift($@) -> 第1引数を破棄し、残りの引数をクォートしてカンマ結合
func BuiltinShift(ctx EngineContext, args []string) (string, error) {
	if len(args) <= 1 {
		return "", nil
	}
	lq := ctx.GetLexer().LeftQuote
	rq := ctx.GetLexer().RightQuote

	var quoted []string
	for _, arg := range args[1:] {
		quoted = append(quoted, lq+arg+rq)
	}
	return strings.Join(quoted, ","), nil
}

// BuiltinDefn: defn(name) -> 定義済みマクロの内容をクォート付きで返す
func BuiltinDefn(ctx EngineContext, args []string) (string, error) {
	if len(args) == 0 || args[0] == "" {
		return "", nil
	}
	name := args[0]
	lq := ctx.GetLexer().LeftQuote
	rq := ctx.GetLexer().RightQuote

	// エンジン側の生マクロマップから取得（EngineContext 経由）
	if getter, ok := ctx.(MacroGetter); ok {
		if val, exists := getter.GetMacro(name); exists {
			return lq + val + rq, nil
		}
	}
	return "", nil
}

type MacroGetter interface {
	GetMacro(name string) (string, bool)
}

// BuiltinChangecom: changecom([left], [right])
func BuiltinChangecom(ctx EngineContext, args []string) (string, error) {
	lex := ctx.GetLexer()
	if len(args) == 0 || args[0] == "" {
		// 引数なしはコメント機能無効化
		lex.LeftComment = ""
		lex.RightComment = ""
		return "", nil
	}
	lex.LeftComment = args[0]
	if len(args) >= 2 && args[1] != "" {
		lex.RightComment = args[1]
	} else {
		lex.RightComment = "\n"
	}
	return "", nil
}

// BuiltinEval: eval(expression, [radix])
func BuiltinEval(ctx EngineContext, args []string) (string, error) {
	if len(args) == 0 || strings.TrimSpace(args[0]) == "" {
		return "0", nil
	}

	radix := 10
	if len(args) >= 2 && args[1] != "" {
		if r, err := strconv.Atoi(strings.TrimSpace(args[1])); err == nil && r >= 2 && r <= 36 {
			radix = r
		}
	}

	exprStr := strings.TrimSpace(args[0])
	parsed, err := parser.ParseExpr(exprStr)
	if err != nil {
		return "", fmt.Errorf("eval: syntax error in expression %q: %w", exprStr, err)
	}

	val, err := evalAST(parsed)
	if err != nil {
		return "", fmt.Errorf("eval: %w", err)
	}

	if radix == 10 {
		return strconv.Itoa(val), nil
	}
	return strconv.FormatInt(int64(val), radix), nil
}

func evalAST(node ast.Expr) (int, error) {
	switch n := node.(type) {
	case *ast.BasicLit:
		if n.Kind == token.INT {
			return strconv.Atoi(n.Value)
		}
		return 0, fmt.Errorf("unsupported literal kind")
	case *ast.ParenExpr:
		return evalAST(n.X)
	case *ast.UnaryExpr:
		x, err := evalAST(n.X)
		if err != nil {
			return 0, err
		}
		switch n.Op {
		case token.SUB:
			return -x, nil
		case token.ADD:
			return x, nil
		case token.NOT:
			if x == 0 {
				return 1, nil
			}
			return 0, nil
		default:
			return 0, fmt.Errorf("unsupported unary operator: %s", n.Op)
		}
	case *ast.BinaryExpr:
		x, err := evalAST(n.X)
		if err != nil {
			return 0, err
		}
		y, err := evalAST(n.Y)
		if err != nil {
			return 0, err
		}
		switch n.Op {
		case token.ADD:
			return x + y, nil
		case token.SUB:
			return x - y, nil
		case token.MUL:
			return x * y, nil
		case token.QUO:
			if y == 0 {
				return 0, fmt.Errorf("division by zero")
			}
			return x / y, nil
		case token.REM:
			if y == 0 {
				return 0, fmt.Errorf("division by zero")
			}
			return x % y, nil
		case token.EQL:
			return boolToInt(x == y), nil
		case token.NEQ:
			return boolToInt(x != y), nil
		case token.LSS:
			return boolToInt(x < y), nil
		case token.LEQ:
			return boolToInt(x <= y), nil
		case token.GTR:
			return boolToInt(x > y), nil
		case token.GEQ:
			return boolToInt(x >= y), nil
		default:
			return 0, fmt.Errorf("unsupported binary operator: %s", n.Op)
		}
	default:
		return 0, fmt.Errorf("unsupported expression")
	}
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
