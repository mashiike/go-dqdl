package parser

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/mashiike/go-dqdl/ast"
	"github.com/mashiike/go-dqdl/token"
)

type parser struct {
	input                string
	lexer                *lexer
	stack                []token.Token
	fileCommentGroups    []ast.CommentGroup
	rulesetCommentGroups []ast.CommentGroup
}

func ParseFile(filename string, reader io.Reader) (*ast.File, error) {
	bs, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	input := string(bs)
	p := &parser{
		input: input,
		lexer: newLexer(filename, input),
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	waiter := p.lexer.run(ctx)
	file, err := p.parseFile()
	if err != nil {
		cancel()
		p.discardUntilToken(token.EOF)
		waiter()
		return nil, err
	}
	file.Filename = filename
	p.discardUntilToken(token.EOF)
	waiter()
	return file, nil
}

var errNoRulesFound = errors.New("no rules found")

func (p *parser) parseFile() (*ast.File, error) {
	p.fileCommentGroups = nil
	file := &ast.File{}
	for {
		ruleset, err := p.parseRuleset()
		if err != nil {
			if err == errNoRulesFound && len(file.Rulesets) > 0 {
				break
			}
			return nil, err
		}
		file.Rulesets = append(file.Rulesets, ruleset)
		t, ok := p.pop()
		if !ok {
			break
		}
		if t.Type == token.EOF {
			break
		}
		p.push(t)
	}
	file.CommentGroups = p.fileCommentGroups
	return file, nil
}

// PaserRuleset はルールセットについての構文解析を行います。
// ParseRuleset parses a ruleset.
func ParseRuleset(rulesetStr string) (*ast.Ruleset, error) {
	p := &parser{
		input: rulesetStr,
		lexer: newLexer("ruleset", rulesetStr),
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	waiter := p.lexer.run(ctx)
	ruleset, err := p.parseRuleset()
	if err != nil {
		cancel()
		p.discardUntilToken(token.EOF)
		waiter()
		return nil, err
	}
	p.discardUntilToken(token.EOF)
	waiter()
	return ruleset, nil
}

func (p *parser) parseRuleset() (*ast.Ruleset, error) {
	ruleset := &ast.Ruleset{}
	var rulesFound bool
	var storedComments ast.CommentGroup
	var lastCommentPos token.Pos
	for {
		t, ok := p.pop()
		if !ok {
			return nil, errors.New("unexpected EOF")
		}
		switch t.Type {
		case token.EOF:
			if !rulesFound {
				return nil, errNoRulesFound
			}
			return nil, fmt.Errorf("syntax error near %s `%s`, missing `]`", t.Start, p.nearString(t.Start))
		case token.RIGHT_BRACKET:
			if !rulesFound {
				return nil, fmt.Errorf("syntax error near %s `%s`, unexpected `]`", t.Start, p.nearString(t.Start))
			}
			ruleset.RightBracketPos = t.Start
			lc, err := p.parseLineComments(t.Start)
			if err != nil {
				return nil, err
			}
			ruleset.Comments = append(ruleset.Comments, lc...)
			return ruleset, nil
		case token.RULES:
			ruleset.DeclPos = t.Start
			expetedEqual, ok := p.pop()
			if !ok {
				return nil, fmt.Errorf("syntax error near %s `%s`, unexpected EOF", t.Start, p.nearString(t.Start))
			}
			if expetedEqual.Type != token.EQUAL {
				return nil, fmt.Errorf("syntax error near %s `%s`, must equal after Rules", t.Start, p.nearString(t.Start))
			}
			expectedLeftBracket, lc, ok := p.popWithLineComment()
			if !ok {
				return nil, fmt.Errorf("syntax error near %s `%s`, unexpected EOF", t.Start, p.nearString(t.Start))
			}
			if expectedLeftBracket.Type != token.LEFT_BRACKET {
				return nil, fmt.Errorf("syntax error near %s `%s`, missing `[`", t.Start, p.nearString(t.Start))
			}
			ruleset.LeftBracketPos = expectedLeftBracket.Start
			ruleset.Comments = lc
			if lastCommentPos.IsValid() && lastCommentPos.Line+1 == t.Start.Line {
				ruleset.Description = storedComments
				storedComments = nil
				lastCommentPos = token.NoPos
			}
			rulesFound = true
		case token.COMMENT:
			if rulesFound {
				p.push(t)
				p.rulesetCommentGroups = nil
				rule, err := p.parseRule(true, false)
				if err != nil {
					return nil, err
				}
				ruleset.Rules = append(ruleset.Rules, rule)
				if len(p.rulesetCommentGroups) > 0 {
					ruleset.InnerComments = p.rulesetCommentGroups
				}
				continue
			}
			if !lastCommentPos.IsValid() || lastCommentPos.Line+1 == t.Start.Line {
				storedComments = append(storedComments, &ast.Comment{
					SharpPos: t.Start,
					Text:     t.Value,
				})
				lastCommentPos = t.Start
				continue
			}
			p.fileCommentGroups = append(p.fileCommentGroups, storedComments)
			storedComments = []*ast.Comment{
				{
					SharpPos: t.Start,
					Text:     t.Value,
				},
			}
			lastCommentPos = t.Start
		default:
			if !rulesFound {
				return nil, fmt.Errorf("syntax error near %s `%s`, unexpected `%s`", t.Start, p.nearString(t.Start), t.Type)
			}
			p.push(t)
			p.rulesetCommentGroups = nil
			rule, err := p.parseRule(true, false)
			if err != nil {
				return nil, err
			}
			ruleset.Rules = append(ruleset.Rules, rule)
			if len(p.rulesetCommentGroups) > 0 {
				ruleset.InnerComments = p.rulesetCommentGroups
			}
		}
	}
}

// ParseRule は単一のルールについての構文解析を行います。
// ParseRule parses a single rule.
func ParseRule(ruleStr string) (ast.RuleDecl, error) {
	p := &parser{
		input: ruleStr,
		lexer: newLexer("rule", ruleStr),
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	waiter := p.lexer.run(ctx)
	rule, err := p.parseRule(false, false)
	if err != nil {
		cancel()
		p.discardUntilToken(token.EOF)
		waiter()
		return nil, err
	}
	p.discardUntilToken(token.EOF)
	waiter()
	return rule, nil
}

func (p *parser) discardUntilToken(tokenType token.TokenType) token.Token {
	for {
		t, ok := p.pop()
		if !ok {
			return token.Token{Type: token.ILLEGAL}
		}
		if t.Type == tokenType {
			return t
		}
	}
}

func (p *parser) pop() (token.Token, bool) {
	if len(p.stack) == 0 {
		t, ok := <-p.lexer.TokenChan()
		return t, ok
	}
	t := p.stack[len(p.stack)-1]
	p.stack = p.stack[:len(p.stack)-1]
	return t, true
}

func (p *parser) push(t token.Token) {
	p.stack = append(p.stack, t)
}

// nearString は指定された位置のトークンから20文字分の文字列を返します。
// nearString returns a string of 20 characters from the specified position of the token.
func (p *parser) nearString(pos token.Pos) string {
	var str string
	if pos.Index-1 < 0 {
		str = p.input
	} else {
		str = p.input[pos.Index-1:]
	}
	if strings.ContainsRune(str, '\n') {
		str = str[:strings.IndexRune(str, '\n')]
	}
	if len(str) > 20 {
		str = str[:20] + "..."
	}
	return str
}

func (p *parser) parseRule(modeRuleset bool, nested bool) (ast.RuleDecl, error) {
	rule := &ast.Rule{}
	var ruleTypeFound, expressionFound bool
	var storedComments []*ast.Comment
	var lastCommentPos token.Pos
	for {
		t, ok := p.pop()
		if !ok {
			return nil, errors.New("unexpected EOF")
		}
		switch t.Type {
		case token.LEFT_PAREN:
			if nested {
				// 2 or more nested rules are not allowed
				return nil, fmt.Errorf("syntax error near %s: `%s`, deep nested rule is not allowed", t.Start, p.nearString(t.Start))
			}
			if len(storedComments) > 0 {
				if lastCommentPos.IsValid() && lastCommentPos.Line+1 == t.Start.Line {
					rule.Description = append(rule.Description, storedComments...)
				} else {
					p.rulesetCommentGroups = append(p.rulesetCommentGroups, storedComments)
				}
			}
			storedComments = nil
			lastCommentPos = token.NoPos
			p.push(t)
			r, err := p.parseCombinedRule(rule, modeRuleset)
			if err != nil {
				return nil, err
			}
			return r, nil
		case token.ILLEGAL:
			return nil, fmt.Errorf("syntax error near %s `%s`, %s", t.Start, p.nearString(t.Start), t.Value)
		case token.EOF:
			if len(storedComments) > 0 {
				p.rulesetCommentGroups = append(p.rulesetCommentGroups, storedComments)
			}
			if !ruleTypeFound {
				return nil, fmt.Errorf("syntax error near %s `%s`, RuleType is required: unexpexted EOF", t.Start, p.nearString(t.Start))
			}
			return rule, nil
		case token.COMMA, token.RIGHT_BRACKET:
			if len(storedComments) > 0 {
				p.rulesetCommentGroups = append(p.rulesetCommentGroups, storedComments)
			}
			if !ruleTypeFound {
				return nil, fmt.Errorf("syntax error near %s `%s`, RuleType is required: unexpected `,`", t.Start, p.nearString(t.Start))
			}
			if !modeRuleset {
				return nil, fmt.Errorf("syntax error near %s: `%s`, parse mode is single rule", t.Start, p.nearString(t.Start))
			}
			if t.Type == token.RIGHT_BRACKET {
				p.push(t)
			}
			return rule, nil
		case token.IDENT:
			if ruleTypeFound {
				return nil, fmt.Errorf("syntax error near %s: `%s`, RuleType is already defined", t.Start, p.nearString(t.Start))
			}
			rule.Type = &ast.Ident{
				NamePos: t.Start,
				Name:    t.Value,
			}
			ruleTypeFound = true
			if len(storedComments) > 0 {
				if lastCommentPos.IsValid() && lastCommentPos.Line+1 == t.Start.Line {
					rule.Description = append(rule.Description, storedComments...)
				} else {
					p.rulesetCommentGroups = append(p.rulesetCommentGroups, storedComments)
				}
			}
			storedComments = nil
			lastCommentPos = token.NoPos
			lineComments, err := p.parseLineComments(t.Start)
			if err != nil {
				return nil, err
			}
			rule.Type.Comments = lineComments
		case token.COMMENT:
			if !lastCommentPos.IsValid() || lastCommentPos.Line+1 == t.Start.Line {
				storedComments = append(storedComments, &ast.Comment{
					SharpPos: t.Start,
					Text:     t.Value,
				})
				lastCommentPos = t.Start
				continue
			}
			p.rulesetCommentGroups = append(p.rulesetCommentGroups, storedComments)
			storedComments = []*ast.Comment{
				{
					SharpPos: t.Start,
					Text:     t.Value,
				},
			}
			lastCommentPos = t.Start
		case token.RIGHT_PAREN:
			if nested {
				p.push(t)
				return rule, nil
			}
			return nil, fmt.Errorf("syntax error near %s: `%s`, unexpected `)`", t.Start, p.nearString(t.Start))
		default:
			if t.Type.IsParameterAcceptable() {
				if !ruleTypeFound {
					return nil, fmt.Errorf("syntax error near %s `%s`, RuleType is required: unexpected <Parameter>", t.Start, p.nearString(t.Start))
				}
				if expressionFound {
					return nil, fmt.Errorf("syntax error near %s `%s`, parameters must be before expression", t.Start, p.nearString(t.Start))
				}
				param, lineComments, err := p.parseParameter(t, rule.Type.Pos())
				if err != nil {
					return nil, err
				}
				if len(lineComments) > 0 {
					if len(rule.Type.Comments) > 0 {
						rule.Comments = rule.Type.Comments
					}
					rule.Comments = append(rule.Comments, lineComments...)
				}
				rule.Parameters = append(rule.Parameters, param)
				continue
			}
			if t.Type.IsExpressionStart() {
				if !ruleTypeFound {
					return nil, fmt.Errorf("syntax error near %s `%s`, RuleType is required: unexpected <Expression>", t.Start, p.nearString(t.Start))
				}
				expr, lc, err := p.parseExpression(t, rule.Pos(), modeRuleset)
				if err != nil {
					return nil, err
				}
				rule.Expression = expr
				rule.Comments = append(rule.Comments, lc...)
				expressionFound = true
				continue
			}
			return nil, fmt.Errorf("syntax error near %s: `%s`, unexpected token `%s`", t.Start, p.nearString(t.Start), t.Type)
		}
	}
}

func (p *parser) parseCombinedRule(firstRule *ast.Rule, modeRuleset bool) (ast.RuleDecl, error) {
	combined := &ast.CombinedRule{
		Description: firstRule.Description,
	}
	firstRule.Description = nil
	for {
		t, ok := p.pop()
		if !ok {
			return nil, errors.New("unexpected EOF")
		}
		switch t.Type {
		case token.LEFT_PAREN:
			if len(combined.Rules) == 0 {
				combined.FirstLParenPos = t.Start
			}
			rule, err := p.parseRule(modeRuleset, true)
			if err != nil {
				return nil, err
			}
			r, ok := rule.(*ast.Rule)
			if !ok {
				return nil, fmt.Errorf("syntax error near %s: `%s`, nested rule must be single rule", t.Start, p.nearString(t.Start))
			}
			combined.Rules = append(combined.Rules, r)
			n, ok := p.pop()
			if !ok {
				return nil, fmt.Errorf("syntax error near %s: `%s`, unexpected EOF", r.End(), p.nearString(r.End()))
			}
			if n.Type != token.RIGHT_PAREN {
				return nil, fmt.Errorf("syntax error near %s: `%s`, must close `)`", n.Start, p.nearString(n.Start))
			}
			combined.LastRParenPos = n.Start
		case token.AND, token.OR:
			if len(combined.Rules) == 0 {
				return nil, fmt.Errorf("syntax error near %s: `%s`, unexpected `%s`", t.Start, p.nearString(t.Start), t.Value)
			}
			if combined.Operator != "" && combined.Operator != t.Value {
				return nil, fmt.Errorf("syntax error near %s: `%s`, can not mixed `%s` and `%s`", t.Start, p.nearString(t.Start), combined.Operator, t.Value)
			}
			combined.Operator = t.Value
		case token.EOF:
			if len(combined.Rules) == 0 {
				return nil, fmt.Errorf("syntax error near %s: `%s`, unexpected EOF", t.Start, p.nearString(t.Start))
			}
			if len(combined.Rules) == 1 {
				combined.Rules[0].Description = combined.Description
				return combined.Rules[0], nil
			}
			return combined, nil
		case token.COMMA:
			if len(combined.Rules) == 0 {
				return nil, fmt.Errorf("syntax error near %s: `%s`, unexpected `,`", t.Start, p.nearString(t.Start))
			}
			if !modeRuleset {
				return nil, fmt.Errorf("syntax error near %s: `%s`, parse mode is single rule", t.Start, p.nearString(t.Start))
			}
			if len(combined.Rules) == 1 {
				combined.Rules[0].Description = combined.Description
				return combined.Rules[0], nil
			}
			return combined, nil
		case token.COMMENT:
			comment := &ast.Comment{
				SharpPos: t.Start,
				Text:     t.Value,
			}
			combined.Comments = append(combined.Comments, comment)
		default:
			return nil, fmt.Errorf("syntax error near %s: `%s`, unexpected token `%s`", t.Start, p.nearString(t.Start), t.Type)
		}
	}
}
func (p *parser) parseParameter(current token.Token, rulePos token.Pos) (ast.Parameter, ast.CommentGroup, error) {
	switch current.Type {
	case token.STRING:
		param := &ast.StringParameter{
			LeftQuotePos:  current.Start,
			Value:         strings.Trim(current.Value, `"`),
			RightQuotePos: current.Start.AddColumn(len(current.Value) - 1),
		}
		lineComments, err := p.parseLineComments(current.Start)
		if err != nil {
			return nil, nil, err
		}
		if rulePos.Line == current.Start.Line {
			return param, lineComments, nil
		}
		param.Comments = lineComments
		return param, nil, nil
	case token.NUMBER:
		next, ok := p.pop()
		if ok {
			switch next.Type {
			case token.DAYS, token.HOURS:
				if strings.ContainsRune(current.Value, '.') {
					return nil, nil, fmt.Errorf("syntax error near %s: `%s`, duration parameter can not be float", current.Start, p.nearString(current.Start))
				}
				param := &ast.DurationParameter{
					NumberPos: current.Start,
					UnitPos:   next.Start,
					Value:     current.Value + " " + next.Value,
					Number:    current.Value,
					Unit:      next.Value,
				}
				lineComments, err := p.parseLineComments(current.Start)
				if err != nil {
					return nil, nil, err
				}
				if rulePos.Line == current.Start.Line {
					return param, lineComments, nil
				}
				param.Comments = lineComments
				return param, nil, nil
			default:
				p.push(next)
			}
		}
		param := &ast.NumberParameter{
			NumberPos: current.Start,
			Value:     current.Value,
		}
		lineComments, err := p.parseLineComments(current.Start)
		if err != nil {
			return nil, nil, err
		}
		if rulePos.Line == current.Start.Line {
			return param, lineComments, nil
		}
		param.Comments = lineComments
		return param, nil, nil
	case token.TRUE, token.FALSE:
		param := &ast.BoolParameter{
			BoolPos: current.Start,
			Value:   current.Type == token.TRUE,
		}
		lineComments, err := p.parseLineComments(current.Start)
		if err != nil {
			return nil, nil, err
		}
		if rulePos.Line == current.Start.Line {
			return param, lineComments, nil
		}
		param.Comments = lineComments
		return param, nil, nil
	case token.NOW:
		param := &ast.DateParamter{
			NowPos: current.Start,
		}
		lineComments, err := p.parseLineComments(current.Start)
		if err != nil {
			return nil, nil, err
		}
		if rulePos.Line == current.Start.Line {
			return param, lineComments, nil
		}
		param.Comments = lineComments
		return param, nil, nil
	default:
		return nil, nil, fmt.Errorf("syntax error near %s: `%s`, no parameter", current.Start, p.nearString(current.Start))
	}
}

func (p *parser) popWithLineComment() (token.Token, ast.CommentGroup, bool) {
	t, ok := p.pop()
	if !ok {
		return token.Token{}, nil, false
	}
	lc, err := p.parseLineComments(t.Start)
	if err != nil {
		return t, nil, true
	}
	return t, lc, true
}

func (p *parser) parseExpression(current token.Token, rulePos token.Pos, modeRuleset bool) (ast.Expression, ast.CommentGroup, error) {
	var lineComments ast.CommentGroup
	switch current.Type {
	case token.GREATER_EQUAL, token.GREATER_THAN, token.LESS_EQUAL, token.LESS_THAN, token.EQUAL:
		expr := &ast.ComparisonExpression{
			ExprPos:  current.Start,
			Operator: current.Value,
		}
		t, lc, ok := p.popWithLineComment()
		if !ok {
			return nil, nil, fmt.Errorf("syntax error near %s: `%s`, unexpected EOF", current.Start, p.nearString(current.Start))
		}
		if rulePos.Line == current.Start.Line {
			lineComments = lc
		} else {
			expr.Comments = lc
		}
		switch {
		case t.Type.IsParameterAcceptable():
			param, lc, err := p.parseParameter(t, rulePos)
			if err != nil {
				return nil, nil, err
			}
			lineComments = lc
			expr.Right = param
		case t.Type == token.LEFT_PAREN:
			// parse date expression as `(now() - 1 days)`
			param := &ast.DateParamter{
				LeftParenPos: t.Start.Ptr(),
			}
			t, lc, ok := p.popWithLineComment()
			if !ok {
				return nil, nil, fmt.Errorf("syntax error near %s: `%s`, unexpected EOF", current.Start, p.nearString(current.Start))
			}
			if t.Type != token.NOW {
				return nil, nil, fmt.Errorf("syntax error near %s: `%s`, unexpected token `%s`", t.Start, p.nearString(t.Start), t.Type)
			}
			if rulePos.Line == t.Start.Line {
				lineComments = append(lineComments, lc...)
			} else {
				param.Comments = append(param.Comments, lc...)
			}
			param.NowPos = t.Start
			t, lc, ok = p.popWithLineComment()
			if !ok {
				return nil, nil, fmt.Errorf("syntax error near %s: `%s`, unexpected EOF", current.Start, p.nearString(current.Start))
			}
			if t.Type != token.MINUS {
				return nil, nil, fmt.Errorf("syntax error near %s: `%s`, unexpected token `%s`", t.Start, p.nearString(t.Start), t.Type)
			}
			if rulePos.Line == t.Start.Line {
				lineComments = append(lineComments, lc...)
			} else {
				param.Comments = append(param.Comments, lc...)
			}
			param.MinusPos = t.Start.Ptr()
			t, ok = p.pop()
			if !ok {
				return nil, nil, fmt.Errorf("syntax error near %s: `%s`, unexpected EOF", current.Start, p.nearString(current.Start))
			}
			dp, lc, err := p.parseParameter(t, rulePos)
			if err != nil {
				return nil, nil, err
			}
			durationParam, ok := dp.(*ast.DurationParameter)
			if !ok {
				return nil, nil, fmt.Errorf("syntax error near %s: `%s`, expected duration parameter", t.Start, p.nearString(t.Start))
			}
			param.Duration = durationParam
			lineComments = append(lineComments, lc...)
			t, lc, ok = p.popWithLineComment()
			if !ok {
				return nil, nil, fmt.Errorf("syntax error near %s: `%s`, unexpected EOF", current.Start, p.nearString(current.Start))
			}
			if t.Type != token.RIGHT_PAREN {
				return nil, nil, fmt.Errorf("syntax error near %s: `%s`, unexpected token `%s`", t.Start, p.nearString(t.Start), t.Type)
			}
			if rulePos.Line == t.Start.Line {
				lineComments = append(lineComments, lc...)
			} else {
				param.Comments = append(param.Comments, lc...)
			}
			param.RightParenPos = t.Start.Ptr()
			expr.Right = param
			lc, err = p.parseLineComments(t.Start)
			if err != nil {
				return nil, nil, err
			}
			if rulePos.Line == current.Start.Line {
				lineComments = append(lineComments, lc...)
			} else {
				expr.Comments = append(expr.Comments, lc...)
			}
		default:
			return nil, nil, fmt.Errorf("syntax error near %s: `%s`, unexpected token `%s`", t.Start, p.nearString(t.Start), t.Type)
		}
		return expr, lineComments, nil
	case token.BETWEEN:
		expr := &ast.BetweenExpression{
			ExprPos: current.Start,
		}
		left, ok := p.pop()
		if !ok {
			return nil, nil, fmt.Errorf("syntax error near %s: `%s`, unexpected EOF", current.Start, p.nearString(current.Start))
		}
		if !left.Type.IsParameterAcceptable() {
			return nil, nil, fmt.Errorf("syntax error near %s: `%s`, unexpected token `%s`", left.Start, p.nearString(left.Start), left.Type)
		}
		leftParam, lc, err := p.parseParameter(left, rulePos)
		if err != nil {
			return nil, nil, err
		}
		expr.Left = leftParam
		if rulePos.Line == left.Start.Line {
			lineComments = append(lineComments, lc...)
		} else {
			expr.Comments = append(expr.Comments, lc...)
		}
		and, lc, ok := p.popWithLineComment()
		if !ok {
			return nil, nil, fmt.Errorf("syntax error near %s: `%s`, unexpected EOF", current.Start, p.nearString(current.Start))
		}
		if and.Type != token.AND {
			return nil, nil, fmt.Errorf("syntax error near %s: `%s`, expected `and` but got `%s`", and.Start, p.nearString(and.Start), and.Value)
		}
		if rulePos.Line == and.Start.Line {
			lineComments = append(lineComments, lc...)
		} else {
			expr.Comments = append(expr.Comments, lc...)
		}
		right, ok := p.pop()
		if !ok {
			return nil, nil, fmt.Errorf("syntax error near %s: `%s`, unexpected EOF", current.Start, p.nearString(current.Start))
		}
		if !right.Type.IsParameterAcceptable() {
			return nil, nil, fmt.Errorf("syntax error near %s: `%s`, unexpected token `%s`", right.Start, p.nearString(right.Start), right.Type)
		}
		rightParam, lc, err := p.parseParameter(right, rulePos)
		if err != nil {
			return nil, nil, err
		}
		expr.Right = rightParam
		if rulePos.Line == right.Start.Line {
			lineComments = append(lineComments, lc...)
		} else {
			expr.Comments = append(expr.Comments, lc...)
		}
		return expr, lineComments, err
	case token.IN:
		expr := &ast.InExpression{
			ExprPos: current.Start,
		}
		left, lc, ok := p.popWithLineComment()
		if !ok {
			return nil, nil, fmt.Errorf("syntax error near %s: `%s`, unexpected EOF", current.Start, p.nearString(current.Start))
		}
		if left.Type != token.LEFT_BRACKET {
			return nil, nil, fmt.Errorf("syntax error near %s: `%s`, expected `[` but got `%s`", left.Start, p.nearString(left.Start), left.Value)
		}
		expr.LeftBracketPos = left.Start
		if rulePos.Line == left.Start.Line {
			lineComments = append(lineComments, lc...)
		} else {
			expr.Comments = append(expr.Comments, lc...)
		}
		for {
			t, ok := p.pop()
			if !ok {
				return nil, nil, fmt.Errorf("syntax error near %s: `%s`, unexpected EOF", current.Start, p.nearString(current.Start))
			}
			if !t.Type.IsParameterAcceptable() {
				return nil, nil, fmt.Errorf("syntax error near %s: `%s`, unexpected token `%s`", t.Start, p.nearString(t.Start), t.Type)
			}
			param, lc, err := p.parseParameter(t, rulePos)
			if err != nil {
				return nil, nil, err
			}
			if rulePos.Line == t.Start.Line {
				lineComments = append(lineComments, lc...)
			} else {
				expr.Comments = append(expr.Comments, lc...)
			}
			expr.Values = append(expr.Values, param)
			t, lc, ok = p.popWithLineComment()
			if !ok {
				return nil, nil, fmt.Errorf("syntax error near %s: `%s`, unexpected EOF", current.Start, p.nearString(current.Start))
			}
			if rulePos.Line == t.Start.Line {
				lineComments = append(lineComments, lc...)
			} else {
				expr.Comments = append(expr.Comments, lc...)
			}
			if t.Type == token.RIGHT_BRACKET {
				expr.RightBracketPos = t.Start
				break
			}
			if t.Type != token.COMMA {
				return nil, nil, fmt.Errorf("syntax error near %s: `%s`, expected `,` but got `%s`", t.Start, p.nearString(t.Start), t.Value)
			}
		}
		withThresholdExpr, lc, err := p.parseWithThreshold(expr, rulePos, modeRuleset)
		if err != nil {
			return nil, nil, err
		}
		lineComments = append(lineComments, lc...)
		return withThresholdExpr, lineComments, err
	case token.MATCHES:
		expr := &ast.MatchesExpression{
			ExprPos: current.Start,
		}
		regexpValue, lc, ok := p.popWithLineComment()
		if !ok {
			return nil, nil, fmt.Errorf("syntax error near %s: `%s`, unexpected EOF", current.Start, p.nearString(current.Start))
		}
		if regexpValue.Type != token.STRING {
			return nil, nil, fmt.Errorf("syntax error near %s: `%s`, expected string but got `%s`", regexpValue.Start, p.nearString(regexpValue.Start), regexpValue.Value)
		}
		if rulePos.Line == regexpValue.Start.Line {
			lineComments = append(lineComments, lc...)
		} else {
			expr.Comments = append(expr.Comments, lc...)
		}
		expr.RegexpPos = regexpValue.Start
		expr.Value = strings.Trim(regexpValue.Value, `"`)
		withThresholdExpr, lc, err := p.parseWithThreshold(expr, rulePos, modeRuleset)
		if err != nil {
			return nil, nil, err
		}
		lineComments = append(lineComments, lc...)
		return withThresholdExpr, lineComments, err
	default:
		return nil, nil, fmt.Errorf("syntax error near %s: `%s`, unexpected token `%s`", current.Start, p.nearString(current.Start), current.Type)
	}
}

func (p *parser) parseWithThreshold(expr ast.ThresholdTarget, rulePos token.Pos, modeRuleset bool) (ast.Expression, ast.CommentGroup, error) {
	with, ok := p.pop()
	if !ok {
		return expr, nil, nil
	}
	if with.Type != token.WITH {
		p.push(with)
		return expr, nil, nil
	}
	thresholdKeywords, lineComments, ok := p.popWithLineComment()
	if !ok {
		return nil, nil, fmt.Errorf("syntax error near %s: `%s`, unexpected EOF", with.Start, p.nearString(with.Start))
	}
	if thresholdKeywords.Type != token.THRESHOLD {
		return nil, nil, fmt.Errorf("syntax error near %s: `%s`, expected `threshold` but got `%s`", thresholdKeywords.Start, p.nearString(thresholdKeywords.Start), thresholdKeywords.Value)
	}
	thresholdValue, ok := p.pop()
	if !ok {
		return nil, nil, fmt.Errorf("syntax error near %s: `%s`, unexpected EOF", thresholdKeywords.Start, p.nearString(thresholdKeywords.Start))
	}
	if !thresholdValue.Type.IsExpressionStart() {
		return nil, nil, fmt.Errorf("syntax error near %s: `%s`, unexpected token `%s`", thresholdValue.Start, p.nearString(thresholdValue.Start), thresholdValue.Type)
	}
	tExpr, lc, err := p.parseExpression(thresholdValue, rulePos, modeRuleset)
	if err != nil {
		return nil, nil, err
	}
	threshold, ok := tExpr.(ast.ThresholdExpression)
	if !ok {
		return nil, nil, fmt.Errorf("syntax error near %s: `%s`, expected threshold expression but got `%s`", thresholdValue.Start, p.nearString(thresholdValue.Start), thresholdValue.Type)
	}
	lineComments = append(lineComments, lc...)
	withThresholdExpr := &ast.WithThresholdExpression{
		ExprPos:   with.Start,
		Target:    expr,
		Threshold: threshold,
	}
	if rulePos.Line == with.Start.Line {
		return withThresholdExpr, lineComments, nil
	}
	withThresholdExpr.Comments = append(withThresholdExpr.Comments, lineComments...)
	return withThresholdExpr, nil, nil
}

// parseLineComments は与えられた位置を元に、以降のコメントを解析します。
// コメントがない場合は nil を返します。
// parseLineComments is parse comments after given position.
// If there is no comment, return nil.
func (p *parser) parseLineComments(pos token.Pos) (ast.CommentGroup, error) {
	comments := ast.CommentGroup{}
	var lastCommentPos token.Pos
	for {
		comment, ok := p.pop()
		if !ok {
			break
		}
		if comment.Type != token.COMMENT {
			p.push(comment)
			break
		}
		if comment.Start.Line != pos.Line {
			if !lastCommentPos.IsValid() {
				p.push(comment)
				break
			}
			if lastCommentPos.Line+1 != comment.Start.Line {
				p.push(comment)
				break
			}
			if lastCommentPos.Column != comment.Start.Column {
				p.push(comment)
				break
			}
		}
		comments = append(comments, &ast.Comment{
			SharpPos: comment.Start,
			Text:     comment.Value,
		})
		lastCommentPos = comment.Start
	}
	if len(comments) == 0 {
		return nil, nil
	}
	return comments, nil
}
