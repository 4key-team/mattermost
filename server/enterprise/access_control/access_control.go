// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

//go:build sourceavailable && !enterprise

package accesscontrol

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/shared/mlog"
	"github.com/mattermost/mattermost/server/public/shared/request"
	"github.com/mattermost/mattermost/server/v8/channels/app"
	channeljobs "github.com/mattermost/mattermost/server/v8/channels/jobs"
	"github.com/mattermost/mattermost/server/v8/channels/store"
	"github.com/mattermost/mattermost/server/v8/einterfaces"
	ejobs "github.com/mattermost/mattermost/server/v8/einterfaces/jobs"
)

func init() {
	app.RegisterAccessControlServiceInterface(func(a *app.App) einterfaces.AccessControlServiceInterface {
		return &service{app: a}
	})
	app.RegisterJobsAccessControlSyncJobInterface(func(s *app.Server) ejobs.AccessControlSyncJobInterface {
		return &syncJobBuilder{server: s}
	})
}

type service struct {
	app *app.App
}

func (s *service) Init(_ request.CTX) *model.AppError {
	if err := s.refreshAttributes(); err != nil {
		return newInternalAppError("Init", err)
	}

	return nil
}

func (s *service) SavePolicy(rctx request.CTX, policy *model.AccessControlPolicy) (*model.AccessControlPolicy, *model.AppError) {
	for _, rule := range policy.Rules {
		if errs, appErr := s.CheckExpression(rctx, rule.Expression); appErr != nil {
			return nil, appErr
		} else if len(errs) > 0 {
			return nil, model.NewAppError("SavePolicy", "app.pap.save_policy.app_error", nil, errs[0].Message, http.StatusBadRequest)
		}
	}

	saved, err := s.app.Srv().Store().AccessControlPolicy().Save(rctx, policy)
	if err != nil {
		return nil, newStoreAppError("SavePolicy", "app.pap.save_policy.app_error", err)
	}

	return saved, nil
}

func (s *service) GetPolicy(rctx request.CTX, id string) (*model.AccessControlPolicy, *model.AppError) {
	policy, err := s.app.Srv().Store().AccessControlPolicy().Get(rctx, id)
	if err != nil {
		return nil, mapStoreError("GetPolicy", "app.pap.get_policy.app_error", err)
	}

	return policy, nil
}

func (s *service) DeletePolicy(rctx request.CTX, id string) *model.AppError {
	if err := s.app.Srv().Store().AccessControlPolicy().Delete(rctx, id); err != nil {
		return mapStoreError("DeletePolicy", "app.pap.delete_policy.app_error", err)
	}

	return nil
}

func (s *service) CheckExpression(_ request.CTX, expression string) ([]model.CELExpressionError, *model.AppError) {
	if _, err := parseExpression(expression); err != nil {
		return []model.CELExpressionError{{
			Line:    1,
			Column:  1,
			Message: err.Error(),
		}}, nil
	}

	return nil, nil
}

func (s *service) ExpressionToVisualAST(_ request.CTX, expression string) (*model.VisualExpression, *model.AppError) {
	node, err := parseExpression(expression)
	if err != nil {
		return nil, model.NewAppError("ExpressionToVisualAST", "app.pap.expression_to_visual_ast.app_error", nil, err.Error(), http.StatusBadRequest)
	}

	conditions, ok := flattenAndConditions(node)
	if !ok {
		return nil, model.NewAppError("ExpressionToVisualAST", "app.pap.expression_to_visual_ast.app_error", nil, "expression cannot be represented in visual editor", http.StatusBadRequest)
	}

	visualConditions := make([]model.Condition, 0, len(conditions))
	for _, condition := range conditions {
		visualConditions = append(visualConditions, model.Condition{
			Attribute: condition.Attribute,
			Operator:  condition.Operator,
			Value:     condition.VisualValue(),
			ValueType: condition.ValueType,
		})
	}

	return &model.VisualExpression{Conditions: visualConditions}, nil
}

func (s *service) NormalizePolicy(_ request.CTX, policy *model.AccessControlPolicy) (*model.AccessControlPolicy, *model.AppError) {
	return policy, nil
}

