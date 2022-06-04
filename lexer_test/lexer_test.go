package lexer_test

import (
	"fmt"
	"testing"

	"github.com/bbuck/go-lexer"
)

const (
	NumberToken lexer.TokenType = iota
	OpToken
	IdentToken
)

func NumberState(l *lexer.L) lexer.StateFunc {
	l.Take("0123456789")
	l.Emit(NumberToken)
	if l.Peek() == '.' {
		l.Next()
		l.Emit(OpToken)
		return IdentState
	}

	return nil
}

func IdentState(l *lexer.L) lexer.StateFunc {
	r := l.Next()
	for (r >= 'a' && r <= 'z') || r == '_' {
		r = l.Next()
	}
	l.Rewind()
	l.Emit(IdentToken)

	return WhitespaceState
}

func WhitespaceState(l *lexer.L) lexer.StateFunc {
	r := l.Next()
	if r == lexer.EOFRune {
		return nil
	}

	if r != ' ' && r != '\t' && r != '\n' && r != '\r' {
		l.Error(fmt.Sprintf("unexpected token %q", r))
		return nil
	}

	l.Take(" \t\n\r")
	l.Ignore()

	return NumberState
}

func Test_LexerMovingThroughString(t *testing.T) {
	l := lexer.New("123", nil)
	run := []struct {
		s string
		r rune
	}{
		{"1", '1'},
		{"12", '2'},
		{"123", '3'},
		{"123", lexer.EOFRune},
	}

	for _, test := range run {
		r := l.Next()
		if r != test.r {
			t.Errorf("Expected %q but got %q", test.r, r)
			return
		}

		if l.Current() != test.s {
			t.Errorf("Expected %q but got %q", test.s, l.Current())
			return
		}
	}
}

func Test_LexingNumbers(t *testing.T) {
	l := lexer.New("123", NumberState)
	l.Start()
	tok, done := l.NextToken()
	if done {
		t.Error("Expected a token, but lexer was finished")
		return
	}

	if tok.Type != NumberToken {
		t.Errorf("Expected a %v but got %v", NumberToken, tok.Type)
		return
	}

	if tok.Value != "123" {
		t.Errorf("Expected %q but got %q", "123", tok.Value)
		return
	}

	tok, done = l.NextToken()
	if !done {
		t.Error("Expected the lexer to be done, but it wasn't.")
		return
	}

	if tok != nil {
		t.Errorf("Expected a nil token, but got %v", *tok)
		return
	}
}

func Test_LexerRewind(t *testing.T) {
	l := lexer.New("1", nil)
	r := l.Next()
	if r != '1' {
		t.Errorf("Expected %q but got %q", '1', r)
		return
	}

	if l.Current() != "1" {
		t.Errorf("Expected %q but got %q", "1", l.Current())
		return
	}

	l.Rewind()
	if l.Current() != "" {
		t.Errorf("Expected empty string, but got %q", l.Current())
		return
	}
}

func Test_MultipleTokens(t *testing.T) {
	cases := []struct {
		tokType lexer.TokenType
		val     string
	}{
		{NumberToken, "123"},
		{OpToken, "."},
		{IdentToken, "hello"},
		{NumberToken, "675"},
		{OpToken, "."},
		{IdentToken, "world"},
	}

	l := lexer.New("123.hello  675.world", NumberState)
	l.Start()

	for _, c := range cases {
		tok, done := l.NextToken()
		if done {
			t.Error("Expected there to be more tokens, but there weren't")
			return
		}

		if c.tokType != tok.Type {
			t.Errorf("Expected token type %v but got %v", c.tokType, tok.Type)
			return
		}

		if c.val != tok.Value {
			t.Errorf("Expected %q but got %q", c.val, tok.Value)
			return
		}
	}

	tok, done := l.NextToken()
	if !done {
		t.Error("Expected the lexer to be done, but it wasn't.")
		return
	}

	if tok != nil {
		t.Errorf("Did not expect a token, but got %v", *tok)
		return
	}
}

func Test_LexerError(t *testing.T) {
	l := lexer.New("1", WhitespaceState)
	l.ErrorHandler = func(e string) {}
	l.Start()

	tok, done := l.NextToken()
	if !done {
		t.Error("Expected token to be done, but it wasn't.")
		return
	}

	if tok != nil {
		t.Errorf("Expected no token, but got %v", *tok)
		return
	}

	if l.Err == nil {
		t.Error("Expected an error to be on the lexer, but none found.")
		return
	}

	if l.Err.Error() != "unexpected token '1'" {
		t.Errorf("Expected specific message from error, but got %q", l.Err.Error())
		return
	}
}

func Test_LexerCanTake(t *testing.T) {
	l := lexer.New("123.hello",
		func(l *lexer.L) lexer.StateFunc {

			if l.CanTake("1") {
				l.Take("1")
				l.Emit(NumberToken)
				return nil
			}

			l.Error("CanTake failed")
			return nil
		},
	)

	l.Start()
	l.NextToken()
}

func acceptNumber(number string) lexer.StateFunc {
	return func(l *lexer.L) lexer.StateFunc {

		if l.Accept(number) {
			l.Take("0123456789")
			l.Emit(NumberToken)
			return nil
		}

		l.Error("CanTake failed")
		return nil
	}
}

func Test_LexerAccept(t *testing.T) {
	shouldSucceed := []*lexer.L{
		lexer.New("123.hello", acceptNumber("123")),
		lexer.New("2234234.hello", acceptNumber("2234234")),
		lexer.New("3.hello", acceptNumber("3")),
		lexer.New("48765.hello", acceptNumber("48765")),
		lexer.New("51.hello", acceptNumber("51")),
	}

	shouldFail := []*lexer.L{
		lexer.New("1.hello", acceptNumber("0")),
	}

	for _, l := range shouldSucceed {
		l.Start()
		tok, done := l.NextToken()
		if tok == nil {
			t.Errorf("Expected non-nil token but got nil")
			return
		}
		if done {
			t.Errorf("Expected lexer to accept more tokens but got done")
			return
		}
	}

	for _, l := range shouldFail {
		l.ErrorHandler = func(string) {}
		l.Start()
		tok, done := l.NextToken()
		if tok != nil {
			t.Errorf("Expected nil token")
			return
		}
		if !done {
			t.Errorf("Expected lexer to accept more tokens but got done")
			return
		}
		if l.Err == nil {
			t.Errorf("Expected err to be set but got nil")
			return
		}
	}
}
