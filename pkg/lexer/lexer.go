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
	TokenOther
)

type Token struct {
	Type  TokenType
	Value string
}

type Lexer struct {
	stack      []string
	LeftQuote  string
	RightQuote string
}

func NewLexer() *Lexer {
	return &Lexer{
		stack:      make([]string, 0),
		LeftQuote:  "`",
		RightQuote: "'",
	}
}

func (l *Lexer) PushInput(s string) {
	if s == "" {
		return
	}
	l.stack = append(l.stack, s)
}

// ReadRune は utf8.DecodeRuneInString を用いて確実に先頭 1 rune を読み出します
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
			// 不正バイトはスキップ
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

func (l *Lexer) NextToken() (Token, bool) {
	r, ok := l.ReadRune()
	if !ok {
		return Token{Type: TokenEOF}, false
	}

	// 1. クォート開始
	if string(r) == l.LeftQuote {
		val := l.readQuoted()
		return Token{Type: TokenQuoted, Value: val}, true
	}

	// 2. 識別子
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

	// 3. その他
	return Token{Type: TokenOther, Value: string(r)}, true
}

func (l *Lexer) readQuoted() string {
	var sb strings.Builder
	depth := 1

	for {
		r, ok := l.ReadRune()
		if !ok {
			return sb.String()
		}

		if string(r) == l.LeftQuote {
			depth++
			sb.WriteString(l.LeftQuote)
			continue
		}
		if string(r) == l.RightQuote {
			depth--
			if depth == 0 {
				return sb.String()
			}
			sb.WriteString(l.RightQuote)
			continue
		}

		sb.WriteRune(r)
	}
}
