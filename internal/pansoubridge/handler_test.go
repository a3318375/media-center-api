package pansoubridge

import (
	"bytes"
	"context"
	"net/http/httptest"
	"pansou/model"
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
		`{"query":"movie","plugins":["bad name"]}`,
		`{"query":"movie","channels":["bad name"]}`,
		`{"query":"movie","mode":"invalid"}`,
		`{"query":"movie","plugins":["a","a","a","a","a","a","a","a","a","a","a","a","a","a","a","a","a","a","a","a","a","a","a","a","a","a","a","a","a","a","a","a","a","a","a","a","a","a","a","a","a"]}`,
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

type captureSearch struct{ args []interface{} }

func (f *captureSearch) Search(keyword string, channels []string, concurrency int, refresh bool, mode, source string, plugins, clouds []string, ext map[string]interface{}) (model.SearchResponse, error) {
	f.args = []interface{}{keyword, channels, concurrency, refresh, mode, source, plugins, clouds}
	return model.SearchResponse{}, nil
}

func TestAdapterPassesExtendedArguments(t *testing.T) {
	fake := &captureSearch{}
	_, err := (searchAdapter{service: fake}).Search(context.Background(), "movie", "plugin", []string{"baidu"}, []string{"5266ys"}, []string{"tgsearchers7"}, "results", true)
	if err != nil {
		t.Fatal(err)
	}
	if len(fake.args) != 8 || fake.args[1].([]string)[0] != "tgsearchers7" || fake.args[4] != "results" || fake.args[6].([]string)[0] != "5266ys" {
		t.Fatalf("arguments not passed through: %#v", fake.args)
	}
}

func TestMetaHandlerReturnsRegisteredOptions(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/meta", MetaHandler())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/meta", nil))
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"plugins"`) || !strings.Contains(w.Body.String(), `"defaults"`) || !strings.Contains(w.Body.String(), `"cloudTypes"`) {
		t.Fatalf("unexpected meta response: %d %s", w.Code, w.Body.String())
	}
}
