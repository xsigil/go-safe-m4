package lexer

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

type TokenType int

const (
	TokenEOF TokenType = iota
	TokenIdent
	TokenQuoted
	TokenComment
	TokenOther
)

type Token struct {
	Type  TokenType
	Value string
}

type Lexer struct {
	stack        []string
	LeftQuote    string
	RightQuote   string
	LeftComment  string
	RightComment string
}

func NewLexer() *Lexer {
	return &Lexer{
		stack:        make([]string, 0),
		LeftQuote:    "`",
		RightQuote:   "'",
		LeftComment:  "#",
		RightComment: "\n",
	}
}

func (l *Lexer) PushInput(s string) {
	if s == "" {
		return
	}
	l.stack = append(l.stack, s)
}

func (l *Lexer) ReadRune() (rune, bool) {
	for len(l.stack) > 0 {
		topIdx := len(l.stack) - 1
		top := l.stack[topIdx]
		if len(top) == 0 {
			l.stack = l.stack[:topIdx]
			continue
		}

		r, size := utf8.DecodeRuneInString(top)
		if r == utf8.RuneError && size == 1 {
			l.stack[topIdx] = top[1:]
			continue
		}
		l.stack[topIdx] = top[size:]
		return r, true
	}
	return 0, false
}

func (l *Lexer) UnreadRune(r rune) {
	l.PushInput(string(r))
}

// MatchPrefix は入力スタック先頭が prefix と一致するかを検査し、一致すれば消費する
func (l *Lexer) MatchPrefix(prefix string) bool {
	if prefix == "" {
		return false
	}
	var readRunes []rune
	matched := true

	for _, pr := range prefix {
		r, ok := l.ReadRune()
		if !ok {
			matched = false
			break
		}
		readRunes = append(readRunes, r)
		if r != pr {
			matched = false
			break
		}
	}

	if matched {
		return true
	}

	// 不一致の場合は逆順でアンリードして元に戻す
	for i := len(readRunes) - 1; i >= 0; i-- {
		l.UnreadRune(readRunes[i])
	}
	return false
}

func (l *Lexer) NextToken() (Token, bool) {
	// 1. コメント判定（マルチ文字対応）
	if l.LeftComment != "" && l.MatchPrefix(l.LeftComment) {
		val := l.readComment()
		return Token{Type: TokenComment, Value: l.LeftComment + val}, true
	}

	// 2. クォート開始判定（マルチ文字対応）
	if l.MatchPrefix(l.LeftQuote) {
		val := l.readQuoted()
		return Token{Type: TokenQuoted, Value: val}, true
	}

	r, ok := l.ReadRune()
	if !ok {
		return Token{Type: TokenEOF}, false
	}

	// 3. 識別子判定
	if unicode.IsLetter(r) || r == '_' {
		var sb strings.Builder
		sb.WriteRune(r)
		for {
			nr, nok := l.ReadRune()
			if !nok {
				break
			}
			if unicode.IsLetter(nr) || unicode.IsDigit(nr) || nr == '_' {
				sb.WriteRune(nr)
			} else {
				l.UnreadRune(nr)
				break
			}
		}
		return Token{Type: TokenIdent, Value: sb.String()}, true
	}

	// 4. その他
	return Token{Type: TokenOther, Value: string(r)}, true
}

func (l *Lexer) readQuoted() string {
	var sb strings.Builder
	depth := 1

	for {
		if l.MatchPrefix(l.LeftQuote) {
			depth++
			sb.WriteString(l.LeftQuote)
			continue
		}
		if l.MatchPrefix(l.RightQuote) {
			depth--
			if depth == 0 {
				return sb.String()
			}
			sb.WriteString(l.RightQuote)
			continue
		}

		r, ok := l.ReadRune()
		if !ok {
			return sb.String()
		}
		sb.WriteRune(r)
	}
}

func (l *Lexer) readComment() string {
	var sb strings.Builder
	rc := l.RightComment
	if rc == "" {
		rc = "\n"
	}

	for {
		if l.MatchPrefix(rc) {
			sb.WriteString(rc)
			return sb.String()
		}
		r, ok := l.ReadRune()
		if !ok {
			return sb.String()
		}
		sb.WriteRune(r)
	}
}
