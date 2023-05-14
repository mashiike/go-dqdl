// The ast package declares the types used to represent syntax trees for DQDL.
package ast

import (
	"strings"

	"github.com/mashiike/go-dqdl/token"
)

// 全ての抽象構文木のノードはこのインタフェースを実装します。
// All nodes in the abstract syntax tree implement this interface.
type Node interface {
	Pos() token.Pos
	End() token.Pos
}

// パラメータを表すノードです。
// A Parameter node represents a parameter.
type Parameter interface {
	Node
	parameterNode()
}

// 表現を表すノードです。
// An Expression node represents an expression.
type Expression interface {
	Node
	expressionNode()
}

// with threasholdの右側に用いれる表現を表すノードです。
// An Expression node represents an expression.
type ThresholdExpression interface {
	Expression
	thresholdExpressionNode()
}

// with thresholdの左側に用いれる表現を表すノードです。
// An Expression node represents an expression.
type ThresholdTarget interface {
	Expression
	thresholdTargetNode()
}

// コメントを表すノードです。
// A Comment node represents a comment.
type Comment struct {
	SharpPos token.Pos // position of "#"
	Text     string    // comment text
}

func (c *Comment) Pos() token.Pos { return c.SharpPos }
func (c *Comment) End() token.Pos {
	return c.SharpPos.AddColumn(len(c.Text))
}

// コメントの塊を表すノードです。
// A CommentGroup represents a group of comments.
type CommentGroup []*Comment

func (g CommentGroup) Pos() token.Pos {
	if len(g) > 0 {
		return g[0].Pos()
	}
	return token.NoPos
}
func (g CommentGroup) End() token.Pos {
	if len(g) > 0 {
		return g[len(g)-1].End()
	}
	return token.NoPos
}

// ファイルを表すノードです。
// A File node represents a DQDL source file.
type File struct {
	Filename      string
	CommentGroups []CommentGroup // list of comments
	Rulesets      []*Ruleset     // list of rulesets
}

// Ruleset の宣言を表すノードです。
// A Ruleset node represents a Ruleset declaration.
type Ruleset struct {
	Description     CommentGroup   // comments before "Rules"
	DeclPos         token.Pos      // position of "Rules" keyword
	LeftBracketPos  token.Pos      // position of "["
	Rules           []RuleDecl     // list of rules
	InnerComments   []CommentGroup // comments inside "[...]"
	RightBracketPos token.Pos      // position of "]"
	Comments        CommentGroup   // list of comments
}

func (d *Ruleset) Pos() token.Pos { return d.DeclPos }
func (d *Ruleset) End() token.Pos {
	if d.Comments != nil {
		return d.Comments.End()
	}
	return d.RightBracketPos.AddColumn(1)
}

type RuleDecl interface {
	Node
	PrependComments(CommentGroup)
	ruleDeclNode()
}

// Ruleの宣言を表すノードです。
// A Rule node represents a Rule declaration.
type Rule struct {
	Description CommentGroup // comments before RuleType
	Type        *Ident       // position of RuleType
	Parameters  []Parameter  // list of parameters
	Expression  Expression   // expression
	Comments    CommentGroup // line comments
}

func (r *Rule) Pos() token.Pos { return r.Type.Pos() }
func (r *Rule) End() token.Pos {
	if r.Expression != nil {
		return r.Expression.End()
	}
	if len(r.Parameters) > 0 {
		return r.Parameters[len(r.Parameters)-1].End()
	}
	return r.Type.End()
}
func (r *Rule) ruleDeclNode() {}
func (r *Rule) PrependComments(comments CommentGroup) {
	r.Comments = append(comments, r.Comments...)
}

type CombinedRule struct {
	Description    CommentGroup // comments before first "("
	FirstLParenPos token.Pos    // position of first "("
	LastRParenPos  token.Pos    // position of last ")"
	Rules          []*Rule      // list of rules
	Operator       string       // operator and/or
	Comments       CommentGroup // line comments
}

func (r *CombinedRule) Pos() token.Pos { return r.FirstLParenPos }
func (r *CombinedRule) End() token.Pos {
	return r.LastRParenPos.AddColumn(1)
}
func (r *CombinedRule) ruleDeclNode() {}
func (r *CombinedRule) PrependComments(comments CommentGroup) {
	r.Comments = append(comments, r.Comments...)
}

// 識別子を表すノードです。
// An Ident node represents an identifier.
type Ident struct {
	NamePos  token.Pos    // identifier position
	Name     string       // identifier name
	Comments CommentGroup // list of comments
}

func (x *Ident) Pos() token.Pos { return x.NamePos }
func (x *Ident) End() token.Pos {
	return x.NamePos.AddColumn(len(x.Name))
}

type StringParameter struct {
	LeftQuotePos  token.Pos    // position of left quote
	RightQuotePos token.Pos    // position of right quote
	Value         string       // string value
	Comments      CommentGroup // list of comments
}

func (x *StringParameter) Pos() token.Pos { return x.LeftQuotePos }
func (x *StringParameter) End() token.Pos {
	return x.RightQuotePos.AddColumn(1)
}
func (x *StringParameter) parameterNode() {}

