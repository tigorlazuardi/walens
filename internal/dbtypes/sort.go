package dbtypes

import (
	"fmt"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-jet/jet/v2/sqlite"
)

type Sort struct {
	Field string  `json:"field" doc:"item key to be sorted with" minLength:"1"`
	Order *string `json:"order" doc:"direction of sort" enum:"asc,desc" default:"asc"`
}

func (s *Sort) IsValid(fields sqlite.ColumnList) bool {
	if s == nil {
		return true
	}
	for _, field := range fields {
		if field.Name() == s.Field {
			return true
		}
	}
	return false
}

type SortList []Sort

func (list SortList) BuildOrderByClause(fields sqlite.ColumnList) ([]sqlite.OrderByClause, error) {
	if len(list) == 0 {
		return []sqlite.OrderByClause{}, nil
	}
	clauses := make([]sqlite.OrderByClause, 0, len(list))
	errs := make([]error, 0, len(list))
	for _, sort := range list {
		if sort.Field == "" {
			continue
		}
		if sort.IsValid(fields) {
			field := sqlite.String(sort.Field)
			orderBy := field.ASC()
			if sort.Order != nil && *sort.Order == "desc" {
				orderBy = field.DESC()
			}
			clauses = append(clauses, orderBy)
		} else {
			errs = append(errs,
				fmt.Errorf(`field '%s' does not exist on resource '%s'`, sort.Field, fields.TableName()),
			)
		}
	}
	if len(errs) > 0 {
		return clauses, huma.Error400BadRequest("sort field validation failed", errs...)
	}
	return clauses, nil
}
