package parser

import (
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/mashiike/go-dqdl/ast"
	"github.com/mashiike/go-dqdl/token"
)

func TestParseRule(t *testing.T) {
	cases := []struct {
		name   string
		input  string
		want   ast.RuleDecl
		errStr string
	}{
		{
			name:   "empty",
			input:  "",
			errStr: "syntax error near 1:1 ``, RuleType is required: unexpexted EOF",
		},
		{
			name:   "lexer error",
			input:  `\" IsUnique "col-A"`,
			errStr: "syntax error near 1:1 `\\\" IsUnique \"col-A\"`, unrecognized character: U+005C '\\'",
		},
		{
			name:   "parameter only",
			input:  `"cal-A"`,
			errStr: "syntax error near 1:1 `\"cal-A\"`, RuleType is required: unexpected <Parameter>",
		},
		{
			name:   "expression only",
			input:  `between 1 and 5`,
			errStr: "syntax error near 1:1 `between 1 and 5`, RuleType is required: unexpected <Expression>",
		},
		{
			name:  "is_unique",
			input: `IsUnique "col-A"`,
			want: &ast.Rule{
				Type: &ast.Ident{
					NamePos: token.Pos{Index: 0, Line: 1, Column: 1},
					Name:    "IsUnique",
				},
				Parameters: []ast.Parameter{
					&ast.StringParameter{
						LeftQuotePos:  token.Pos{Index: 9, Line: 1, Column: 10},
						Value:         "col-A",
						RightQuotePos: token.Pos{Index: 15, Line: 1, Column: 16},
					},
				},
			},
		},
		{
			name: "is_unique_with_comment",
			input: `
				# this is sample of DQDL rule

				# IsUnique rule checks whether all of the values in a column are unique, and returns a Boolean value.
				IsUnique    # RuleType
				            # IsUnique accept 1 column parameter
				    "col-A" # ColumnName
				# more details: https://docs.aws.amazon.com/glue/latest/dg/dqdl.html#dqdl-rule-types-IsUnique
			`,
			want: &ast.Rule{
				Description: ast.CommentGroup{
					&ast.Comment{
						SharpPos: token.Pos{Index: 40, Line: 4, Column: 5},
						Text:     "# IsUnique rule checks whether all of the values in a column are unique, and returns a Boolean value.",
					},
				},
				Type: &ast.Ident{
					NamePos: token.Pos{Index: 146, Line: 5, Column: 5},
					Name:    "IsUnique",
					Comments: ast.CommentGroup{
						&ast.Comment{
							SharpPos: token.Pos{Index: 158, Line: 5, Column: 17},
							Text:     "# RuleType",
						},
						&ast.Comment{
							SharpPos: token.Pos{Index: 185, Line: 6, Column: 17},
							Text:     "# IsUnique accept 1 column parameter",
						},
					},
				},
				Parameters: []ast.Parameter{
					&ast.StringParameter{
						LeftQuotePos:  token.Pos{Index: 230, Line: 7, Column: 9},
						Value:         "col-A",
						RightQuotePos: token.Pos{Index: 236, Line: 7, Column: 15},
						Comments: ast.CommentGroup{
							&ast.Comment{
								SharpPos: token.Pos{Index: 238, Line: 7, Column: 17},
								Text:     "# ColumnName",
							},
						},
					},
				},
			},
		},
		{
			name:  "is_unique_with_paren",
			input: `(IsUnique "col-A")`,
			want: &ast.Rule{
				Type: &ast.Ident{
					NamePos: token.Pos{Index: 1, Line: 1, Column: 2},
					Name:    "IsUnique",
				},
				Parameters: []ast.Parameter{
					&ast.StringParameter{
						LeftQuotePos:  token.Pos{Index: 10, Line: 1, Column: 11},
						Value:         "col-A",
						RightQuotePos: token.Pos{Index: 16, Line: 1, Column: 17},
					},
				},
			},
		},
		{
			name:  "expression less then",
			input: `ColumnCorrelation "colA" "colB" < 0.5`,
			want: &ast.Rule{
				Type: &ast.Ident{
					NamePos: token.Pos{Index: 0, Line: 1, Column: 1},
					Name:    "ColumnCorrelation",
				},
				Parameters: []ast.Parameter{
					&ast.StringParameter{
						LeftQuotePos:  token.Pos{Index: 18, Line: 1, Column: 19},
						Value:         "colA",
						RightQuotePos: token.Pos{Index: 23, Line: 1, Column: 24},
					},
					&ast.StringParameter{
						LeftQuotePos:  token.Pos{Index: 25, Line: 1, Column: 26},
						Value:         "colB",
						RightQuotePos: token.Pos{Index: 30, Line: 1, Column: 31},
					},
				},
				Expression: &ast.ComparisonExpression{
					ExprPos:  token.Pos{Index: 32, Line: 1, Column: 33},
					Operator: "<",
					Right: &ast.NumberParameter{
						NumberPos: token.Pos{Index: 34, Line: 1, Column: 35},
						Value:     "0.5",
					},
				},
			},
		},
		{
			name: "is_unique_before_comment_and_line_comment",
			input: `# comment
			IsUnique "col-A"  # line comment`,
			want: &ast.Rule{
				Description: ast.CommentGroup{
					&ast.Comment{
						SharpPos: token.Pos{Index: 0, Line: 1, Column: 1},
						Text:     "# comment",
					},
				},
				Comments: ast.CommentGroup{
					&ast.Comment{
						SharpPos: token.Pos{Index: 31, Line: 2, Column: 22},
						Text:     "# line comment",
					},
				},
				Type: &ast.Ident{
					NamePos: token.Pos{Index: 13, Line: 2, Column: 4},
					Name:    "IsUnique",
				},
				Parameters: []ast.Parameter{
					&ast.StringParameter{
						LeftQuotePos:  token.Pos{Index: 22, Line: 2, Column: 13},
						Value:         "col-A",
						RightQuotePos: token.Pos{Index: 28, Line: 2, Column: 19},
					},
				},
			},
		},
		{
			name:  "time parameter",
			input: `DataFreshness "Order_Date" <= 24 hours`,
			want: &ast.Rule{
				Type: &ast.Ident{
					NamePos: token.Pos{Index: 0, Line: 1, Column: 1},
					Name:    "DataFreshness",
				},
				Parameters: []ast.Parameter{
					&ast.StringParameter{
						LeftQuotePos:  token.Pos{Index: 14, Line: 1, Column: 15},
						Value:         "Order_Date",
						RightQuotePos: token.Pos{Index: 25, Line: 1, Column: 26},
					},
				},
				Expression: &ast.ComparisonExpression{
					ExprPos:  token.Pos{Index: 27, Line: 1, Column: 28},
					Operator: "<=",
					Right: &ast.DurationParameter{
						NumberPos: token.Pos{Index: 30, Line: 1, Column: 31},
						UnitPos:   token.Pos{Index: 33, Line: 1, Column: 34},
						Value:     "24 hours",
						Number:    "24",
						Unit:      "hours",
					},
				},
			},
		},
		{
			name:  "between duration",
			input: `DataFreshness "Order_Date" between 2 days and 5 days`,
			want: &ast.Rule{
				Type: &ast.Ident{
					NamePos: token.Pos{Index: 0, Line: 1, Column: 1},
					Name:    "DataFreshness",
				},
				Parameters: []ast.Parameter{
					&ast.StringParameter{
						LeftQuotePos:  token.Pos{Index: 14, Line: 1, Column: 15},
						Value:         "Order_Date",
						RightQuotePos: token.Pos{Index: 25, Line: 1, Column: 26},
					},
				},
				Expression: &ast.BetweenExpression{
					ExprPos: token.Pos{Index: 27, Line: 1, Column: 28},
					Left: &ast.DurationParameter{
						NumberPos: token.Pos{Index: 35, Line: 1, Column: 36},
						UnitPos:   token.Pos{Index: 37, Line: 1, Column: 38},
						Value:     "2 days",
						Number:    "2",
						Unit:      "days",
					},
					Right: &ast.DurationParameter{
						NumberPos: token.Pos{Index: 46, Line: 1, Column: 47},
						UnitPos:   token.Pos{Index: 48, Line: 1, Column: 49},
						Value:     "5 days",
						Number:    "5",
						Unit:      "days",
					},
				},
			},
		},
		{
			name:  "in expression",
			input: `ColumnValues "colA" in [ "a", "b", "c" ]`,
			want: &ast.Rule{
				Type: &ast.Ident{
					NamePos: token.Pos{Index: 0, Line: 1, Column: 1},
					Name:    "ColumnValues",
				},
				Parameters: []ast.Parameter{
					&ast.StringParameter{
						LeftQuotePos:  token.Pos{Index: 13, Line: 1, Column: 14},
						Value:         "colA",
						RightQuotePos: token.Pos{Index: 18, Line: 1, Column: 19},
					},
				},
				Expression: &ast.InExpression{
					ExprPos:         token.Pos{Index: 20, Line: 1, Column: 21},
					LeftBracketPos:  token.Pos{Index: 23, Line: 1, Column: 24},
					RightBracketPos: token.Pos{Index: 39, Line: 1, Column: 40},
					Values: []ast.Parameter{
						&ast.StringParameter{
							LeftQuotePos:  token.Pos{Index: 25, Line: 1, Column: 26},
							Value:         "a",
							RightQuotePos: token.Pos{Index: 27, Line: 1, Column: 28},
						},
						&ast.StringParameter{
							LeftQuotePos:  token.Pos{Index: 30, Line: 1, Column: 31},
							Value:         "b",
							RightQuotePos: token.Pos{Index: 32, Line: 1, Column: 33},
						},
						&ast.StringParameter{
							LeftQuotePos:  token.Pos{Index: 35, Line: 1, Column: 36},
							Value:         "c",
							RightQuotePos: token.Pos{Index: 37, Line: 1, Column: 38},
						},
					},
				},
			},
		},
		{
			name:  "matches expression",
			input: `ColumnValues "colA" matches "[a-ZA-Z]*"`,
			want: &ast.Rule{
				Type: &ast.Ident{
					NamePos: token.Pos{Index: 0, Line: 1, Column: 1},
					Name:    "ColumnValues",
				},
				Parameters: []ast.Parameter{
					&ast.StringParameter{
						LeftQuotePos:  token.Pos{Index: 13, Line: 1, Column: 14},
						Value:         "colA",
						RightQuotePos: token.Pos{Index: 18, Line: 1, Column: 19},
					},
				},
				Expression: &ast.MatchesExpression{
					ExprPos:   token.Pos{Index: 20, Line: 1, Column: 21},
					RegexpPos: token.Pos{Index: 28, Line: 1, Column: 29},
					Value:     "[a-ZA-Z]*",
				},
			},
		},
		{
			name:  "now expression",
			input: `ColumnValues "load_date" > (now() - 3 days)`,
			want: &ast.Rule{
				Type: &ast.Ident{
					NamePos: token.Pos{Index: 0, Line: 1, Column: 1},
					Name:    "ColumnValues",
				},
				Parameters: []ast.Parameter{
					&ast.StringParameter{
						LeftQuotePos:  token.Pos{Index: 13, Line: 1, Column: 14},
						Value:         "load_date",
						RightQuotePos: token.Pos{Index: 23, Line: 1, Column: 24},
					},
				},
				Expression: &ast.ComparisonExpression{
					ExprPos:  token.Pos{Index: 25, Line: 1, Column: 26},
					Operator: ">",
					Right: &ast.DateParamter{
						LeftParenPos:  &token.Pos{Index: 27, Line: 1, Column: 28},
						RightParenPos: &token.Pos{Index: 42, Line: 1, Column: 43},
						NowPos:        token.Pos{Index: 28, Line: 1, Column: 29},
						MinusPos:      &token.Pos{Index: 34, Line: 1, Column: 35},
						Duration: &ast.DurationParameter{
							NumberPos: token.Pos{Index: 36, Line: 1, Column: 37},
							UnitPos:   token.Pos{Index: 38, Line: 1, Column: 39},
							Value:     "3 days",
							Number:    "3",
							Unit:      "days",
						},
					},
				},
			},
		},
		{
			name:  "now expression with line comment",
			input: `ColumnValues "load_date" > (now() - 3 days) #line comment`,
			want: &ast.Rule{
				Type: &ast.Ident{
					NamePos: token.Pos{Index: 0, Line: 1, Column: 1},
					Name:    "ColumnValues",
				},
				Parameters: []ast.Parameter{
					&ast.StringParameter{
						LeftQuotePos:  token.Pos{Index: 13, Line: 1, Column: 14},
						Value:         "load_date",
						RightQuotePos: token.Pos{Index: 23, Line: 1, Column: 24},
					},
				},
				Expression: &ast.ComparisonExpression{
					ExprPos:  token.Pos{Index: 25, Line: 1, Column: 26},
					Operator: ">",
					Right: &ast.DateParamter{
						LeftParenPos:  &token.Pos{Index: 27, Line: 1, Column: 28},
						RightParenPos: &token.Pos{Index: 42, Line: 1, Column: 43},
						NowPos:        token.Pos{Index: 28, Line: 1, Column: 29},
						MinusPos:      &token.Pos{Index: 34, Line: 1, Column: 35},
						Duration: &ast.DurationParameter{
							NumberPos: token.Pos{Index: 36, Line: 1, Column: 37},
							UnitPos:   token.Pos{Index: 38, Line: 1, Column: 39},
							Value:     "3 days",
							Number:    "3",
							Unit:      "days",
						},
					},
				},
				Comments: ast.CommentGroup{
					&ast.Comment{
						SharpPos: token.Pos{Index: 44, Line: 1, Column: 45},
						Text:     "#line comment",
					},
				},
			},
		},
		{
			name:  "only now",
			input: `ColumnValues "load_date" <= now()`,
			want: &ast.Rule{
				Type: &ast.Ident{
					NamePos: token.Pos{Index: 0, Line: 1, Column: 1},
					Name:    "ColumnValues",
				},
				Parameters: []ast.Parameter{
					&ast.StringParameter{
						LeftQuotePos:  token.Pos{Index: 13, Line: 1, Column: 14},
						Value:         "load_date",
						RightQuotePos: token.Pos{Index: 23, Line: 1, Column: 24},
					},
				},
				Expression: &ast.ComparisonExpression{
					ExprPos:  token.Pos{Index: 25, Line: 1, Column: 26},
					Operator: "<=",
					Right: &ast.DateParamter{
						NowPos: token.Pos{Index: 28, Line: 1, Column: 29},
					},
				},
			},
		},
		{
			name:  "matches expression with threshold",
			input: `ColumnValues "colA" matches "[a-zA-Z]*" with threshold between 0.2 and 0.9`,
			want: &ast.Rule{
				Type: &ast.Ident{
					NamePos: token.Pos{Index: 0, Line: 1, Column: 1},
					Name:    "ColumnValues",
				},
				Parameters: []ast.Parameter{
					&ast.StringParameter{
						LeftQuotePos:  token.Pos{Index: 13, Line: 1, Column: 14},
						Value:         "colA",
						RightQuotePos: token.Pos{Index: 18, Line: 1, Column: 19},
					},
				},
				Expression: &ast.WithThresholdExpression{
					ExprPos: token.Pos{Index: 40, Line: 1, Column: 41},
					Target: &ast.MatchesExpression{
						ExprPos:   token.Pos{Index: 20, Line: 1, Column: 21},
						RegexpPos: token.Pos{Index: 28, Line: 1, Column: 29},
						Value:     "[a-zA-Z]*",
					},
					Threshold: &ast.BetweenExpression{
						ExprPos: token.Pos{Index: 55, Line: 1, Column: 56},
						Left: &ast.NumberParameter{
							NumberPos: token.Pos{Index: 63, Line: 1, Column: 64},
							Value:     "0.2",
						},
						Right: &ast.NumberParameter{
							NumberPos: token.Pos{Index: 71, Line: 1, Column: 72},
							Value:     "0.9",
						},
					},
				},
			},
		},
		{
			name:  "in expression with threshold",
			input: `ColumnValues "colA" in ["A", "B"] with threshold > 0.8`,
			want: &ast.Rule{
				Type: &ast.Ident{
					NamePos: token.Pos{Index: 0, Line: 1, Column: 1},
					Name:    "ColumnValues",
				},
				Parameters: []ast.Parameter{
					&ast.StringParameter{
						LeftQuotePos:  token.Pos{Index: 13, Line: 1, Column: 14},
						Value:         "colA",
						RightQuotePos: token.Pos{Index: 18, Line: 1, Column: 19},
					},
				},
				Expression: &ast.WithThresholdExpression{
					ExprPos: token.Pos{Index: 34, Line: 1, Column: 35},
					Target: &ast.InExpression{
						ExprPos:         token.Pos{Index: 20, Line: 1, Column: 21},
						LeftBracketPos:  token.Pos{Index: 23, Line: 1, Column: 24},
						RightBracketPos: token.Pos{Index: 32, Line: 1, Column: 33},
						Values: []ast.Parameter{
							&ast.StringParameter{
								LeftQuotePos:  token.Pos{Index: 24, Line: 1, Column: 25},
								Value:         "A",
								RightQuotePos: token.Pos{Index: 26, Line: 1, Column: 27},
							},
							&ast.StringParameter{
								LeftQuotePos:  token.Pos{Index: 29, Line: 1, Column: 30},
								Value:         "B",
								RightQuotePos: token.Pos{Index: 31, Line: 1, Column: 32},
							},
						},
					},
					Threshold: &ast.ComparisonExpression{
						ExprPos:  token.Pos{Index: 49, Line: 1, Column: 50},
						Operator: ">",
						Right: &ast.NumberParameter{
							NumberPos: token.Pos{Index: 51, Line: 1, Column: 52},
							Value:     "0.8",
						},
					},
				},
			},
		},
		{
			name:   "combined mix rule",
			input:  `(IsUnique "col-A") and (IsPrimaryKey "col-A") or (IsUnique "col-B") and (IsPrimaryKey "col-B")`,
			errStr: "syntax error near 1:47: ` or (IsUnique \"col-B...`, can not mixed `and` and `or`",
		},
		{
			name:  "combined and rule",
			input: `(IsUnique "col-A") and (IsUnique "col-B")`,
			want: &ast.CombinedRule{
				FirstLParenPos: token.Pos{Index: 0, Line: 1, Column: 1},
				LastRParenPos:  token.Pos{Index: 40, Line: 1, Column: 41},
				Operator:       "and",
				Rules: []*ast.Rule{
					{
						Type: &ast.Ident{
							NamePos: token.Pos{Index: 1, Line: 1, Column: 2},
							Name:    "IsUnique",
						},
						Parameters: []ast.Parameter{
							&ast.StringParameter{
								LeftQuotePos:  token.Pos{Index: 10, Line: 1, Column: 11},
								Value:         "col-A",
								RightQuotePos: token.Pos{Index: 16, Line: 1, Column: 17},
							},
						},
					},
					{
						Type: &ast.Ident{
							NamePos: token.Pos{Index: 24, Line: 1, Column: 25},
							Name:    "IsUnique",
						},
						Parameters: []ast.Parameter{
							&ast.StringParameter{
								LeftQuotePos:  token.Pos{Index: 33, Line: 1, Column: 34},
								Value:         "col-B",
								RightQuotePos: token.Pos{Index: 39, Line: 1, Column: 40},
							},
						},
					},
				},
			},
		},
		{
			name:  "combined or rule",
			input: `(IsUnique "col-A") or (IsPrimaryKey "col-A")`,
			want: &ast.CombinedRule{
				FirstLParenPos: token.Pos{Index: 0, Line: 1, Column: 1},
				LastRParenPos:  token.Pos{Index: 43, Line: 1, Column: 44},
				Operator:       "or",
				Rules: []*ast.Rule{
					{
						Type: &ast.Ident{
							NamePos: token.Pos{Index: 1, Line: 1, Column: 2},
							Name:    "IsUnique",
						},
						Parameters: []ast.Parameter{
							&ast.StringParameter{
								LeftQuotePos:  token.Pos{Index: 10, Line: 1, Column: 11},
								Value:         "col-A",
								RightQuotePos: token.Pos{Index: 16, Line: 1, Column: 17},
							},
						},
					},
					{
						Type: &ast.Ident{
							NamePos: token.Pos{Index: 23, Line: 1, Column: 24},
							Name:    "IsPrimaryKey",
						},
						Parameters: []ast.Parameter{
							&ast.StringParameter{
								LeftQuotePos:  token.Pos{Index: 36, Line: 1, Column: 37},
								Value:         "col-A",
								RightQuotePos: token.Pos{Index: 42, Line: 1, Column: 43},
							},
						},
					},
				},
			},
		},
		{
			name: "combined or rule with comment",
			input: `# comment
			# comment 2
			(IsUnique "col-A") or (IsPrimaryKey "col-A") or (IsUnique "col-B") or (IsPrimaryKey "col-B")`,
			want: &ast.CombinedRule{
				Description: ast.CommentGroup{
					&ast.Comment{
						SharpPos: token.Pos{Index: 0, Line: 1, Column: 1},
						Text:     "# comment",
					},
					&ast.Comment{
						SharpPos: token.Pos{Index: 13, Line: 2, Column: 4},
						Text:     "# comment 2",
					},
				},
				FirstLParenPos: token.Pos{Index: 28, Line: 3, Column: 4},
				LastRParenPos:  token.Pos{Index: 119, Line: 3, Column: 95},
				Operator:       "or",
				Rules: []*ast.Rule{
					{
						Type: &ast.Ident{
							NamePos: token.Pos{Index: 29, Line: 3, Column: 5},
							Name:    "IsUnique",
						},
						Parameters: []ast.Parameter{
							&ast.StringParameter{
								LeftQuotePos:  token.Pos{Index: 38, Line: 3, Column: 14},
								Value:         "col-A",
								RightQuotePos: token.Pos{Index: 44, Line: 3, Column: 20},
							},
						},
					},
					{
						Type: &ast.Ident{
							NamePos: token.Pos{Index: 51, Line: 3, Column: 27},
							Name:    "IsPrimaryKey",
						},
						Parameters: []ast.Parameter{
							&ast.StringParameter{
								LeftQuotePos:  token.Pos{Index: 64, Line: 3, Column: 40},
								Value:         "col-A",
								RightQuotePos: token.Pos{Index: 70, Line: 3, Column: 46},
							},
						},
					},
					{
						Type: &ast.Ident{
							NamePos: token.Pos{Index: 77, Line: 3, Column: 53},
							Name:    "IsUnique",
						},
						Parameters: []ast.Parameter{
							&ast.StringParameter{
								LeftQuotePos:  token.Pos{Index: 86, Line: 3, Column: 62},
								Value:         "col-B",
								RightQuotePos: token.Pos{Index: 92, Line: 3, Column: 68},
							},
						},
					},
					{
						Type: &ast.Ident{
							NamePos: token.Pos{Index: 99, Line: 3, Column: 75},
							Name:    "IsPrimaryKey",
						},
						Parameters: []ast.Parameter{
							&ast.StringParameter{
								LeftQuotePos:  token.Pos{Index: 112, Line: 3, Column: 88},
								Value:         "col-B",
								RightQuotePos: token.Pos{Index: 118, Line: 3, Column: 94},
							},
						},
					},
				},
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := ParseRule(c.input)
			if c.errStr != "" {
				if err == nil {
					t.Errorf("expected error %q, got nil", c.errStr)
					return
				}
				if got != nil {
					t.Errorf("got %v, want nil", got)
					return
				}
				if err.Error() != c.errStr {
					t.Errorf("got error %q, want %q", err.Error(), c.errStr)
				}
				return
			}
			if c.errStr == "" {
				if err != nil {
					t.Errorf("unexpected error: %s", err)

					return
				}
				if got == nil {
					t.Errorf("got nil, want %v", c.want)
					return
				}
			}
			if diff := cmp.Diff(got, c.want); diff != "" {
				t.Errorf("unexpected result (-got +want):\n%s", diff)
			}
		})
	}
}

