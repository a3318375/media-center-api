package anime

import (
	"context"
	"net"
	"strings"
	"testing"
)

func validDraft() Draft {
	return Draft{Title: "测试动画", BangumiID: "123", Source: "other", URL: "https://example.com/rss", Season: 1, Match: []string{}, Exclude: []string{}, Standby: []Feed{}}
}
func TestRulesAndOffset(t *testing.T) {
	d := validDraft()
	d.Offset = -12
	d.Match = []string{`1080p`, `简`}
	d.Exclude = []string{`合集`}
	d.TotalEpisodes = 12
	entries := []Entry{{Title: "[组] 动画 - 13 [1080p 简]", URL: "magnet:?xt=urn:btih:a"}, {Title: "[组] 动画 - 14 [1080p 繁]", URL: "https://x.org/a.torrent"}, {Title: "动画 第15集 [1080p 简 合集]", URL: "https://x.org/b.torrent"}, {Title: "动画 S01E13 [1080p 简]", URL: "https://x.org/c.torrent"}, {Title: "动画 第16集 [1080p 简]"}}
	out := Apply(d, entries, d.Offset, map[float64]bool{})
	if !out[0].Accepted || *out[0].Episode != 1 {
		t.Fatalf("offset not applied: %+v", out[0])
	}
	for _, it := range out[1:] {
		if it.Accepted {
			t.Fatalf("invalid entry accepted: %+v", it)
		}
	}
	if !strings.Contains(out[0].FileName, "S01E01") {
		t.Fatal(out[0].FileName)
	}
}
func TestValidateRejectsBadInput(t *testing.T) {
	for _, change := range []func(*Draft){func(d *Draft) { d.Match = []string{"["} }, func(d *Draft) { d.URL = "file:///etc/passwd" }, func(d *Draft) { d.BangumiID = "0" }, func(d *Draft) { d.Season = 0 }, func(d *Draft) { d.Standby = []Feed{{URL: d.URL}} }} {
		d := validDraft()
		change(&d)
		if Validate(&d) == nil {
			t.Fatal("invalid input accepted")
		}
	}
}
func TestPreviewReadsEveryFeedAndDoesNotPersist(t *testing.T) {
	s := New()
	calls := 0
	s.Fetch = func(_ context.Context, _ string) ([]byte, error) {
		calls++
		return []byte(`<rss version="2.0"><channel><item><title>番剧 第01集</title><enclosure url="https://example.com/a.torrent"/></item></channel></rss>`), nil
	}
	d := validDraft()
	d.Standby = []Feed{{Label: "备用", URL: "https://example.com/backup"}}
	p, e := s.Preview(context.Background(), d)
	if e != nil || p.Accepted != 1 || len(p.Items) != 2 || calls != 2 {
		t.Fatalf("%+v %v calls=%d", p, e, calls)
	}
	_, e = s.Preview(context.Background(), d)
	if e != nil || calls != 4 {
		t.Fatal("RSS incorrectly cached")
	}
}
func TestRejectLocalAddresses(t *testing.T) {
	for _, ip := range []string{"127.0.0.1", "10.0.0.1", "169.254.169.254", "::1", "192.168.1.2", "0.0.0.0"} {
		if publicIP(net.ParseIP(ip)) {
			t.Fatal(ip)
		}
	}
	if _, e := fetchPublic(context.Background(), "http://127.0.0.1/a"); e == nil {
		t.Fatal("local fetch allowed")
	}
}
func TestCatalogAndGroupsCache(t *testing.T) {
	s := New()
	calls := 0
	s.Fetch = func(_ context.Context, raw string) ([]byte, error) {
		calls++
		if strings.Contains(raw, "Bangumi/123") {
			return []byte(`<a href="https://bgm.tv/subject/456">BGM</a><div class="leftbar-item"><a class="subgroup-name" data-anchor="#1">组</a></div><div id="1"><a class="mikan-rss" href="/RSS/Bangumi?bangumiId=123&amp;subgroupid=1">RSS</a></div>`), nil
		}
		return []byte(`<div class="sk-bangumi"><div>星期二</div><ul><li><span data-src="/a.jpg"></span><a href="/Home/Bangumi/123">动画</a></li></ul></div>`), nil
	}
	c, e := s.Catalog(context.Background(), "", "", "", false)
	if e != nil || len(c.Items) != 1 {
		t.Fatalf("%+v %v", c, e)
	}
	_, _ = s.Catalog(context.Background(), "", "", "", false)
	g, e := s.Groups(context.Background(), "123", false)
	if e != nil || len(g) != 1 || g[0].BangumiID != "456" {
		t.Fatalf("%+v %v", g, e)
	}
	_, _ = s.Groups(context.Background(), "123", false)
	if calls != 2 {
		t.Fatal(calls)
	}
	_, _ = s.Catalog(context.Background(), "", "", "", true)
	if calls != 3 {
		t.Fatal("refresh did not bypass cache")
	}
}
