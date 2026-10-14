package pan115

import (
	"fmt"
	"strconv"
	"strings"

	sdk "github.com/SheltonZhu/115driver/pkg/driver"
)

type ShareGateway interface {
	Gateway
	ListShare(code, password, directory string) ([]File, error)
	CreateDirectory(parent, name string) (File, error)
	ReceiveShare(code, password string, ids []string, directory string) error
}

func (c *Client) ListShare(code, password, directory string) ([]File, error) {
	out := []File{}
	for offset := 0; offset < 10000; offset += 100 {
		var result struct {
			sdk.BasicResp
			Data struct {
				Count int             `json:"count"`
				List  []sdk.ShareFile `json:"list"`
			} `json:"data"`
		}
		resp, err := c.sdk.NewRequest().SetQueryParams(map[string]string{"share_code": code, "receive_code": password, "cid": directory, "limit": "100", "offset": strconv.Itoa(offset), "format": "json"}).SetHeader("Referer", sdk.BuildShareReferer(code, password)).ForceContentType("application/json").SetResult(&result).Get(sdk.ApiShareSnap)
		if err = sdk.CheckErr(err, &result, resp); err != nil {
			return nil, SafeError(err)
		}
		if result.Data.Count > 10000 {
			return nil, &Error{"scan_limit", "115 分享目录超过查询限制，请选择更小的子目录", true}
		}
		if len(result.Data.List) == 0 && offset < result.Data.Count {
			return nil, &Error{"upstream", "115 分享列表分页不完整，稍后重试", false}
		}
		for _, f := range result.Data.List {
			id := f.FileID
			isDir := id == ""
			if isDir {
				id = string(f.CategoryID)
			}
			if !ValidDirectory(id) || id == "0" {
				return nil, &Error{"upstream", "115 分享文件标识无效", false}
			}
			out = append(out, File{ID: id, ParentID: directory, Name: f.FileName, Size: int64(f.Size), SHA1: f.Sha1, Directory: isDir})
		}
		if offset+100 >= result.Data.Count {
			return out, nil
		}
	}
	return nil, &Error{"scan_limit", "115 分享目录超过查询限制", true}
}
func (c *Client) CreateDirectory(parent, name string) (File, error) {
	id, err := c.sdk.Mkdir(parent, name)
	if err != nil {
		return File{}, SafeError(err)
	}
	if !ValidDirectory(id) || id == "0" {
		return File{}, &Error{"uncertain", "115 未返回创建目录标识，等待核对", false}
	}
	f, err := c.GetFile(id)
	if err != nil {
		return File{}, err
	}
	if !f.Directory || f.ParentID != parent {
		return File{}, &Error{"uncertain", "115 新建目录尚未确认", false}
	}
	return f, nil
}
func (c *Client) ReceiveShare(code, password string, ids []string, directory string) error {
	if len(ids) == 0 || len(ids) > 100 {
		return fmt.Errorf("分享选择数量无效")
	}
	var result sdk.BasicResp
	resp, err := c.sdk.NewRequest().SetFormData(map[string]string{"share_code": code, "receive_code": password, "file_id": strings.Join(ids, ","), "cid": directory, "user_id": strconv.FormatInt(c.sdk.UserID, 10)}).ForceContentType("application/json").SetResult(&result).Post("https://webapi.115.com/share/receive")
	if err = sdk.CheckErr(err, &result, resp); err != nil {
		return SafeError(err)
	}
	return nil
}
