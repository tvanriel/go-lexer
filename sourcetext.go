package lexer

import (
	"strings"
	"unicode/utf8"
)

type sourcetext struct {
	source string
	pos    int
	start  int
}

func newSourceText(s string) *sourcetext {
	return &sourcetext{
		source: s,
		pos:    0,
	}
}

func (s *sourcetext) sourceString() string {
	return s.source
}

func (s *sourcetext) fromHere() string {
	return s.source[s.pos:]
}

func (s *sourcetext) untilHere() string {
	return s.source[:s.pos]
}

func (s *sourcetext) lines() []string {
	return strings.Split(s.source, "\n")
}

func (s *sourcetext) inc() {
	s.advance(1)
}

func (s *sourcetext) advance(by int) {
	s.pos += by
}

func (s *sourcetext) update() {
	s.start = s.pos
}
func (s *sourcetext) len() int {
	return len(s.source)
}

func (s *sourcetext) current() string {
	return s.source[s.start:s.pos]
}

func (s *sourcetext) rewind(r rune) {
	size := utf8.RuneLen(r)
	s.pos -= size
	if s.pos < s.start {
		s.update()
	}
}

// Get the line number and position in that line the lexer position is currently on.
func (s *sourcetext) getPos() (int, int) {
	untilNow := s.untilHere()
	linenum := strings.Count(untilNow, "\n") + 1
	lastNewLineIndex := clamp(strings.LastIndex(untilNow, "\n"), 0, s.pos)
	posInLine := s.pos - lastNewLineIndex
	return linenum, posInLine
}

func clamp(num, min, max int) int {
	if min > max {
		return 0
	}
	if num < min {
		return min
	}
	if num > max {
		return max
	}
	return num
}

func (s *sourcetext) getContext(l int) (before []string, line string, after []string, beforeStart, afterStart int) {
	lines := s.lines()

	beforeStart = clamp(l-3, 0, len(lines))
	beforeEnd := clamp(l, beforeStart, l)

	afterStart = clamp(l+1, 0, len(lines))
	afterEnd := clamp(l+4, afterStart, len(lines))

	if l == beforeStart {
		before = []string{}
	} else {
		before = lines[beforeStart:beforeEnd]
	}

	if l == afterEnd {
		after = []string{}
	} else {
		after = lines[afterStart:afterEnd]
	}
	line = lines[l]
	return

}
