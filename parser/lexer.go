package parser

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/mashiike/go-dqdl/token"
)

const eof = -1

type stateFn func(context.Context, *lexer) stateFn

// lexer of DQDL(Declarative Query Definition Language).
type lexer struct {
	name        string           // used only for error reports.
	input       string           // the string being scanned.
	start       int              // start position of this item.
	startLine   int              // start line of this item.
	startCol    int              // start column of this item.
	pos         int              // current position in the input.
	line        int              // 1+number of newlines seen.
	col         int              // 1+number of characters seen on this line.
	prevLineCol int              // 1+number of characters seen on previous line.
	width       int              // width of last rune read from input.
	tokens      chan token.Token // channel of scanned tokens.
}

// newLexer creates a new scanner for the input string.
func newLexer(name, input string) *lexer {
	return &lexer{
		name:      name,
		input:     input,
		line:      1,
		startLine: 1,
		col:       1,
		startCol:  1,
		tokens:    make(chan token.Token),
	}
}

// run lexes the input by executing state functions until
//
// it is in the background using go routine.
func (l *lexer) run(ctx context.Context) func() {
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer func() {
			close(l.tokens) // No more tokens will be delivered.
			wg.Done()
		}()
		for state := lexRule; state != nil; {
			select {
			case <-ctx.Done():
				return
			default:
			}
			state = state(ctx, l)
		}
	}()
	return wg.Wait
}

// TokenChan returns the channel on which tokens are delivered.
func (l *lexer) TokenChan() chan token.Token {
	return l.tokens
}

// String returns the input string being scanned.
func (l *lexer) String() string {
	return l.name
}

// emit passes an token back to the client.
func (l *lexer) emit(t token.TokenType) {
	l.tokens <- token.Token{
		Type:  t,
		Value: l.input[l.start:l.pos],
		Start: token.Pos{
			Index:  l.start,
			Line:   l.startLine,
			Column: l.startCol,
		},
		End: token.Pos{
			Index:  l.pos,
			Line:   l.line,
			Column: l.col,
		},
	}
	l.start = l.pos
	l.startLine = l.line
	l.startCol = l.col
}

// next returns the next rune in the input.
func (l *lexer) next() rune {
	l.col++
	if int(l.pos) >= len(l.input) {
		l.width = 0
		return eof // EOF
	}
	r, w := utf8.DecodeRuneInString(l.input[l.pos:])
	l.width = w
	l.pos += l.width
	if r == '\n' {
		l.line++
		l.prevLineCol = l.col
		l.col = 1
	}

	return r
}

// ignore skips over the pending input before this point.
func (l *lexer) ignore() {
	l.start = l.pos
	l.startLine = l.line
	l.startCol = l.col
}

// backup steps back one rune. Can only be called once per call of next.
func (l *lexer) backup() {
	l.pos -= l.width
	l.col--
	if l.col < 1 {
		l.line--
		l.col = l.prevLineCol
	}
}

// accept consumes the next rune if it's from the valid set.
func (l *lexer) accept(valid string) bool {
	if strings.ContainsRune(valid, l.next()) {
		return true
	}
	l.backup()
	return false
}

// errorf returns an error token and terminates the scan by passing
// back a nil pointer that will be the next state, terminating l.run.
func (l *lexer) errorf(format string, args ...interface{}) stateFn {
	l.tokens <- token.Token{
		Type:  token.ILLEGAL,
		Value: fmt.Sprintf(format, args...),
		Start: token.Pos{
			Index:  l.start,
			Line:   l.startLine,
			Column: l.startCol,
		},
		End: token.Pos{
			Index:  l.pos,
			Line:   l.line,
			Column: l.col,
		},
	}
	return nil
}

