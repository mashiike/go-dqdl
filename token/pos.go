package token

import "fmt"

// Pos は元の入力テキストのバイト位置、行、列を表します。
// Pos represents a byte position in the original input text from which
type Pos struct {
	Index  int // index of the token in the input string
	Line   int // line number of the token, starting at 1
	Column int // column number of the token, starting at 1
}

// NoPos は無効な位置を表します。
// NoPos is the zero value for Pos; there is no file and line 1, column 1
var NoPos Pos

// IsValid reports whether the position is valid.
func (pos Pos) IsValid() bool {
	return pos.Line > 0
}

// String returns a string in one of several forms:
func (pos Pos) String() string {
	if !pos.IsValid() {
		return "-"
	}
	if pos.Column == 0 {
		return fmt.Sprintf("%d", pos.Line)
	}
	return fmt.Sprintf("%d:%d", pos.Line, pos.Column)
}

// AddColumn adds the given number of columns to the position.
func (pos Pos) AddColumn(n int) Pos {
	pos.Index += n
	pos.Column += n
	return pos
}

// AddLine adds the given number of lines to the position.
func (pos Pos) AddLine(n int) Pos {
	pos.Index += n
	pos.Line += n
	pos.Column = 1
	return pos
}

// Ptr returns a pointer to pos.
func (pos Pos) Ptr() *Pos {
	return &pos
}
