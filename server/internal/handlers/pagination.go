package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// listParams is the shared, clamped set of pagination/sort inputs every
// collection endpoint accepts. Centralizing the parse keeps defaults and
// limits identical across lists (page 1, 50 rows per page, hard cap 200).
type listParams struct {
	Page   int
	Limit  int
	Offset int
	Search string
	Order  string // "ASC" or "DESC"
}

func parseListParams(c *gin.Context) listParams {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 200 {
		limit = 50
	}
	order := "DESC"
	if strings.EqualFold(c.Query("order"), "asc") {
		order = "ASC"
	}
	return listParams{
		Page:   page,
		Limit:  limit,
		Offset: (page - 1) * limit,
		Search: strings.TrimSpace(c.Query("search")),
		Order:  order,
	}
}

// sortColumn resolves the client's `sort` key against an allowlist so raw
// input never reaches SQL. Unknown or empty keys fall back to defaultCol.
func (p listParams) sortColumn(c *gin.Context, allowed map[string]string, defaultCol string) string {
	if col := allowed[c.Query("sort")]; col != "" {
		return col
	}
	return defaultCol
}

// respondPaginated writes the canonical list envelope every endpoint returns:
// {data, total, page, limit}. data is expected to be a non-nil slice (asArray).
func respondPaginated(c *gin.Context, data any, total int64, p listParams) {
	c.JSON(http.StatusOK, gin.H{
		"data":  data,
		"total": total,
		"page":  p.Page,
		"limit": p.Limit,
	})
}
