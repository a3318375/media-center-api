package pansoubridge

import (
	"context"
	"errors"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

var cloudTypeRE = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,24}$`)

type searchRequest struct {
	Query      string   `json:"query"`
	Source     string   `json:"source"`
	CloudTypes []string `json:"cloudTypes"`
	Refresh    bool     `json:"refresh"`
}

// Handler is Media Center's own validated endpoint; mount behind requireAPI.
func (b *Bootstrap) Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 8<<10)
		var req searchRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid search request"})
			return
		}
		req.Query = strings.TrimSpace(req.Query)
		if n := len([]rune(req.Query)); n < 1 || n > 120 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "query must contain 1 to 120 characters"})
			return
		}
		if req.Source == "" {
			req.Source = "all"
		}
		if req.Source != "all" && req.Source != "tg" && req.Source != "plugin" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "source must be all, tg, or plugin"})
			return
		}
		if len(req.CloudTypes) > 4 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "at most 4 cloud types are allowed"})
			return
		}
		for _, typ := range req.CloudTypes {
			if !cloudTypeRE.MatchString(typ) {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid cloud type"})
				return
			}
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 25*time.Second)
		defer cancel()
		result, err := b.Search(ctx, req.Query, req.Source, req.CloudTypes, req.Refresh)
		if err != nil {
			status := http.StatusServiceUnavailable
			if errors.Is(err, context.DeadlineExceeded) {
				status = http.StatusGatewayTimeout
			}
			c.JSON(status, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": result})
	}
}
