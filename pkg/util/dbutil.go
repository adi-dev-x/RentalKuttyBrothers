package util

import (
	"fmt"
	"strings"
)

type QueryBuilders struct {
	Key       string `json:"key"`
	Condition string `json:"condition"`
	Value     string `json:"value"`
	Table     string `json:"table"`
	Joins     string `json:"joins"`
	//JsonKey        *string              `json:"json_key"`
	//GroupCondition *GroupWhereCondition `json:"group_condition"`
	//SubQuery       *SubQueryCondition   `json:"sub_query"`
}

func QueryBuilder(handler Handler, typeQuery string) (string, []interface{}) {
	var conditions []string
	var args []interface{}
	var joinsBuilder strings.Builder

	for _, value := range handler.KeyValue {
		// collect joins if any
		if handler.Joins != "" {
			joinsBuilder.WriteString(" ")
			joinsBuilder.WriteString(handler.Joins)
		}

		// build condition with placeholder
		var condition string
		placeholder := "?"
		if typeQuery == "JOIN" && handler.Table != "" {
			condition = fmt.Sprintf("%s.%s %s %s", handler.Table, value.Key, value.Condition, placeholder)
		} else {
			condition = fmt.Sprintf("%s %s %s", value.Key, value.Condition, placeholder)
		}

		conditions = append(conditions, condition)
		args = append(args, value.Value) // safe param binding
	}

	query := joinsBuilder.String()
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	// optional: add limit + offset from Handler
	if handler.Limit != "" {
		query += " LIMIT " + handler.Limit
	}
	if handler.Offset != "" {
		query += " OFFSET " + handler.Offset
	}

	return query, args
}
