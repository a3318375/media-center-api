package pansoubridge

import (
	"bytes"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestHandlerRejectsInvalidRequests(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := (&Bootstrap{}).Handler()
	cases := []string{
		`{}`,
		`{"query":"` + strings.Repeat("a", 121) + `"}`,
		`{"query":"movie","source":"admin"}`,
		`{"query":"movie","cloudTypes":["a","b","c","d","e"]}`,
		`{"query":"movie","cloudTypes":["bad type"]}`,
	}
	for _, body := range cases {
		r := gin.New()
		r.POST("/search", h)
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/search", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != 400 {
			t.Errorf("body %q: status=%d response=%s", body, w.Code, w.Body.String())
		}
	}
}
