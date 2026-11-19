// Copyright 2014 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Command hey is an HTTP load generator.
package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"math"
	"net/http"
	gourl "net/url"
	"os"
	"os/signal"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/rakyll/hey/requester"
)

const (
	headerRegexp = `^([\w-]+):\s*(.+)`
	authRegexp   = `^(.+):([^\s].+)`
	heyUA        = "hey/0.0.1"
)

var (
	m           = flag.String("m", "", "")
	headers     = flag.String("h", "", "")
	body        = flag.String("d", "", "")
	bodyFile    = flag.String("D", "", "")
	httpFile    = flag.String("f", "", "")
	accept      = flag.String("A", "", "")
	contentType = flag.String("T", "", "")
	authHeader  = flag.String("a", "", "")
	hostHeader  = flag.String("host", "", "")
	userAgent   = flag.String("U", "", "")

	output = flag.String("o", "", "")

	c = flag.Int("c", 50, "")
	n = flag.Int("n", 200, "")
	q = flag.Float64("q", 0, "")
	t = flag.Int("t", 20, "")
	z = flag.Duration("z", 0, "")

	h2   = flag.Bool("h2", false, "")
	cpus = flag.Int("cpus", runtime.GOMAXPROCS(-1), "")

	disableCompression = flag.Bool("disable-compression", false, "")
	disableKeepAlives  = flag.Bool("disable-keepalive", false, "")
	disableRedirects   = flag.Bool("disable-redirects", false, "")
	proxyAddr          = flag.String("x", "", "")
)

var usage = `用法：hey [选项...] <网址>

选项：
  -n  要执行的请求次数，默认 200。
  -c  并发工作线程数。请求总数不能小于并发数，默认 50。
  -q  每个工作线程的每秒请求数（QPS）限速，默认不限速。
  -z  持续压测时间。达到时间后停止并退出；指定此项时忽略 -n。
      示例：-z 10s 或 -z 3m。
  -o  输出格式。未指定时打印中文汇总。
      目前支持的替代格式为 "csv"，以逗号分隔值格式输出各次响应指标。

  -m  HTTP 方法，可选 GET、POST、PUT、DELETE、HEAD、OPTIONS。
      默认为 GET；指定 -f 时默认使用文件中的方法。
  -H  自定义 HTTP 请求头。可重复指定，例如：
      -H "Accept: text/html" -H "Content-Type: application/xml"
  -t  每个请求的超时时间（秒），默认 20，0 表示无限超时。
  -A  HTTP Accept 请求头。
  -d  HTTP 请求正文。
  -D  从文件读取 HTTP 请求正文，例如 /home/user/file.txt 或 ./file.txt。
  -f  从 .http 文件读取请求（方法、URL、请求头、正文），
      文件格式兼容 JetBrains/VS Code REST Client。
      文件中必须有且仅有一个请求，否则报错退出。
      命令行显式指定的 -m/-H/-T/-d/-D 会覆盖文件中的对应内容；
      提供 -f 时 URL 位置参数可省略（使用文件中的 URL）。
  -T  Content-Type，默认为 text/html；-f 文件中已含
      Content-Type 且未显式指定 -T 时，以文件为准。
  -U  User-Agent，默认为 hey/0.0.1。
  -a  基本认证，格式为 用户名:密码。
  -x  HTTP 代理地址，格式为 host:port。
  -h2 启用 HTTP/2。

  -host HTTP Host 请求头。

  -disable-compression  禁用压缩。
  -disable-keepalive    禁用 keep-alive，不复用不同请求的 TCP 连接。
  -disable-redirects    禁止跟随 HTTP 重定向。
  -cpus                 使用的 CPU 核心数。
                        （当前机器默认为 %d 核）
`