func (s *service) GetPolicyRuleAttributes(rctx request.CTX, policyID string, action string) (map[string][]string, *model.AppError) {
	rules, appErr := s.getEffectiveRules(rctx, policyID, action)
	if appErr != nil {
		return nil, appErr
	}

	attributes := make(map[string][]string)
	for _, rule := range rules {
		node, err := parseExpression(rule.Expression)
		if err != nil {
			continue
		}

		for _, condition := range collectConditions(node) {
			name := condition.AttributeName()
			if name == "" {
				continue
			}

			existing := attributes[name]
			for _, value := range condition.Values {
				if !slices.Contains(existing, value) {
					existing = append(existing, value)
				}
			}
			attributes[name] = existing
		}
	}

	return attributes, nil
}

func (s *service) QueryUsersForExpression(rctx request.CTX, expression string, opts model.SubjectSearchOptions) ([]*model.User, int64, *model.AppError) {
	if err := s.refreshAttributes(); err != nil {
		return nil, 0, newInternalAppError("QueryUsersForExpression", err)
	}

	node, err := parseExpression(expression)
	if err != nil {
		return nil, 0, model.NewAppError("QueryUsersForExpression", "app.pap.query_users_for_expression.app_error", nil, err.Error(), http.StatusBadRequest)
	}

	query, args, err := compileSQL(node)
	if err != nil {
		return nil, 0, model.NewAppError("QueryUsersForExpression", "app.pap.query_users_for_expression.app_error", nil, err.Error(), http.StatusBadRequest)
	}

	opts.Query = query
	opts.Args = args
	users, total, storeErr := s.app.Srv().Store().Attributes().SearchUsers(rctx, opts)
	if storeErr != nil {
		return nil, 0, newStoreAppError("QueryUsersForExpression", "app.pap.query_users_for_expression.app_error", storeErr)
	}

	return users, total, nil
}

func (s *service) QueryUsersForResource(rctx request.CTX, resourceID, action string, opts model.SubjectSearchOptions) ([]*model.User, int64, *model.AppError) {
	expression, appErr := s.buildResourceExpression(rctx, resourceID, action)
	if appErr != nil {
		return nil, 0, appErr
	}
	if expression == "" {
		return []*model.User{}, 0, nil
	}

	return s.QueryUsersForExpression(rctx, expression, opts)
}

func (s *service) GetChannelMembersToRemove(rctx request.CTX, channelID string) ([]*model.ChannelMember, *model.AppError) {
	if err := s.refreshAttributes(); err != nil {
		return nil, newInternalAppError("GetChannelMembersToRemove", err)
	}

	expression, appErr := s.buildResourceExpression(rctx, channelID, "*")
	if appErr != nil {
		return nil, appErr
	}
	if expression == "" {
		return []*model.ChannelMember{}, nil
	}

	node, err := parseExpression(expression)
	if err != nil {
		return nil, model.NewAppError("GetChannelMembersToRemove", "app.pap.get_channel_members_to_remove.app_error", nil, err.Error(), http.StatusBadRequest)
	}

	query, args, err := compileSQL(node)
	if err != nil {
		return nil, model.NewAppError("GetChannelMembersToRemove", "app.pap.get_channel_members_to_remove.app_error", nil, err.Error(), http.StatusBadRequest)
	}

	members, storeErr := s.app.Srv().Store().Attributes().GetChannelMembersToRemove(rctx, channelID, model.SubjectSearchOptions{
		Query: query,
		Args:  args,
	})
	if storeErr != nil {
		return nil, newStoreAppError("GetChannelMembersToRemove", "app.pap.get_channel_members_to_remove.app_error", storeErr)
	}

	return members, nil
}

func (s *service) AccessEvaluation(rctx request.CTX, accessRequest model.AccessRequest) (model.AccessDecision, *model.AppError) {
	if err := s.refreshAttributes(); err != nil {
		return model.AccessDecision{}, newInternalAppError("AccessEvaluation", err)
	}

	expression, appErr := s.buildResourceExpression(rctx, accessRequest.Resource.ID, accessRequest.Action)
	if appErr != nil {
		return model.AccessDecision{}, appErr
	}
	if expression == "" {
		return model.AccessDecision{Decision: false}, nil
	}

	node, err := parseExpression(expression)
	if err != nil {
		return model.AccessDecision{}, model.NewAppError("AccessEvaluation", "app.pdp.access_evaluation.app_error", nil, err.Error(), http.StatusBadRequest)
	}

	return model.AccessDecision{Decision: node.Evaluate(accessRequest.Subject.Attributes)}, nil
}