// lexRule scans the input for a rule.
func lexRule(ctx context.Context, l *lexer) stateFn {
	for {
		select {
		case <-ctx.Done():
			l.errorf("canceled")
			return nil
		default:
		}
		switch r := l.next(); {
		case r == eof:
			l.emit(token.EOF)
			return nil
		case isSpace(r):
			l.ignore()
		case isLetter(r):
			l.backup()
			return lexIdentifier
		case r == '"':
			return lexString
		case isDigit(r):
			l.backup()
			return lexNumber
		case r == '#':
			return lexComment
		case r == '(':
			l.emit(token.LEFT_PAREN)
		case r == ')':
			l.emit(token.RIGHT_PAREN)
		case r == ',':
			l.emit(token.COMMA)
		case r == '+':
			l.emit(token.PLUS)
		case r == '-':
			l.emit(token.MINUS)
		case r == '*':
			l.emit(token.MULTIPLY)
		case r == '/':
			l.emit(token.DIVIDE)
		case r == '=':
			l.emit(token.EQUAL)
		case r == ']':
			l.emit(token.RIGHT_BRACKET)
		case r == '[':
			l.emit(token.LEFT_BRACKET)
		case r == '>':
			if l.accept("=") {
				l.emit(token.GREATER_EQUAL)
			} else {
				l.emit(token.GREATER_THAN)
			}
		case r == '<':
			if l.accept("=") {
				l.emit(token.LESS_EQUAL)
			} else {
				l.emit(token.LESS_THAN)
			}
		default:
			return l.errorf("unrecognized character: %#U", r)
		}
	}
}

// lexComment scans a comment.
func lexComment(ctx context.Context, l *lexer) stateFn {
	for {
		select {
		case <-ctx.Done():
			l.errorf("canceled")
			return nil
		default:
		}
		switch r := l.next(); {
		case r == '\n':
			l.backup()
			l.emit(token.COMMENT)
			return lexRule
		case r == eof:
			l.backup()
			l.emit(token.COMMENT)
			return lexRule
		default:
			// absorb.
		}
	}
}

// lexIdentifier scans an alphanumeric.
func lexIdentifier(ctx context.Context, l *lexer) stateFn {
	for {
		select {
		case <-ctx.Done():
			l.errorf("canceled")
			return nil
		default:
		}
		switch r := l.next(); {
		case isLetter(r):
			// absorb.
		default:
			l.backup()
			keyword := l.input[l.start:l.pos]
			t := token.LookupIdent(keyword)
			if t == token.NOW {
				// NOW is a special case, it can be followed by '()'.
				if r := l.next(); r != '(' {
					return l.errorf("expected '()' after NOW")
				}
				if r := l.next(); r != ')' {
					return l.errorf("expected '()' after NOW")
				}
			}
			l.emit(t)
			return lexRule
		}
	}
}

// lexString scans a quoted string.
func lexString(ctx context.Context, l *lexer) stateFn {
	for {
		select {
		case <-ctx.Done():
			l.errorf("canceled")
			return nil
		default:
		}
		switch r := l.next(); {
		case r == eof:
			return l.errorf("unterminated string")
		case r == '"':
			l.emit(token.STRING)
			return lexRule
		default:
			// absorb.
		}
	}
}

// lexNumber scans a number.
func lexNumber(ctx context.Context, l *lexer) stateFn {
	var seenDot bool
	for {
		select {
		case <-ctx.Done():
			l.errorf("canceled")
			return nil
		default:
		}
		switch r := l.next(); {
		case isDigit(r):
			// absorb.
		case r == '.':
			if seenDot {
				return l.errorf("invalid number")
			}
			seenDot = true
		case isLetter(r):
			return l.errorf("invalid number")
		default:
			l.backup()
			l.emit(token.NUMBER)
			return lexRule
		}
	}
}

// isSpace reports whether r is a space character.
func isSpace(r rune) bool {
	return r == ' ' || r == '\t' || r == '\n'
}

// isLetter reports whether r is a letter.
func isLetter(r rune) bool {
	return 'a' <= r && r <= 'z' || 'A' <= r && r <= 'Z'
}

// isDigit reports whether r is a digit.
func isDigit(r rune) bool {
	return '0' <= r && r <= '9'
}
