package lexer_test

import (
	"fmt"
	"testing"

	"github.com/tvanriel/go-lexer"
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

	if l.Err.Error() != "lexer (pos=1,2): unexpected token '1'" {
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

var latinAlphabet = "abcdefghijklmnopqrstuvxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func LexWord(l *lexer.L) lexer.StateFunc {
	if !l.CanTake(latinAlphabet) {
		l.Error(fmt.Sprintf("Expected latin alphabet character, got %v", l.Peek()))
	}
	l.Take(latinAlphabet)
	l.Emit(1)
	if l.CanTake(whitespace) {
		return LexWhitespace
	}
	if l.CanTake(punctuation) {
		return LexPunctuation
	}
	l.Error("Expected Punctuation or Whitespace")
	return nil
}

var punctuation = ",.\"'`"

func LexPunctuation(l *lexer.L) lexer.StateFunc {
	if !l.CanTake(punctuation) {
		l.Error(fmt.Sprintf("Expected punctuation, got %v", l.Peek()))
	}
	l.Take(punctuation)
	l.Emit(2)
	if l.CanTake(whitespace) {
		return LexWhitespace
	}
	if l.CanTake(latinAlphabet) {
		return LexWord
	}
	l.Error("Expected Punctuation or Whitespace")
	return nil
}

var whitespace = " \r\n\t"

func LexWhitespace(l *lexer.L) lexer.StateFunc {
	if !l.CanTake(whitespace) {
		l.Error(fmt.Sprintf("Expected whitespace, got %v", l.Peek()))
	}
	l.Take(whitespace)
	l.Emit(3)
	if l.CanTake(punctuation) {
		return LexPunctuation
	}
	if l.CanTake(latinAlphabet) {
		return LexWord
	}
	l.Error("Expected Punctuation or Word")
	return nil
}

var testtext = `Lorem ipsum dolor sit amet, consectetur adipiscing elit.
Maecenas nec accumsan orci, id venenatis nunc.
Donec et porttitor ligula, id suscipit nibh.
Phasellus suscipit eu tortor rutrum molestie.
Quisque quam elit, laoreet laoreet iaculis nec,
ultrices quis elit.
~
Mauris efficitur laoreet sapien,
in facilisis tortor feugiat eu.
Nam lobortis lobortis lectus ac cursus.
Pellentesque vehicula magna non molestie rutrum.
`

var expectedErrorText = `lexer:    4: Phasellus suscipit eu tortor rutrum molestie.
lexer:    5: Quisque quam elit, laoreet laoreet iaculis nec,
lexer:    6: ultrices quis elit.
lexer:    7: ~
lexer:     : ^ Expected Punctuation or Word
lexer:    8: Mauris efficitur laoreet sapien,
lexer:    9: in facilisis tortor feugiat eu.
lexer:   10: Nam lobortis lobortis lectus ac cursus.
`

func Test_LexerErrorPrettyPrint(t *testing.T) {
	l := lexer.New(testtext, LexWord)
	l.ErrorHandler = func(e string) {
		var err = l.PrettyError(e)
		if err != expectedErrorText {
			t.Errorf("Unexpected format for error:\n%v\n", err)
		}
	}
	l.StartSync()

}
