package filter_test

import (
	"context"
	"testing"

	"github.com/SebTardifLabs/terraform-provider-coolify/internal/filter"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
)

func TestMatch_ExactMatch(t *testing.T) {
	t.Parallel()
	values := []types.String{types.StringValue("prod")}
	assert.True(t, filter.Match("prod", values))
	assert.False(t, filter.Match("staging", values))
}

func TestMatch_GlobPrefix(t *testing.T) {
	t.Parallel()
	values := []types.String{types.StringValue("prod-*")}
	assert.True(t, filter.Match("prod-us-east", values))
	assert.True(t, filter.Match("prod-", values))
	assert.False(t, filter.Match("staging-us", values))
	assert.False(t, filter.Match("prod", values))
}

func TestMatch_MultipleValues(t *testing.T) {
	t.Parallel()
	values := []types.String{types.StringValue("prod"), types.StringValue("staging")}
	assert.True(t, filter.Match("prod", values))
	assert.True(t, filter.Match("staging", values))
	assert.False(t, filter.Match("dev", values))
}

func TestMatch_EmptyValues(t *testing.T) {
	t.Parallel()
	assert.False(t, filter.Match("anything", nil))
	assert.False(t, filter.Match("anything", []types.String{}))
}

type testItem struct {
	Name   string
	Status string
	Count  int
}

func testAccessor(item testItem, field string) (string, bool) {
	switch field {
	case "name":
		return item.Name, true
	case "status":
		return item.Status, true
	case "count":
		return filter.IntToString(item.Count), true
	default:
		return "", false
	}
}

func TestApply_NoFilters(t *testing.T) {
	t.Parallel()
	items := []testItem{{Name: "a"}, {Name: "b"}}
	result := filter.Apply(context.Background(), items, nil, testAccessor)
	assert.Equal(t, items, result)
}

func TestApply_SingleFilter(t *testing.T) {
	t.Parallel()
	items := []testItem{
		{Name: "prod-web", Status: "running"},
		{Name: "staging-web", Status: "stopped"},
		{Name: "prod-api", Status: "running"},
	}
	filters := []filter.Config{
		{Name: types.StringValue("name"), Values: []types.String{types.StringValue("prod-*")}},
	}
	result := filter.Apply(context.Background(), items, filters, testAccessor)
	assert.Len(t, result, 2)
	assert.Equal(t, "prod-web", result[0].Name)
	assert.Equal(t, "prod-api", result[1].Name)
}

func TestApply_MultipleFiltersANDed(t *testing.T) {
	t.Parallel()
	items := []testItem{
		{Name: "prod-web", Status: "running"},
		{Name: "prod-api", Status: "stopped"},
		{Name: "staging-web", Status: "running"},
	}
	filters := []filter.Config{
		{Name: types.StringValue("name"), Values: []types.String{types.StringValue("prod-*")}},
		{Name: types.StringValue("status"), Values: []types.String{types.StringValue("running")}},
	}
	result := filter.Apply(context.Background(), items, filters, testAccessor)
	assert.Len(t, result, 1)
	assert.Equal(t, "prod-web", result[0].Name)
}

func TestApply_UnknownField(t *testing.T) {
	t.Parallel()
	items := []testItem{{Name: "a"}}
	filters := []filter.Config{
		{Name: types.StringValue("nonexistent"), Values: []types.String{types.StringValue("x")}},
	}
	result := filter.Apply(context.Background(), items, filters, testAccessor)
	assert.Empty(t, result)
}

func TestApply_EmptySlice(t *testing.T) {
	t.Parallel()
	filters := []filter.Config{
		{Name: types.StringValue("name"), Values: []types.String{types.StringValue("a")}},
	}
	result := filter.Apply(context.Background(), []testItem{}, filters, testAccessor)
	assert.Empty(t, result)
}

func TestApply_IntField(t *testing.T) {
	t.Parallel()
	items := []testItem{
		{Name: "a", Count: 5},
		{Name: "b", Count: 10},
	}
	filters := []filter.Config{
		{Name: types.StringValue("count"), Values: []types.String{types.StringValue("5")}},
	}
	result := filter.Apply(context.Background(), items, filters, testAccessor)
	assert.Len(t, result, 1)
	assert.Equal(t, "a", result[0].Name)
}

func TestBoolToString(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "true", filter.BoolToString(true))
	assert.Equal(t, "false", filter.BoolToString(false))
}

func TestInt64ToString(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "42", filter.Int64ToString(42))
	assert.Equal(t, "0", filter.Int64ToString(0))
}