func (s *service) buildResourceExpression(rctx request.CTX, resourceID, action string) (string, *model.AppError) {
	rules, appErr := s.getEffectiveRules(rctx, resourceID, action)
	if appErr != nil {
		return "", appErr
	}

	expressions := make([]string, 0, len(rules))
	for _, rule := range rules {
		if strings.TrimSpace(rule.Expression) != "" {
			expressions = append(expressions, fmt.Sprintf("(%s)", rule.Expression))
		}
	}

	return strings.Join(expressions, " || "), nil
}

func (s *service) getEffectiveRules(rctx request.CTX, policyID, action string) ([]model.AccessControlPolicyRule, *model.AppError) {
	return s.getEffectiveRulesRecursive(rctx, policyID, action, map[string]bool{})
}

func (s *service) getEffectiveRulesRecursive(rctx request.CTX, policyID, action string, seen map[string]bool) ([]model.AccessControlPolicyRule, *model.AppError) {
	if seen[policyID] {
		return nil, model.NewAppError("getEffectiveRulesRecursive", "app.pap.get_policy.app_error", nil, "cyclic access control policy import", http.StatusBadRequest)
	}
	seen[policyID] = true

	policy, appErr := s.GetPolicy(rctx, policyID)
	if appErr != nil {
		if appErr.StatusCode == http.StatusNotFound {
			return nil, nil
		}
		return nil, appErr
	}
	if policy == nil || !policy.Active {
		return nil, nil
	}

	var rules []model.AccessControlPolicyRule
	for _, importedID := range policy.Imports {
		importedRules, importedErr := s.getEffectiveRulesRecursive(rctx, importedID, action, seen)
		if importedErr != nil {
			return nil, importedErr
		}
		rules = append(rules, importedRules...)
	}

	for _, rule := range policy.Rules {
		if actionMatches(rule.Actions, action) {
			rules = append(rules, rule)
		}
	}

	return rules, nil
}

func actionMatches(actions []string, action string) bool {
	if len(actions) == 0 {
		return true
	}
	for _, candidate := range actions {
		if candidate == "*" || candidate == action {
			return true
		}
	}
	return false
}

func mapStoreError(where, id string, err error) *model.AppError {
	var notFound *store.ErrNotFound
	if errors.As(err, &notFound) {
		return model.NewAppError(where, id, nil, err.Error(), http.StatusNotFound)
	}

	return newStoreAppError(where, id, err)
}

func newStoreAppError(where, id string, err error) *model.AppError {
	return model.NewAppError(where, id, nil, err.Error(), http.StatusInternalServerError)
}

func newInternalAppError(where string, err error) *model.AppError {
	return model.NewAppError(where, "app.pap.internal.app_error", nil, err.Error(), http.StatusInternalServerError)
}

func (s *service) refreshAttributes() error {
	return s.app.Srv().Store().Attributes().RefreshAttributes()
}

type syncJobBuilder struct {
	server *app.Server
}

func (b *syncJobBuilder) MakeWorker() model.Worker {
	const workerName = "SourceAvailableAccessControlSync"

	execute := func(logger mlog.LoggerIFace, job *model.Job) error {
		policyID := job.Data["policy_id"]
		if policyID == "" {
			return nil
		}

		appInstance := app.New(app.ServerConnector(b.server.Channels()))
		if appErr := appInstance.SyncAccessControlledChannelMembers(request.EmptyContext(logger), policyID); appErr != nil {
			return appErr
		}

		return nil
	}

	isEnabled := func(*model.Config) bool {
		return true
	}

	return channeljobs.NewSimpleWorker(workerName, b.server.Jobs, execute, isEnabled)
}

func (b *syncJobBuilder) MakeScheduler() ejobs.Scheduler {
	return nil
}

type exprNode interface {
	Evaluate(attrs map[string]any) bool
}

type binaryNode struct {
	Operator string
	Left     exprNode
	Right    exprNode
}

func (n binaryNode) Evaluate(attrs map[string]any) bool {
	switch n.Operator {
	case "&&":
		return n.Left.Evaluate(attrs) && n.Right.Evaluate(attrs)
	case "||":
		return n.Left.Evaluate(attrs) || n.Right.Evaluate(attrs)
	default:
		return false
	}
}

type conditionNode struct {
	Attribute string
	Operator  string
	Values    []string
	ValueType model.ValueType
}

func (n conditionNode) AttributeName() string {
	return strings.TrimPrefix(n.Attribute, "user.attributes.")
}

func (n conditionNode) VisualValue() any {
	if n.Operator == "in" && len(n.Values) > 1 {
		return append([]string(nil), n.Values...)
	}
	if len(n.Values) == 1 {
		return n.Values[0]
	}
	return append([]string(nil), n.Values...)
}

