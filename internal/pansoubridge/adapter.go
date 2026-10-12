package pansoubridge

import (
	"context"
	"time"

	"pansou/model"
	"pansou/plugin"
	"pansou/service"
)

type resourceLink struct {
	Cloud     string    `json:"cloud"`
	URL       string    `json:"url"`
	Password  string    `json:"password,omitempty"`
	Note      string    `json:"note,omitempty"`
	Source    string    `json:"source,omitempty"`
	UpdatedAt time.Time `json:"updatedAt,omitempty"`
}
type resourceItem struct {
	Title       string         `json:"title"`
	Description string         `json:"description,omitempty"`
	Channel     string         `json:"channel,omitempty"`
	UpdatedAt   time.Time      `json:"updatedAt,omitempty"`
	Tags        []string       `json:"tags,omitempty"`
	Links       []resourceLink `json:"links,omitempty"`
}
type searchData struct {
	Total        int                       `json:"total"`
	Items        []resourceItem            `json:"items,omitempty"`
	LinksByCloud map[string][]resourceLink `json:"linksByCloud,omitempty"`
}
type searchAdapter struct{ service *service.SearchService }

// Search translates Media Center concepts to PanSou arguments and maps upstream
// data into a stable local DTO. No PanSou HTTP contract is exposed.
func (a searchAdapter) Search(ctx context.Context, keyword, source string, cloudTypes []string, refresh bool) (searchData, error) {
	ext := map[string]interface{}{"refresh": refresh, plugin.ExtContextKey: ctx}
	resp, err := a.service.Search(keyword, nil, 4, refresh, "all", source, nil, cloudTypes, ext)
	if err != nil {
		return searchData{}, err
	}
	return mapSearchResponse(resp), nil
}

func mapSearchResponse(resp model.SearchResponse) searchData {
	data := searchData{Total: resp.Total, Items: make([]resourceItem, 0, len(resp.Results)), LinksByCloud: make(map[string][]resourceLink, len(resp.MergedByType))}
	for _, item := range resp.Results {
		out := resourceItem{Title: item.Title, Description: item.Content, Channel: item.Channel, UpdatedAt: item.Datetime, Tags: item.Tags}
		for _, link := range item.Links {
			out.Links = append(out.Links, resourceLink{Cloud: link.Type, URL: link.URL, Password: link.Password, UpdatedAt: link.Datetime})
		}
		data.Items = append(data.Items, out)
	}
	for cloud, links := range resp.MergedByType {
		for _, link := range links {
			data.LinksByCloud[cloud] = append(data.LinksByCloud[cloud], resourceLink{Cloud: cloud, URL: link.URL, Password: link.Password, Note: link.Note, Source: link.Source, UpdatedAt: link.Datetime})
		}
	}
	return data
}
