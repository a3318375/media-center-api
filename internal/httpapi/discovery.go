package httpapi

import (
	"context"
	"errors"
	"github.com/gin-gonic/gin"
	"media-center/internal/tmdb"
	"net/http"
	"strconv"
	"time"
)

func (a *API) mediaKey(c *gin.Context) (string, bool) {
	key, err := a.store.ReadSecret("tmdb", "tmdb.apiKey")
	if err != nil || key == "" {
		fail(c, 503, "请先在设置中配置 TMDB API Key")
		return "", false
	}
	return key, true
}
func mediaFailure(c *gin.Context, err error) {
	var e *tmdb.Error
	if errors.As(err, &e) {
		if e.RetryAfter != "" {
			c.Header("Retry-After", e.RetryAfter)
		}
		fail(c, e.Status, e.Message)
	} else if errors.Is(err, context.DeadlineExceeded) {
		fail(c, 504, "TMDB 请求超时")
	} else if errors.Is(err, context.Canceled) {
		fail(c, 408, "请求已取消")
	} else {
		fail(c, 502, "读取 TMDB 失败")
	}
}
func (a *API) discoverMedia(c *gin.Context) {
	if c.Query("source") != "tmdb" {
		fail(c, 400, "本接口仅支持 TMDB")
		return
	}
	page := 1
	var err error
	if s := c.Query("page"); s != "" {
		page, err = strconv.Atoi(s)
		if err != nil {
			fail(c, 400, "page 无效")
			return
		}
	}
	if size := c.Query("pageSize"); size != "" && size != "20" {
		fail(c, 400, "TMDB 每页固定 20 条")
		return
	}
	q := tmdb.Query{Type: c.Query("mediaType"), Page: page, Genre: c.Query("genre"), Region: c.Query("region"), Year: c.Query("year"), Sort: c.Query("sort"), Search: c.Query("query")}
	if err := q.Validate(); err != nil {
		mediaFailure(c, err)
		return
	}
	key, ok := a.mediaKey(c)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
	defer cancel()
	result, err := a.tmdb.Discover(ctx, key, q)
	if err != nil {
		mediaFailure(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}
func (a *API) detailMedia(c *gin.Context) {
	if c.Param("source") != "tmdb" {
		fail(c, 400, "本接口仅支持 TMDB")
		return
	}
	key, ok := a.mediaKey(c)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
	defer cancel()
	d, err := a.tmdb.Detail(ctx, key, c.Query("mediaType"), c.Param("sourceId"))
	if err != nil {
		mediaFailure(c, err)
		return
	}
	c.JSON(200, d)
}