func (n conditionNode) Evaluate(attrs map[string]any) bool {
	attr := attrs[n.AttributeName()]

	switch n.Operator {
	case "==":
		return compareScalar(attr, firstValue(n.Values))
	case "!=":
		return attr != nil && !compareScalar(attr, firstValue(n.Values))
	case "in":
		if len(n.Values) == 1 && isStringList(attr) {
			return listContains(asStringSlice(attr), n.Values[0])
		}

		actual := asString(attr)
		if actual == "" {
			return false
		}
		for _, value := range n.Values {
			if actual == value {
				return true
			}
		}
		return false
	case "contains":
		return strings.Contains(asString(attr), firstValue(n.Values))
	case "startsWith":
		return strings.HasPrefix(asString(attr), firstValue(n.Values))
	case "endsWith":
		return strings.HasSuffix(asString(attr), firstValue(n.Values))
	case "array_in":
		actual := asStringSlice(attr)
		if len(actual) == 0 {
			return false
		}
		for _, value := range n.Values {
			if !listContains(actual, value) {
				return false
			}
		}
		return true
	default:
		return false
	}
}

func compareScalar(attr any, expected string) bool {
	switch value := attr.(type) {
	case string:
		return value == expected
	case []string:
		return len(value) == 1 && value[0] == expected
	case []any:
		return len(value) == 1 && asString(value[0]) == expected
	default:
		return false
	}
}

func asString(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case fmt.Stringer:
		return v.String()
	default:
		raw, _ := json.Marshal(v)
		return strings.Trim(string(raw), "\"")
	}
}

func asStringSlice(value any) []string {
	switch v := value.(type) {
	case []string:
		return append([]string(nil), v...)
	case []any:
		result := make([]string, 0, len(v))
		for _, item := range v {
			result = append(result, asString(item))
		}
		return result
	default:
		return nil
	}
}

func isStringList(value any) bool {
	switch value.(type) {
	case []string, []any:
		return true
	default:
		return false
	}
}

