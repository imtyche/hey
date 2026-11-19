package main

import (
	"fmt"
	"net/http"
	"os"
	"strings"
)

// httpFileRequest 表示从 .http 文件中解析出的单个请求。
type httpFileRequest struct {
	Method  string
	URL     string
	Headers http.Header
	Body    []byte
}

// parseHTTPFile 解析 JetBrains/VS Code REST Client 风格的 .http 文件。
// 文件中允许以 "###" 分隔多个请求块，但必须有且仅有一个请求，
// 否则返回错误。
func parseHTTPFile(path string) (*httpFileRequest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")

	// 按 "###" 分隔线切块。
	var blocks [][]string
	cur := []string{}
	for _, ln := range lines {
		if strings.HasPrefix(strings.TrimSpace(ln), "###") {
			blocks = append(blocks, cur)
			cur = []string{}
			continue
		}
		cur = append(cur, ln)
	}
	blocks = append(blocks, cur)

	var reqs []*httpFileRequest
	for _, b := range blocks {
		r, err := parseHTTPBlock(b)
		if err != nil {
			return nil, err
		}
		if r != nil {
			reqs = append(reqs, r)
		}
	}

	switch len(reqs) {
	case 0:
		return nil, fmt.Errorf("文件中未找到请求：%s", path)
	case 1:
		return reqs[0], nil
	default:
		return nil, fmt.Errorf("只允许包含一个请求，但文件中包含 %d 个：%s", len(reqs), path)
	}
}

// parseHTTPBlock 解析一个请求块。块内只有注释或空白时返回 nil, nil。
func parseHTTPBlock(lines []string) (*httpFileRequest, error) {
	i := 0
	// 跳过块开头的空行与注释行。
	for ; i < len(lines); i++ {
		t := strings.TrimSpace(lines[i])
		if t == "" || isComment(t) {
			continue
		}
		break
	}
	if i == len(lines) {
		return nil, nil
	}

	// 请求行：METHOD URL [HTTP/x.y]
	fields := strings.Fields(lines[i])
	i++
	if len(fields) < 2 {
		return nil, fmt.Errorf("非法请求行：%q", strings.TrimSpace(lines[i-1]))
	}
	method := strings.ToUpper(fields[0])
	if !isValidMethodToken(method) {
		return nil, fmt.Errorf("非法请求行：%q", strings.TrimSpace(lines[i-1]))
	}
	url := fields[1]

	// 请求头：直到第一个空行为止。
	header := make(http.Header)
	for ; i < len(lines); i++ {
		t := strings.TrimSpace(lines[i])
		if t == "" {
			i++
			break
		}
		if isComment(t) {
			continue
		}
		idx := strings.Index(t, ":")
		if idx <= 0 {
			return nil, fmt.Errorf("非法请求头行：%q", t)
		}
		header.Add(strings.TrimSpace(t[:idx]), strings.TrimSpace(t[idx+1:]))
	}
	// 正文：其余全部内容，去掉末尾空白。
	var bodyLines []string
	for ; i < len(lines); i++ {
		bodyLines = append(bodyLines, lines[i])
	}
	req := &httpFileRequest{Method: method, URL: url, Headers: header}
	if b := strings.TrimRight(strings.Join(bodyLines, "\n"), " \t\n"); b != "" {
		req.Body = []byte(b)
	}
	return req, nil
}

func isComment(t string) bool {
	return strings.HasPrefix(t, "#") || strings.HasPrefix(t, "//")
}

// isValidMethodToken 校验请求方法只由字母组成（http.NewRequest 会做进一步校验）。
func isValidMethodToken(m string) bool {
	if m == "" {
		return false
	}
	for _, r := range m {
		if (r < 'A' || r > 'Z') && (r < 'a' || r > 'z') {
			return false
		}
	}
	return true
}