type NumberParameter struct {
	NumberPos token.Pos // position of number
	Value     string    // number value
	Comments  CommentGroup
}

func (x *NumberParameter) Pos() token.Pos { return x.NumberPos }
func (x *NumberParameter) End() token.Pos {
	return x.NumberPos.AddColumn(len(x.Value))
}
func (x *NumberParameter) parameterNode() {}
func (x *NumberParameter) IsInteger() bool {
	return !strings.ContainsRune(x.Value, '.')
}

type BoolParameter struct {
	BoolPos  token.Pos    // position of bool
	Value    bool         // bool value
	Comments CommentGroup // list of comments
}

func (x *BoolParameter) Pos() token.Pos { return x.BoolPos }
func (x *BoolParameter) End() token.Pos {
	if x.Value {
		return x.BoolPos.AddColumn(4)
	}
	return x.BoolPos.AddColumn(5)
}
func (x *BoolParameter) parameterNode() {}

type DurationParameter struct {
	NumberPos token.Pos    // position of number
	UnitPos   token.Pos    // position of unit
	Value     string       // duration value
	Number    string       // number
	Unit      string       // unit
	Comments  CommentGroup // list of comments
}

func (x *DurationParameter) Pos() token.Pos { return x.NumberPos }
func (x *DurationParameter) End() token.Pos {
	return x.UnitPos.AddColumn(len(x.Unit))
}
func (x *DurationParameter) parameterNode() {}

type DateParamter struct {
	LeftParenPos  *token.Pos         // position of left paren
	RightParenPos *token.Pos         // position of right paren
	NowPos        token.Pos          // position of now()
	MinusPos      *token.Pos         // position of minus
	Duration      *DurationParameter // duration
	Comments      CommentGroup       // list of comments
}

func (x *DateParamter) Pos() token.Pos {
	if x.LeftParenPos != nil {
		return *x.LeftParenPos
	}
	return x.NowPos
}
func (x *DateParamter) End() token.Pos {
	if x.RightParenPos != nil {
		return x.RightParenPos.AddColumn(1)
	}
	return x.NowPos.AddColumn(6)
}
func (x *DateParamter) parameterNode() {}

// ComparisonExpressionは比較表現を表すノードです。
// A ComparisonExpression node represents a comparison expression.
type ComparisonExpression struct {
	ExprPos  token.Pos    // position of expression
	Operator string       // operator
	Right    Parameter    // parameter
	Comments CommentGroup // list of comments
}

func (x *ComparisonExpression) Pos() token.Pos { return x.ExprPos }
func (x *ComparisonExpression) End() token.Pos {
	return x.Right.End()
}
func (x *ComparisonExpression) expressionNode()          {}
func (x *ComparisonExpression) thresholdExpressionNode() {}

// BetweenExpressionはbetween表現を表すノードです。
// A BetweenExpression node represents a between expression.
type BetweenExpression struct {
	ExprPos  token.Pos    // position of expression
	Left     Parameter    // parameter
	Right    Parameter    // parameter
	Comments CommentGroup // list of comments
}

func (x *BetweenExpression) Pos() token.Pos { return x.ExprPos }
func (x *BetweenExpression) End() token.Pos {
	return x.Right.End()
}
func (x *BetweenExpression) expressionNode()          {}
func (x *BetweenExpression) thresholdExpressionNode() {}

// InExpressionはin表現を表すノードです。ex: in [1,2,3]
// A InExpression node represents a in expression.
type InExpression struct {
	ExprPos         token.Pos    // position of expression
	LeftBracketPos  token.Pos    // position of left bracket
	RightBracketPos token.Pos    // position of right bracket
	Values          []Parameter  // list of values
	Comments        CommentGroup // list of comments
}

func (x *InExpression) Pos() token.Pos { return x.ExprPos }
func (x *InExpression) End() token.Pos {
	return x.RightBracketPos.AddColumn(1)
}
func (x *InExpression) expressionNode()      {}
func (x *InExpression) thresholdTargetNode() {}

// MatchesExpressionはmatches表現を表すノードです。
// A MatchesExpression node represents a matches expression.
type MatchesExpression struct {
	ExprPos   token.Pos    // position of expression
	RegexpPos token.Pos    // position of regexp
	Value     string       // regexp value
	Comments  CommentGroup // list of comments
}

func (x *MatchesExpression) Pos() token.Pos { return x.ExprPos }
func (x *MatchesExpression) End() token.Pos {
	return x.RegexpPos.AddColumn(len(x.Value) + 2)
}
func (x *MatchesExpression) expressionNode()      {}
func (x *MatchesExpression) thresholdTargetNode() {}

type WithThresholdExpression struct {
	ExprPos   token.Pos           // position of expression
	Target    ThresholdTarget     // target
	Threshold ThresholdExpression // threshold expression
	Comments  CommentGroup        // list of comments
}

func (x *WithThresholdExpression) Pos() token.Pos { return x.Target.Pos() }
func (x *WithThresholdExpression) End() token.Pos {
	return x.Threshold.End()
}
func (x *WithThresholdExpression) expressionNode() {}