func firstValue(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

func listContains(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}

func collectConditions(node exprNode) []conditionNode {
	switch n := node.(type) {
	case conditionNode:
		return []conditionNode{n}
	case binaryNode:
		left := collectConditions(n.Left)
		right := collectConditions(n.Right)
		return append(left, right...)
	default:
		return nil
	}
}

func flattenAndConditions(node exprNode) ([]conditionNode, bool) {
	switch n := node.(type) {
	case conditionNode:
		return []conditionNode{n}, true
	case binaryNode:
		if n.Operator != "&&" {
			return nil, false
		}
		left, ok := flattenAndConditions(n.Left)
		if !ok {
			return nil, false
		}
		right, ok := flattenAndConditions(n.Right)
		if !ok {
			return nil, false
		}
		return append(left, right...), true
	default:
		return nil, false
	}
}

func compileSQL(node exprNode) (string, []any, error) {
	builder := &sqlBuilder{}
	sql, err := builder.compile(node)
	if err != nil {
		return "", nil, err
	}
	return sql, builder.args, nil
}

type sqlBuilder struct {
	args []any
}

func (b *sqlBuilder) compile(node exprNode) (string, error) {
	switch n := node.(type) {
	case binaryNode:
		left, err := b.compile(n.Left)
		if err != nil {
			return "", err
		}
		right, err := b.compile(n.Right)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("(%s %s %s)", left, n.Operator, right), nil
	case conditionNode:
		return b.compileCondition(n)
	default:
		return "", fmt.Errorf("unsupported expression node %T", node)
	}
}

func (b *sqlBuilder) compileCondition(node conditionNode) (string, error) {
	attr := fmt.Sprintf("Attributes -> '%s'", node.AttributeName())
	attrText := fmt.Sprintf("Attributes ->> '%s'", node.AttributeName())

	switch node.Operator {
	case "==":
		return fmt.Sprintf("(%s = %s)", attrText, b.bind(firstValue(node.Values))), nil
	case "!=":
		return fmt.Sprintf("(Attributes ? '%s' AND %s <> %s)", node.AttributeName(), attrText, b.bind(firstValue(node.Values))), nil
	case "in":
		if len(node.Values) == 1 {
			return fmt.Sprintf("(%s = %s)", attrText, b.bind(node.Values[0])), nil
		}
		placeholders := make([]string, 0, len(node.Values))
		for _, value := range node.Values {
			placeholders = append(placeholders, b.bind(value))
		}
		return fmt.Sprintf("(%s IN (%s))", attrText, strings.Join(placeholders, ", ")), nil
	case "array_in":
		payload, _ := json.Marshal(node.Values)
		return fmt.Sprintf("(COALESCE(%s, '[]'::jsonb) @> %s::jsonb)", attr, b.bind(string(payload))), nil
	case "contains":
		return fmt.Sprintf("(%s LIKE %s)", attrText, b.bind("%"+firstValue(node.Values)+"%")), nil
	case "startsWith":
		return fmt.Sprintf("(%s LIKE %s)", attrText, b.bind(firstValue(node.Values)+"%")), nil
	case "endsWith":
		return fmt.Sprintf("(%s LIKE %s)", attrText, b.bind("%"+firstValue(node.Values))), nil
	default:
		return "", fmt.Errorf("unsupported operator %q", node.Operator)
	}
}

func (b *sqlBuilder) bind(value any) string {
	b.args = append(b.args, value)
	return fmt.Sprintf("$%d", len(b.args))
}

type parser struct {
	input string
	pos   int
}

func parseExpression(expression string) (exprNode, error) {
	p := &parser{input: strings.TrimSpace(expression)}
	if p.input == "" {
		return nil, errors.New("expression is empty")
	}

	node, err := p.parseOr()
	if err != nil {
		return nil, err
	}
	p.skipSpaces()
	if !p.eof() {
		return nil, fmt.Errorf("unexpected token near %q", p.input[p.pos:])
	}
	return node, nil
}

func (p *parser) parseOr() (exprNode, error) {
	left, err := p.parseAnd()
	if err != nil {
		return nil, err
	}

	for {
		p.skipSpaces()
		if !p.consume("||") {
			return left, nil
		}

		right, err := p.parseAnd()
		if err != nil {
			return nil, err
		}
		left = binaryNode{Operator: "||", Left: left, Right: right}
	}
}

func (p *parser) parseAnd() (exprNode, error) {
	left, err := p.parsePrimary()
	if err != nil {
		return nil, err
	}

	for {
		p.skipSpaces()
		if !p.consume("&&") {
			return left, nil
		}

		right, err := p.parsePrimary()
		if err != nil {
			return nil, err
		}
		left = binaryNode{Operator: "&&", Left: left, Right: right}
	}
}

func (p *parser) parsePrimary() (exprNode, error) {
	p.skipSpaces()
	if p.consume("(") {
		node, err := p.parseOr()
		if err != nil {
			return nil, err
		}
		p.skipSpaces()
		if !p.consume(")") {
			return nil, errors.New("missing closing parenthesis")
		}
		return node, nil
	}

	return p.parseCondition()
}

func (p *parser) parseCondition() (exprNode, error) {
	p.skipSpaces()

	if p.peek() == '[' {
		values, err := p.parseList()
		if err != nil {
			return nil, err
		}
		p.skipSpaces()
		if !p.consumeWord("in") {
			return nil, errors.New("expected 'in' after list")
		}
		attr, err := p.parseAttribute()
		if err != nil {
			return nil, err
		}
		return conditionNode{Attribute: attr, Operator: "array_in", Values: values, ValueType: model.LiteralValue}, nil
	}

	if quote := p.peek(); quote == '\'' || quote == '"' {
		value, err := p.parseString()
		if err != nil {
			return nil, err
		}
		p.skipSpaces()
		if !p.consumeWord("in") {
			return nil, errors.New("expected 'in' after value")
		}
		attr, err := p.parseAttribute()
		if err != nil {
			return nil, err
		}
		return conditionNode{Attribute: attr, Operator: "in", Values: []string{value}, ValueType: model.LiteralValue}, nil
	}

	attr, err := p.parseAttribute()
	if err != nil {
		return nil, err
	}

	p.skipSpaces()
	if p.consume(".") {
		method, err := p.parseIdentifier()
		if err != nil {
			return nil, err
		}
		if !slices.Contains([]string{"contains", "startsWith", "endsWith"}, method) {
			return nil, fmt.Errorf("unsupported method %q", method)
		}
		p.skipSpaces()
		if !p.consume("(") {
			return nil, errors.New("expected '(' after method")
		}
		value, err := p.parseString()
		if err != nil {
			return nil, err
		}
		p.skipSpaces()
		if !p.consume(")") {
			return nil, errors.New("expected ')' after method argument")
		}
		return conditionNode{Attribute: attr, Operator: method, Values: []string{value}, ValueType: model.LiteralValue}, nil
	}

	switch {
	case p.consume("=="):
		value, err := p.parseString()
		if err != nil {
			return nil, err
		}
		return conditionNode{Attribute: attr, Operator: "==", Values: []string{value}, ValueType: model.LiteralValue}, nil
	case p.consume("!="):
		value, err := p.parseString()
		if err != nil {
			return nil, err
		}
		return conditionNode{Attribute: attr, Operator: "!=", Values: []string{value}, ValueType: model.LiteralValue}, nil
	case p.consumeWord("in"):
		values, err := p.parseList()
		if err != nil {
			return nil, err
		}
		return conditionNode{Attribute: attr, Operator: "in", Values: values, ValueType: model.LiteralValue}, nil
	default:
		return nil, errors.New("expected comparison operator")
	}
}

func (p *parser) parseAttribute() (string, error) {
	p.skipSpaces()
	first, err := p.parseIdentifier()
	if err != nil {
		return "", err
	}
	second := ""
	third := ""
	if p.consume(".") {
		second, err = p.parseIdentifier()
		if err != nil {
			return "", err
		}
	}
	if p.consume(".") {
		third, err = p.parseIdentifier()
		if err != nil {
			return "", err
		}
	}

	if first != "user" || second != "attributes" || third == "" {
		return "", errors.New("only user.attributes.<name> selectors are supported")
	}

	return strings.Join([]string{first, second, third}, "."), nil
}

func (p *parser) parseList() ([]string, error) {
	p.skipSpaces()
	if !p.consume("[") {
		return nil, errors.New("expected '['")
	}

	values := []string{}
	for {
		p.skipSpaces()
		if p.consume("]") {
			return values, nil
		}

		value, err := p.parseString()
		if err != nil {
			return nil, err
		}
		values = append(values, value)

		p.skipSpaces()
		if p.consume("]") {
			return values, nil
		}
		if !p.consume(",") {
			return nil, errors.New("expected ',' or ']' in list")
		}
	}
}

func (p *parser) parseString() (string, error) {
	p.skipSpaces()
	if p.eof() {
		return "", errors.New("expected string literal")
	}

	quote := p.peek()
	if quote != '\'' && quote != '"' {
		return "", errors.New("expected string literal")
	}

	p.pos++
	start := p.pos
	for !p.eof() && p.peek() != quote {
		if p.peek() == '\\' {
			p.pos += 2
			continue
		}
		p.pos++
	}
	if p.eof() {
		return "", errors.New("unterminated string literal")
	}

	raw := p.input[start:p.pos]
	p.pos++
	value, err := strconvUnquote(string(quote) + raw + string(quote))
	if err != nil {
		return "", err
	}
	return value, nil
}

func strconvUnquote(s string) (string, error) {
	value, err := strconv.Unquote(s)
	if err == nil {
		return value, nil
	}

	if strings.HasPrefix(s, "'") && strings.HasSuffix(s, "'") {
		return strings.TrimSuffix(strings.TrimPrefix(s, "'"), "'"), nil
	}

	return "", err
}

func (p *parser) parseIdentifier() (string, error) {
	p.skipSpaces()
	start := p.pos
	for !p.eof() {
		ch := p.peek()
		if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '_' {
			p.pos++
			continue
		}
		break
	}

	if start == p.pos {
		return "", errors.New("expected identifier")
	}

	return p.input[start:p.pos], nil
}

func (p *parser) consume(s string) bool {
	p.skipSpaces()
	if strings.HasPrefix(p.input[p.pos:], s) {
		p.pos += len(s)
		return true
	}
	return false
}

func (p *parser) consumeWord(word string) bool {
	p.skipSpaces()
	if !strings.HasPrefix(p.input[p.pos:], word) {
		return false
	}
	end := p.pos + len(word)
	if end < len(p.input) {
		next := p.input[end]
		if (next >= 'a' && next <= 'z') || (next >= 'A' && next <= 'Z') || (next >= '0' && next <= '9') || next == '_' {
			return false
		}
	}
	p.pos = end
	return true
}

func (p *parser) skipSpaces() {
	for !p.eof() {
		switch p.input[p.pos] {
		case ' ', '\n', '\r', '\t':
			p.pos++
		default:
			return
		}
	}
}

func (p *parser) eof() bool {
	return p.pos >= len(p.input)
}

func (p *parser) peek() byte {
	if p.eof() {
		return 0
	}
	return p.input[p.pos]
}
