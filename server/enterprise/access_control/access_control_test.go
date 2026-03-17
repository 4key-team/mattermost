// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

//go:build sourceavailable && !enterprise

package accesscontrol

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseExpressionSupportsBasicOperators(t *testing.T) {
	node, err := parseExpression(`user.attributes.department == "Engineering" && ["Remote", "Hybrid"] in user.attributes.workplace`)
	require.NoError(t, err)

	conditions, ok := flattenAndConditions(node)
	require.True(t, ok)
	require.Len(t, conditions, 2)
	require.Equal(t, "user.attributes.department", conditions[0].Attribute)
	require.Equal(t, "==", conditions[0].Operator)
	require.Equal(t, []string{"Engineering"}, conditions[0].Values)
	require.Equal(t, "array_in", conditions[1].Operator)
}

func TestCompileSQLBuildsNestedBooleanExpression(t *testing.T) {
	node, err := parseExpression(`user.attributes.department == "Engineering" || user.attributes.email.endsWith("example.com")`)
	require.NoError(t, err)

	query, args, err := compileSQL(node)
	require.NoError(t, err)
	require.Equal(t, "((Attributes ->> 'department' = $1) || (Attributes ->> 'email' LIKE $2))", query)
	require.Equal(t, []any{"Engineering", "%example.com"}, args)
}

func TestConditionEvaluateSupportsMultiselectContainsAll(t *testing.T) {
	node := conditionNode{
		Attribute: "user.attributes.tags",
		Operator:  "array_in",
		Values:    []string{"alpha", "beta"},
	}

	require.True(t, node.Evaluate(map[string]any{
		"tags": []any{"alpha", "beta", "gamma"},
	}))
	require.False(t, node.Evaluate(map[string]any{
		"tags": []any{"alpha"},
	}))
}