type parserRulesetTestCase struct {
	name   string
	input  string
	want   *ast.Ruleset
	errStr string
}

func (c *parserRulesetTestCase) Run(t *testing.T) {
	t.Helper()
	got, err := ParseRuleset(c.input)
	if c.errStr != "" {
		if err == nil {
			t.Errorf("expected error %q, got nil", c.errStr)
			return
		}
		if got != nil {
			t.Errorf("got %v, want nil", got)
			return
		}
		if err.Error() != c.errStr {
			t.Errorf("got error %q, want %q", err.Error(), c.errStr)
		}
		return
	}
	if c.errStr == "" {
		if err != nil {
			t.Errorf("unexpected error: %s", err)

			return
		}
		if got == nil {
			t.Errorf("got nil, want %v", c.want)
			return
		}
	}
	if diff := cmp.Diff(got, c.want); diff != "" {
		t.Errorf("unexpected result (-got +want):\n%s", diff)
	}
}

func TestParseRuleset__Invalid(t *testing.T) {
	cases := []parserRulesetTestCase{
		{
			name:   "empty",
			input:  ``,
			errStr: "no rules found",
		},
		{
			name:   "missing left bracket",
			input:  `Rules =`,
			errStr: "syntax error near 1:1 `Rules =`, missing `[`",
		},
		{
			name:   "missing right bracket",
			input:  `Rules = [`,
			errStr: "syntax error near 1:10 `[`, missing `]`",
		},
	}
	for _, c := range cases {
		t.Run(c.name, c.Run)
	}
}

