package dbtypes

import "github.com/go-jet/jet/v2/sqlite"

type CursorPaginationRequest struct {
	Next   *UUID    `json:"next" doc:"Unique token for next page"`
	Prev   *UUID    `json:"prev" doc:"Unique token for prev page. If both next and prev is provided, next has precedence over prev"`
	Offset *int     `json:"offset" doc:"Offset value to skip number of items. Use offset with next or prev to create 'page numbers' with formula of (page - 1) * limit. Keep using positive offset even if using prev"`
	Limit  *int     `json:"limit" doc:"Number of items to fetch. Limit value varies between endpoints and each of those endpoints may have different default values"`
	Sorts  SortList `json:"sorts" doc:"Sort results. Keep sorts stable across paginated requests for expected results"`
}

func (c *CursorPaginationRequest) GetLimitOrDefault(def int64, maximum int64) int64 {
	if c == nil {
		return def
	}
	if c.Limit == nil {
		return def
	}
	l := *c.Limit
	if l <= 0 {
		return def
	}
	return min(maximum, int64(l))
}

func (c *CursorPaginationRequest) NextToken() string {
	if c == nil {
		return ""
	}
	if c.Next == nil {
		return ""
	}
	return c.Next.String()
}

func (c *CursorPaginationRequest) PrevToken() string {
	if c == nil {
		return ""
	}
	if c.Prev == nil {
		return ""
	}
	return c.Prev.String()
}

func (c *CursorPaginationRequest) GetOffset() int64 {
	if c == nil {
		return 0
	}
	if c.Offset == nil {
		return 0
	}
	return max(int64(*c.Offset), 0)
}

// BuildOrderByClause builds the ORDER BY clauses for SQL queries based on the Sorts configuration.
//
// This method validates each sort field against the provided column list and returns
// the appropriate ORDER BY clauses for use with Go-Jet query builder.
//
// Parameters:
//   - fields: A ColumnList from the generated Go-Jet table (e.g., Devices.AllColumns)
//     containing the valid columns that can be sorted.
//
// Returns:
//   - []sqlite.OrderByClause: A slice of ORDER BY clauses to be used with ORDER_BY()
//   - error: Returns huma.Error400BadRequest if any sort field is invalid (not found in fields)
//
// Behavior:
//   - If c is nil, returns empty clauses with no error (no sorting applied)
//   - If c.Sorts is empty, returns empty clauses with no error (no sorting applied)
//   - Invalid sort fields are collected and returned as a single 400 error
//   - Valid sort fields are returned as clauses even if some fields are invalid
//   - Sort order defaults to ASC if not specified or if Order is nil/empty
//   - Sort order must be exactly "desc" (case-sensitive) for descending order
//
// Example usage:
//
//	orderBy, err := req.Pagination.BuildOrderByClause(Devices.AllColumns)
//	if err != nil {
//	    return error // huma.Error400BadRequest with details
//	}
//	if len(orderBy) == 0 {
//	    orderBy = append(orderBy, Devices.Name.ASC()) // default sort
//	}
//
//	stmt := SELECT(Devices.AllColumns).
//	    FROM(Devices).
//	    ORDER_BY(orderBy...)
func (c *CursorPaginationRequest) BuildOrderByClause(fields sqlite.ColumnList) ([]sqlite.OrderByClause, error) {
	if c == nil {
		return []sqlite.OrderByClause{}, nil
	}
	return c.Sorts.BuildOrderByClause(fields)
}

type CursorPaginationResponse struct {
	Next *UUID `json:"next,omitzero" doc:"unique token for next page"`
	Prev *UUID `json:"prev,omitzero" doc:"unique token for prev page"`
}
