// Package tokenはDQDLの字句トークンを表す定数と、トークンに対する基本的な操作を定義します。
// Package token defines constants representing the lexical tokens of DQDL and basic operations on tokens (printing, predicates).
package token

import "fmt"

// TokenTypeはトークンの種類を表します。
// TokenType represents the type of a token.
type TokenType int

const (
	ILLEGAL TokenType = iota
	EOF
	IDENT
	NUMBER
	STRING
	BETWEEN
	OR
	AND
	RIGHT_BRACKET
	LEFT_BRACKET
	GREATER_THAN
	LESS_THAN
	EQUAL
	GREATER_EQUAL
	LESS_EQUAL
	RIGHT_PAREN
	LEFT_PAREN
	IN
	MATCHES
	NOW
	DAYS
	HOURS
	WITH
	THRESHOLD
	COMMA
	TRUE
	FALSE
	PLUS
	MINUS
	MULTIPLY
	DIVIDE
	RULES
	COMMENT
)

var tokenTypeStrings = map[TokenType]string{
	ILLEGAL:       "ILLEGAL",
	EOF:           "EOF",
	IDENT:         "IDENT",
	COMMENT:       "COMMENT",
	NUMBER:        "NUMBER",
	STRING:        "STRING",
	BETWEEN:       "between",
	AND:           "and",
	OR:            "or",
	RIGHT_BRACKET: "]",
	LEFT_BRACKET:  "[",
	GREATER_THAN:  ">",
	LESS_THAN:     "<",
	EQUAL:         "=",
	GREATER_EQUAL: ">=",
	LESS_EQUAL:    "<=",
	RIGHT_PAREN:   ")",
	LEFT_PAREN:    "(",
	IN:            "in",
	MATCHES:       "matches",
	NOW:           "now()",
	DAYS:          "days",
	HOURS:         "hours",
	WITH:          "with",
	THRESHOLD:     "threshold",
	COMMA:         ",",
	TRUE:          "true",
	FALSE:         "false",
	MINUS:         "-",
	RULES:         "Rules",
}

// Stringはトークンの種類を文字列で返します。
// String returns the string corresponding to the token tok.
func (t TokenType) String() string {
	if s, ok := tokenTypeStrings[t]; ok {
		return s
	}
	return "unknown token"
}

// IsExpressionStart はExpressionの始まりの字句であるかどうかを返します。
// IsExpressionStart returns true if the token is the start of an expression.
func (t TokenType) IsExpressionStart() bool {
	switch t {
	case BETWEEN, IN, MATCHES, EQUAL, GREATER_THAN, LESS_THAN, GREATER_EQUAL, LESS_EQUAL:
		return true
	default:
		return false
	}
}

// IsParameterAcceptable は パラメーターとして受け入れられる字句であるかどうかを返します。
// IsParameterAcceptable returns true if the token is acceptable as a parameter.
func (t TokenType) IsParameterAcceptable() bool {
	switch t {
	case NUMBER, STRING, TRUE, FALSE, NOW:
		return true
	default:
		return false
	}
}

// Tokenは字句トークンを表します。
// Token represents a token.
type Token struct {
	Type  TokenType
	Start Pos
	End   Pos
	Value string
}

// Stringはトークンの文字列表現を返します。
// String returns the string corresponding to the token tok.
func (t Token) String() string {
	return t.Value
}

// GoStringはトークンのGo言語の文字列表現を返します。
// GoString returns the Go string representation of the token tok.
func (t Token) GoString() string {
	return fmt.Sprintf("token.Token{Type: \"%s\", Start: \"%s\", End: \"%s\", Value: \"%s\"}", t.Type, t.Start, t.End, t.Value)
}

var keywords = map[string]TokenType{
	"between":   BETWEEN,
	"and":       AND,
	"or":        OR,
	"in":        IN,
	"matches":   MATCHES,
	"now":       NOW,
	"hours":     HOURS,
	"days":      DAYS,
	"with":      WITH,
	"threshold": THRESHOLD,
	"true":      TRUE,
	"false":     FALSE,
	"Rules":     RULES,
}

// LookupIdentは識別子として登録されている場合はそのトークンの種類を返します。そうでない場合はtoken.IDENTをかえします。
// LookupIdent returns the token type of the string s if it is a keyword, and token.IDENT otherwise.
func LookupIdent(ident string) TokenType {
	if tok, ok := keywords[ident]; ok {
		return tok
	}
	return IDENT
}