func TestParseRuleset__EmptyRuleset(t *testing.T) {
	input := `
Rules = [

]`
	want := &ast.Ruleset{
		DeclPos:         token.Pos{Index: 1, Line: 2, Column: 1},
		LeftBracketPos:  token.Pos{Index: 9, Line: 2, Column: 9},
		RightBracketPos: token.Pos{Index: 12, Line: 4, Column: 1},
	}
	c := parserRulesetTestCase{
		name:  "empty ruleset",
		input: input,
		want:  want,
	}
	c.Run(t)
}

func TestParseRuleset__SimpleOneRule(t *testing.T) {
	input := `
Rules = [
	IsUnique "col-A"
]`
	want := &ast.Ruleset{
		DeclPos:         token.Pos{Index: 1, Line: 2, Column: 1},
		LeftBracketPos:  token.Pos{Index: 9, Line: 2, Column: 9},
		RightBracketPos: token.Pos{Index: 29, Line: 4, Column: 1},
		Rules: []ast.RuleDecl{
			&ast.Rule{
				Type: &ast.Ident{
					NamePos: token.Pos{Index: 12, Line: 3, Column: 2},
					Name:    "IsUnique",
				},
				Parameters: []ast.Parameter{
					&ast.StringParameter{
						LeftQuotePos:  token.Pos{Index: 21, Line: 3, Column: 11},
						Value:         "col-A",
						RightQuotePos: token.Pos{Index: 27, Line: 3, Column: 17},
					},
				},
			},
		},
	}
	c := &parserRulesetTestCase{
		input: input,
		want:  want,
	}
	c.Run(t)
}

