package main

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeTempHTTP(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "req.http")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestParseHTTPFilePostWithBody(t *testing.T) {
	path := writeTempHTTP(t, `### 创建用户
POST https://example.com/api/users HTTP/1.1
Content-Type: application/json
X-Trace: abc123

{
  "name": "hey",
  "items": [1, 2, 3]
}
`)
	req, err := parseHTTPFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if req.Method != "POST" {
		t.Errorf("Method = %q, want POST", req.Method)
	}
	if req.URL != "https://example.com/api/users" {
		t.Errorf("URL = %q", req.URL)
	}
	if got := req.Headers.Get("X-Trace"); got != "abc123" {
		t.Errorf("X-Trace = %q", got)
	}
	if got := req.Headers.Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type = %q", got)
	}
	want := "{\n  \"name\": \"hey\",\n  \"items\": [1, 2, 3]\n}"
	if string(req.Body) != want {
		t.Errorf("Body = %q, want %q", req.Body, want)
	}
}

func TestParseHTTPFileGetNoBody(t *testing.T) {
	path := writeTempHTTP(t, `# 注释
// 另一种注释
GET https://example.com/health
Accept: text/plain
`)
	req, err := parseHTTPFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if req.Method != "GET" || req.URL != "https://example.com/health" {
		t.Errorf("got %s %s", req.Method, req.URL)
	}
	if len(req.Body) != 0 {
		t.Errorf("Body = %q, want empty", req.Body)
	}
	if got := req.Headers.Get("Accept"); got != "text/plain" {
		t.Errorf("Accept = %q", got)
	}
}

func TestParseHTTPFileMultipleRequestsError(t *testing.T) {
	path := writeTempHTTP(t, `GET https://example.com/a

###

POST https://example.com/b
Content-Type: text/plain

body
`)
	_, err := parseHTTPFile(path)
	if err == nil || !strings.Contains(err.Error(), "只允许包含一个请求") {
		t.Fatalf("err = %v, want 只允许包含一个请求", err)
	}
}

func TestParseHTTPFileNoRequestError(t *testing.T) {
	path := writeTempHTTP(t, `# 只有注释

`)
	_, err := parseHTTPFile(path)
	if err == nil || !strings.Contains(err.Error(), "未找到请求") {
		t.Fatalf("err = %v, want 未找到请求", err)
	}
}

func TestParseHTTPFileInvalidLines(t *testing.T) {
	cases := map[string]string{
		"bad request line": "GET\n",
		"bad method token": "G3T https://example.com\n",
		"bad header line":  "GET https://example.com\nnot-a-header\n",
	}
	for name, content := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := parseHTTPFile(writeTempHTTP(t, content)); err == nil {
				t.Fatal("want error, got nil")
			}
		})
	}
}

func TestParseHTTPFileCRLFAndTrailingBlank(t *testing.T) {
	path := writeTempHTTP(t, "POST https://example.com/api\r\nContent-Type: text/plain\r\n\r\nhello\r\n\r\n")
	req, err := parseHTTPFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(req.Body) != "hello" {
		t.Errorf("Body = %q, want %q", req.Body, "hello")
	}
}

// TestHTTPFileRequestEndToEnd 模拟 hey 的构建流程：解析 .http 文件、
// 应用 CLI 覆盖规则，并实际发送请求验证服务端收到的内容。
func TestHTTPFileRequestEndToEnd(t *testing.T) {
	path := writeTempHTTP(t, `POST https://placeholder.invalid/api
Content-Type: application/json
X-From-File: yes

{"k":"v"}
`)
	fh, err := parseHTTPFile(path)
	if err != nil {
		t.Fatal(err)
	}

	var got *http.Request
	var gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		got = r
		w.WriteHeader(200)
	}))
	defer srv.Close()

	// 与 hey.go 相同的合并规则：URL 位置参数覆盖文件；显式 -H 覆盖文件头。
	header := fh.Headers
	header.Set("Content-Type", header.Get("Content-Type")) // -T 未显式指定，保留文件值
	header.Set("X-Override", "cli")
	req, err := http.NewRequest(fh.Method, srv.URL+"/api", bytes.NewReader(fh.Body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header = header

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	if got.Method != "POST" {
		t.Errorf("server got method %q", got.Method)
	}
	if gotBody != `{"k":"v"}` {
		t.Errorf("server got body %q", gotBody)
	}
	if got.Header.Get("X-From-File") != "yes" {
		t.Error("file header not sent")
	}
	if got.Header.Get("X-Override") != "cli" {
		t.Error("cli header not sent")
	}
	if got.Header.Get("Content-Type") != "application/json" {
		t.Errorf("Content-Type = %q", got.Header.Get("Content-Type"))
	}
}
