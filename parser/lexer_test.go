package parser

import (
	"context"
	"testing"

	"github.com/mashiike/go-dqdl/token"
)

func makeToken(typ token.TokenType, lit string) token.Token {
	return token.Token{Type: typ, Value: lit}
}

func assertToken(t *testing.T, want, got token.Token) {
	t.Helper()
	if want.Type != got.Type {
		t.Errorf("want token type %s, got %s", want.Type, got.Type)
	}
	if want.Value != got.Value {
		t.Errorf("want token Value %s, got %s", want.Value, got.Value)
	}
}

func TestLexer(t *testing.T) {
	// table driven tests
	cases := []struct {
		name   string
		input  string
		tokens []token.Token
	}{
		{name: "empty", input: "", tokens: []token.Token{makeToken(token.EOF, "")}},
		{name: "whitespace", input: " \t\n", tokens: []token.Token{makeToken(token.EOF, "")}},
		{name: "number", input: "123", tokens: []token.Token{makeToken(token.NUMBER, "123"), makeToken(token.EOF, "")}},
		{name: "string", input: `"abc"`, tokens: []token.Token{makeToken(token.STRING, `"abc"`), makeToken(token.EOF, "")}},
		{name: "unterminated string", input: `"abc`, tokens: []token.Token{makeToken(token.ILLEGAL, "unterminated string")}},
		{name: "identifier", input: `abc`, tokens: []token.Token{makeToken(token.IDENT, "abc"), makeToken(token.EOF, "")}},
		{name: "identifier with whitespace", input: `abc 123`, tokens: []token.Token{makeToken(token.IDENT, "abc"), makeToken(token.NUMBER, "123"), makeToken(token.EOF, "")}},
		{name: "identifier can not start number", input: `123abc`, tokens: []token.Token{makeToken(token.ILLEGAL, "invalid number")}},
		{name: "number with dot", input: `123.456`, tokens: []token.Token{makeToken(token.NUMBER, "123.456"), makeToken(token.EOF, "")}},
		{name: "invalid number", input: `123.456.789`, tokens: []token.Token{makeToken(token.ILLEGAL, "invalid number")}},
		{name: "bool", input: `true false`, tokens: []token.Token{makeToken(token.TRUE, "true"), makeToken(token.FALSE, "false"), makeToken(token.EOF, "")}},
		{
			name: "comment",
			input: `# this is comment
					123.5# hoge fuga piyo
					# foo bar baz
					#################`,
			tokens: []token.Token{
				makeToken(token.COMMENT, "# this is comment"),
				makeToken(token.NUMBER, "123.5"),
				makeToken(token.COMMENT, "# hoge fuga piyo"),
				makeToken(token.COMMENT, "# foo bar baz"),
				makeToken(token.COMMENT, "#################"),
				makeToken(token.EOF, ""),
			},
		},
		{
			name:  "hours",
			input: `DataFreshness "col-A" > 3 hours`,
			tokens: []token.Token{
				makeToken(token.IDENT, "DataFreshness"),
				makeToken(token.STRING, `"col-A"`),
				makeToken(token.GREATER_THAN, ">"),
				makeToken(token.NUMBER, "3"),
				makeToken(token.HOURS, "hours"),
				makeToken(token.EOF, ""),
			},
		},
		{
			name:  "grater equal",
			input: `ColumnValues "colA" >= 10`,
			tokens: []token.Token{
				makeToken(token.IDENT, "ColumnValues"),
				makeToken(token.STRING, `"colA"`),
				makeToken(token.GREATER_EQUAL, ">="),
				makeToken(token.NUMBER, "10"),
				makeToken(token.EOF, ""),
			},
		},
		{
			name:  "less equal",
			input: `ColumnValues "colA" <= 10`,
			tokens: []token.Token{
				makeToken(token.IDENT, "ColumnValues"),
				makeToken(token.STRING, `"colA"`),
				makeToken(token.LESS_EQUAL, "<="),
				makeToken(token.NUMBER, "10"),
				makeToken(token.EOF, ""),
			},
		},
		{
			name:  "equal",
			input: `ColumnValues "colA" = "2022-06-30"`,
			tokens: []token.Token{
				makeToken(token.IDENT, "ColumnValues"),
				makeToken(token.STRING, `"colA"`),
				makeToken(token.EQUAL, "="),
				makeToken(token.STRING, `"2022-06-30"`),
				makeToken(token.EOF, ""),
			},
		},
		{
			name:  "matches",
			input: `ColumnValues "colA" matches "[a-ZA-Z]*"`,
			tokens: []token.Token{
				makeToken(token.IDENT, "ColumnValues"),
				makeToken(token.STRING, `"colA"`),
				makeToken(token.MATCHES, "matches"),
				makeToken(token.STRING, `"[a-ZA-Z]*"`),
				makeToken(token.EOF, ""),
			},
		},
		{
			name:  "between",
			input: `Mean "colA" between 80 and 100`,
			tokens: []token.Token{
				makeToken(token.IDENT, "Mean"),
				makeToken(token.STRING, `"colA"`),
				makeToken(token.BETWEEN, "between"),
				makeToken(token.NUMBER, "80"),
				makeToken(token.AND, "and"),
				makeToken(token.NUMBER, "100"),
				makeToken(token.EOF, ""),
			},
		},
		{
			name:  "in",
			input: `ColumnValues "colA" in ["a", "b", "c"]`,
			tokens: []token.Token{
				makeToken(token.IDENT, "ColumnValues"),
				makeToken(token.STRING, `"colA"`),
				makeToken(token.IN, "in"),
				makeToken(token.LEFT_BRACKET, "["),
				makeToken(token.STRING, `"a"`),
				makeToken(token.COMMA, ","),
				makeToken(token.STRING, `"b"`),
				makeToken(token.COMMA, ","),
				makeToken(token.STRING, `"c"`),
				makeToken(token.RIGHT_BRACKET, "]"),
				makeToken(token.EOF, ""),
			},
		},
		{
			name:  "now",
			input: `ColumnValues "load_date" > (now() - 3 days)`,
			tokens: []token.Token{
				makeToken(token.IDENT, "ColumnValues"),
				makeToken(token.STRING, `"load_date"`),
				makeToken(token.GREATER_THAN, ">"),
				makeToken(token.LEFT_PAREN, "("),
				makeToken(token.NOW, "now()"),
				makeToken(token.MINUS, "-"),
				makeToken(token.NUMBER, "3"),
				makeToken(token.DAYS, "days"),
				makeToken(token.RIGHT_PAREN, ")"),
				makeToken(token.EOF, ""),
			},
		},
		{
			name:  "matches with threshold",
			input: `ColumnValues "colA" matches "[a-zA-Z]*" with threshold between 0.2 and 0.9`,
			tokens: []token.Token{
				makeToken(token.IDENT, "ColumnValues"),
				makeToken(token.STRING, `"colA"`),
				makeToken(token.MATCHES, "matches"),
				makeToken(token.STRING, `"[a-zA-Z]*"`),
				makeToken(token.WITH, "with"),
				makeToken(token.THRESHOLD, "threshold"),
				makeToken(token.BETWEEN, "between"),
				makeToken(token.NUMBER, "0.2"),
				makeToken(token.AND, "and"),
				makeToken(token.NUMBER, "0.9"),
				makeToken(token.EOF, ""),
			},
		},
		{
			name:  "rule and combination",
			input: `(Mean "Star_Rating" > 3) and (Mean "Order_Total" > 500) and (IsComplete "Order_Id")`,
			tokens: []token.Token{
				makeToken(token.LEFT_PAREN, "("),
				makeToken(token.IDENT, "Mean"),
				makeToken(token.STRING, `"Star_Rating"`),
				makeToken(token.GREATER_THAN, ">"),
				makeToken(token.NUMBER, "3"),
				makeToken(token.RIGHT_PAREN, ")"),
				makeToken(token.AND, "and"),
				makeToken(token.LEFT_PAREN, "("),
				makeToken(token.IDENT, "Mean"),
				makeToken(token.STRING, `"Order_Total"`),
				makeToken(token.GREATER_THAN, ">"),
				makeToken(token.NUMBER, "500"),
				makeToken(token.RIGHT_PAREN, ")"),
				makeToken(token.AND, "and"),
				makeToken(token.LEFT_PAREN, "("),
				makeToken(token.IDENT, "IsComplete"),
				makeToken(token.STRING, `"Order_Id"`),
				makeToken(token.RIGHT_PAREN, ")"),
				makeToken(token.EOF, ""),
			},
		},
		{
			name:  "rule or combination",
			input: `(RowCount "id" < 100) or (IsPrimaryKey "id")`,
			tokens: []token.Token{
				makeToken(token.LEFT_PAREN, "("),
				makeToken(token.IDENT, "RowCount"),
				makeToken(token.STRING, `"id"`),
				makeToken(token.LESS_THAN, "<"),
				makeToken(token.NUMBER, "100"),
				makeToken(token.RIGHT_PAREN, ")"),
				makeToken(token.OR, "or"),
				makeToken(token.LEFT_PAREN, "("),
				makeToken(token.IDENT, "IsPrimaryKey"),
				makeToken(token.STRING, `"id"`),
				makeToken(token.RIGHT_PAREN, ")"),
				makeToken(token.EOF, ""),
			},
		},
		{
			name: "rules",
			input: `Rules = [
				IsUnique "order-id",
				IsComplete "order-id"
			]`,
			tokens: []token.Token{
				makeToken(token.RULES, "Rules"),
				makeToken(token.EQUAL, "="),
				makeToken(token.LEFT_BRACKET, "["),
				makeToken(token.IDENT, "IsUnique"),
				makeToken(token.STRING, `"order-id"`),
				makeToken(token.COMMA, ","),
				makeToken(token.IDENT, "IsComplete"),
				makeToken(token.STRING, `"order-id"`),
				makeToken(token.RIGHT_BRACKET, "]"),
				makeToken(token.EOF, ""),
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Logf("input: %s", c.input)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			l := newLexer(c.name, c.input)
			cleanup := l.run(ctx)
			defer cleanup()
			for i, expected := range c.tokens {
				actual, ok := <-l.tokens
				if !ok {
					t.Fatalf("expected %d token is %s, got closed channel", i, expected)
				}
				assertToken(t, expected, actual)

			}
			for actual := range l.tokens {
				t.Error("unexpected token:", actual)
			}
		})
	}
}