func TestParseRuleset__SimpleTwoRule(t *testing.T) {
	input := `
Rules = [
	IsComplete "order-id",
	IsUnique "order-id"
]`
	want := &ast.Ruleset{
		DeclPos:         token.Pos{Index: 1, Line: 2, Column: 1},
		LeftBracketPos:  token.Pos{Index: 9, Line: 2, Column: 9},
		RightBracketPos: token.Pos{Index: 56, Line: 5, Column: 1},
		Rules: []ast.RuleDecl{
			&ast.Rule{
				Type: &ast.Ident{
					NamePos: token.Pos{Index: 12, Line: 3, Column: 2},
					Name:    "IsComplete",
				},
				Parameters: []ast.Parameter{
					&ast.StringParameter{
						LeftQuotePos:  token.Pos{Index: 23, Line: 3, Column: 13},
						Value:         "order-id",
						RightQuotePos: token.Pos{Index: 32, Line: 3, Column: 22},
					},
				},
			},
			&ast.Rule{
				Type: &ast.Ident{
					NamePos: token.Pos{Index: 36, Line: 4, Column: 2},
					Name:    "IsUnique",
				},
				Parameters: []ast.Parameter{
					&ast.StringParameter{
						LeftQuotePos:  token.Pos{Index: 45, Line: 4, Column: 11},
						Value:         "order-id",
						RightQuotePos: token.Pos{Index: 54, Line: 4, Column: 20},
					},
				},
			},
		},
	}
	c := &parserRulesetTestCase{
		input: input,
		want:  want,
	}
	c.Run(t)
}