func main() {
	flag.Usage = func() {
		fmt.Fprint(os.Stderr, fmt.Sprintf(usage, runtime.NumCPU()))
	}

	var hs headerSlice
	flag.Var(&hs, "H", "")

	flag.Parse()
	if flag.NArg() < 1 && *httpFile == "" {
		usageAndExit("")
	}

	runtime.GOMAXPROCS(*cpus)
	num := *n
	conc := *c
	q := *q
	dur := *z

	if dur > 0 {
		num = math.MaxInt32
		if conc <= 0 {
			usageAndExit("-c 不能小于 1。")
		}
	} else {
		if num <= 0 || conc <= 0 {
			usageAndExit("-n 和 -c 不能小于 1。")
		}

		if num < conc {
			usageAndExit("-n 不能小于 -c。")
		}
	}

	url := ""
	if flag.NArg() >= 1 {
		url = flag.Args()[0]
	}

	var fh *httpFileRequest
	if *httpFile != "" {
		var err error
		fh, err = parseHTTPFile(*httpFile)
		if err != nil {
			errAndExit(err.Error())
		}
		if url == "" {
			url = fh.URL
		}
	}

	method := strings.ToUpper(*m)
	if method == "" {
		if fh != nil {
			method = fh.Method
		} else {
			method = "GET"
		}
	}

	header := make(http.Header)
	if fh != nil {
		header = fh.Headers
	}
	ct := *contentType
	if ct == "" && header.Get("Content-Type") == "" {
		ct = "text/html"
	}
	if ct != "" {
		header.Set("Content-Type", ct)
	}
	if *headers != "" {
		usageAndExit("参数 '-h' 已弃用，请改用 '-H'。")
	}
	for _, h := range hs {
		match, err := parseInputWithRegexp(h, headerRegexp)
		if err != nil {
			usageAndExit(err.Error())
		}
		header.Set(match[1], match[2])
	}

	if *accept != "" {
		header.Set("Accept", *accept)
	}

	var username, password string
	if *authHeader != "" {
		match, err := parseInputWithRegexp(*authHeader, authRegexp)
		if err != nil {
			usageAndExit(err.Error())
		}
		username, password = match[1], match[2]
	}

	var bodyAll []byte
	if *body != "" {
		bodyAll = []byte(*body)
	}
	if *bodyFile != "" {
		slurp, err := ioutil.ReadFile(*bodyFile)
		if err != nil {
			errAndExit(err.Error())
		}
		bodyAll = slurp
	}
	if bodyAll == nil && fh != nil {
		bodyAll = fh.Body
	}

	var proxyURL *gourl.URL
	if *proxyAddr != "" {
		var err error
		proxyURL, err = gourl.Parse(*proxyAddr)
		if err != nil {
			usageAndExit(err.Error())
		}
	}

	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		usageAndExit(err.Error())
	}
	req.ContentLength = int64(len(bodyAll))
	if username != "" || password != "" {
		req.SetBasicAuth(username, password)
	}

	if *hostHeader != "" {
		req.Host = *hostHeader
	}

	ua := header.Get("User-Agent")
	if ua == "" {
		ua = heyUA
	} else {
		ua += " " + heyUA
	}
	header.Set("User-Agent", ua)

	if *userAgent != "" {
		ua = *userAgent + " " + heyUA
		header.Set("User-Agent", ua)
	}

	req.Header = header

	w := &requester.Work{
		Request:            req,
		RequestBody:        bodyAll,
		N:                  num,
		C:                  conc,
		QPS:                q,
		Timeout:            *t,
		DisableCompression: *disableCompression,
		DisableKeepAlives:  *disableKeepAlives,
		DisableRedirects:   *disableRedirects,
		H2:                 *h2,
		ProxyAddr:          proxyURL,
		Output:             *output,
	}
	w.Init()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		<-c
		w.Stop()
	}()
	if dur > 0 {
		go func() {
			time.Sleep(dur)
			w.Stop()
		}()
	}
	w.Run()
}

func errAndExit(msg string) {
	fmt.Fprintf(os.Stderr, "%s", msg)
	fmt.Fprintf(os.Stderr, "\n")
	os.Exit(1)
}

func usageAndExit(msg string) {
	if msg != "" {
		fmt.Fprintf(os.Stderr, "%s", msg)
		fmt.Fprintf(os.Stderr, "\n\n")
	}
	flag.Usage()
	fmt.Fprintf(os.Stderr, "\n")
	os.Exit(1)
}

func parseInputWithRegexp(input, regx string) ([]string, error) {
	re := regexp.MustCompile(regx)
	matches := re.FindStringSubmatch(input)
	if len(matches) < 1 {
		return nil, fmt.Errorf("无法解析提供的参数；参数 = %v", input)
	}
	return matches, nil
}

type headerSlice []string

func (h *headerSlice) String() string {
	return fmt.Sprintf("%s", *h)
}

func (h *headerSlice) Set(value string) error {
	*h = append(*h, value)
	return nil
}
