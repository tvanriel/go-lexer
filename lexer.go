// This package provides a Lexer that functions similarly to Rob Pike's discussion
// about lexer design in this [talk](https://www.youtube.com/watch?v=HxaD_trXwRE).
//
// You can define your token types by using the `lexer.TokenType` type (`int`) via
//
//     const (
//             StringToken lexer.TokenType = iota
//             IntegerToken
//             // etc...
//     )
//
// And then you define your own state functions (`lexer.StateFunc`) to handle
// analyzing the string.
//
//     func StringState(l *lexer.L) lexer.StateFunc {
//             l.Next() // eat starting "
//             l.Ignore() // drop current value
//             for l.Peek() != '"' {
//                     l.Next()
//             }
//             l.Emit(StringToken)
//
//             return SomeStateFunction
//     }
//
// This Lexer is meant to emit tokens in such a fashion that it can be consumed
// by go yacc.
package lexer

import (
	"fmt"
	"io"
	"os"
	"strings"
	"unicode/utf8"
)

type StateFunc func(*L) StateFunc

type TokenType int

const (
	EOFRune    rune      = -1
	EmptyToken TokenType = 0
)

type Token struct {
	Type  TokenType
	Value string
}

type L struct {
	source       *sourcetext
	startState   StateFunc
	Err          error
	tokens       chan Token
	ErrorHandler func(e string)
	rewind       runeStack
}

// New creates a returns a lexer ready to parse the given source code.
func New(src string, start StateFunc) *L {
	return &L{
		source:     newSourceText(src),
		startState: start,
		rewind:     newRuneStack(),
	}
}

// Start begins executing the Lexer in an asynchronous manner (using a goroutine).
func (l *L) Start() {
	// Take half the string length as a buffer size.
	buffSize := l.source.len() / 2
	if buffSize <= 0 {
		buffSize = 1
	}
	l.tokens = make(chan Token, buffSize)
	go l.run()
}

func (l *L) StartSync() {
	// Take half the string length as a buffer size.
	buffSize := l.source.len() / 2
	if buffSize <= 0 {
		buffSize = 1
	}
	l.tokens = make(chan Token, buffSize)
	l.run()
}

// Current returns the value being being analyzed at this moment.
func (l *L) Current() string {
	return l.source.current()
}

// Emit will receive a token type and push a new token with the current analyzed
// value into the tokens channel.
func (l *L) Emit(t TokenType) {
	tok := Token{
		Type:  t,
		Value: l.Current(),
	}
	l.tokens <- tok
	l.source.update()
	l.rewind.clear()
}

// Ignore clears the rewind stack and then sets the current beginning position
// to the current position in the source which effectively ignores the section
// of the source being analyzed.
func (l *L) Ignore() {
	l.rewind.clear()
	l.source.update()
}

// Peek performs a Next operation immediately followed by a Rewind returning the
// peeked rune.
func (l *L) Peek() rune {
	r := l.Next()
	l.Rewind()

	return r
}

// Rewind will take the last rune read (if any) and rewind back. Rewinds can
// occur more than once per call to Next but you can never rewind past the
// last point a token was emitted.
func (l *L) Rewind() {
	r := l.rewind.pop()
	if r > EOFRune {
		l.source.rewind(r)
	}
}

// Next pulls the next rune from the Lexer and returns it, moving the position
// forward in the source.
func (l *L) Next() rune {
	var (
		r rune
		s int
	)
	str := l.source.fromHere()
	if len(str) == 0 {
		r, s = EOFRune, 0
	} else {
		r, s = utf8.DecodeRuneInString(str)
	}
	l.source.advance(s)
	l.rewind.push(r)

	return r
}

// Take receives a string containing all acceptable strings and will continue
// over each consecutive character in the source until a token not in the given
// string is encountered. This should be used to quickly pull token parts.
func (l *L) Take(chars string) {
	r := l.Next()
	for strings.ContainsRune(chars, r) {
		r = l.Next()
	}
	l.Rewind() // last next wasn't a match
}

// Accept receives a string and checks if the following characters match
// that string in order.
func (l *L) Accept(chars string) bool {
	return strings.HasPrefix(l.source.fromHere(), chars)
}

// CanTake receives a string and checks if the next rune is in that string.
func (l *L) CanTake(chars string) bool {
	return strings.ContainsRune(chars, l.Peek())
}

// NextToken returns the next token from the lexer and a value to denote whether
// or not the token is finished.
func (l *L) NextToken() (*Token, bool) {
	if tok, ok := <-l.tokens; ok {
		return &tok, false
	} else {
		return nil, true
	}
}

// Partial yyLexer implementation

func (l *L) Error(e string) {
	if l.ErrorHandler != nil {

		linenum, pos := l.source.getPos()
		l.Err = fmt.Errorf("lexer (pos=%d,%d): %v", linenum, pos, e)
		l.ErrorHandler(e)
	} else {
		panic(e)
	}
}

func (l *L) PrettyError(e string) string {
	var sb strings.Builder
	line, pos := l.source.getPos()
	before, linetext, after, beforeStart, afterStart := l.source.getContext(line - 1)

	if len(before) > 0 {
		i := beforeStart + 1
		for _, l := range before {
			sb.WriteString(fmt.Sprintf("lexer: %4d: %s\n", i, l))
			i++
		}
	}

	sb.WriteString(fmt.Sprintf("lexer: %4d: %s\n", line, linetext))
	sb.WriteString(fmt.Sprintf("lexer:     :%s^ %s\n", strings.Repeat(" ", pos), e))

	if len(after) > 0 {
		i := afterStart + 1
		for _, l := range after {
			sb.WriteString(fmt.Sprintf("lexer: %4d: %s\n", i, l))
			i++
		}
	}

	return sb.String()
}

func (l *L) writeError(to io.Writer, e string) {
	fmt.Fprint(to, l.PrettyError(e))
}

func (l *L) PrintError(e string) {
	l.writeError(os.Stdout, e)
}

// Private methods

func (l *L) run() {
	state := l.startState
	for state != nil {
		state = state(l)
	}
	close(l.tokens)
}