func TestParseRuleset__RulesetWithComment(t *testing.T) {
	input := `# This is a file comment

# this is a ruleset description
Rules = [
	# this is a IsComplete rule description
	IsComplete "order-id",

	# this is Ruleset inner comment

	# this is a IsUnique rule description
	IsUnique "order-id"
] #this is ruleset comment

# this is last file comment`

	want := &ast.Ruleset{
		Description: ast.CommentGroup{
			&ast.Comment{
				SharpPos: token.Pos{Index: 26, Line: 3, Column: 1},
				Text:     "# this is a ruleset description",
			},
		},
		DeclPos:         token.Pos{Index: 58, Line: 4, Column: 1},
		LeftBracketPos:  token.Pos{Index: 66, Line: 4, Column: 9},
		RightBracketPos: token.Pos{Index: 228, Line: 12, Column: 1},
		Rules: []ast.RuleDecl{
			&ast.Rule{
				Description: ast.CommentGroup{
					&ast.Comment{
						SharpPos: token.Pos{Index: 69, Line: 5, Column: 2},
						Text:     "# this is a IsComplete rule description",
					},
				},
				Type: &ast.Ident{
					NamePos: token.Pos{Index: 110, Line: 6, Column: 2},
					Name:    "IsComplete",
				},
				Parameters: []ast.Parameter{
					&ast.StringParameter{
						LeftQuotePos:  token.Pos{Index: 121, Line: 6, Column: 13},
						Value:         "order-id",
						RightQuotePos: token.Pos{Index: 130, Line: 6, Column: 22},
					},
				},
			},
			&ast.Rule{
				Description: ast.CommentGroup{
					&ast.Comment{
						SharpPos: token.Pos{Index: 169, Line: 10, Column: 2},
						Text:     "# this is a IsUnique rule description",
					},
				},
				Type: &ast.Ident{
					NamePos: token.Pos{Index: 208, Line: 11, Column: 2},
					Name:    "IsUnique",
				},
				Parameters: []ast.Parameter{
					&ast.StringParameter{
						LeftQuotePos:  token.Pos{Index: 217, Line: 11, Column: 11},
						Value:         "order-id",
						RightQuotePos: token.Pos{Index: 226, Line: 11, Column: 20},
					},
				},
			},
		},
		InnerComments: []ast.CommentGroup{
			{
				&ast.Comment{
					SharpPos: token.Pos{Index: 135, Line: 8, Column: 2},
					Text:     "# this is Ruleset inner comment",
				},
			},
		},
		Comments: ast.CommentGroup{
			&ast.Comment{
				SharpPos: token.Pos{Index: 230, Line: 12, Column: 3},
				Text:     "#this is ruleset comment",
			},
		},
	}
	c := &parserRulesetTestCase{
		input: input,
		want:  want,
	}
	c.Run(t)
}

var update = flag.Bool("update", false, "update golden files")

func TestParseFile(t *testing.T) {
	filename := "testdata/sample.dqdl"
	fp, err := os.Open(filename)
	if err != nil {
		t.Fatal(err)
	}
	defer fp.Close()

	astFile, err := ParseFile(filename, fp)
	if err != nil {
		t.Fatal(err)
	}
	got, err := json.MarshalIndent(astFile, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	golden := filepath.Join("testdata/", t.Name()+".golden")

	if *update {
		os.WriteFile(golden, got, 0644)
	}

	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatal(err)
	}

	if diff := cmp.Diff(string(want), string(got)); diff != "" {
		t.Errorf("ParseFile(%s) mismatch (-want +got):\n%s", filename, diff)
	}
}
