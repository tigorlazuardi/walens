package dbtypes

import "github.com/go-jet/jet/v2/sqlite"

type CursorPaginationRequest struct {
	Next   *UUID    `json:"next" doc:"unique token for next page"`
	Prev   *UUID    `json:"prev" doc:"unique token for prev page. If both next and prev is provided, next has precedence over prev"`
	Offset *int     `json:"offset" doc:"offset value to skip number of items. Use offset with next or prev to create 'page numbers' with formula of (page - 1) * limit. Keep using positive offset even if using prev"`
	Limit  *int     `json:"limit" doc:"number of items to fetch. Limit value varies between endpoints and each of those endpoints may have different default values"`
	Sorts  SortList `json:"sorts" doc:"sort results. Keep sorts stable across paginated requests for expected results"`
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

func (c *CursorPaginationRequest) BuildOrderByClause(fields sqlite.ColumnList) ([]sqlite.OrderByClause, error) {
	if c == nil {
		return []sqlite.OrderByClause{}, nil
	}
	return c.Sorts.BuildOrderByClause(fields)
}

type CursorPaginationResponse struct {
	Next *UUID `json:"next" doc:"unique token for next page"`
	Prev *UUID `json:"prev" doc:"unique token for prev page"`
}
