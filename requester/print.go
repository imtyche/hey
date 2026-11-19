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

/*
Hey supports two output formats: summary and CSV

The summary output presents a number of statistics about the requests in a
human-readable format, including:
- general statistics: requests/second, total runtime, and average, fastest, and slowest requests.
- a response time histogram.
- a percentile latency distribution.
- statistics (average, fastest, slowest) on the stages of the requests.

The comma-separated CSV format is proceeded by a header, and consists of the following columns:
1. response-time:	Total time taken for request (in seconds)
2. DNS+dialup:		Time taken to establish the TCP connection (in seconds)
3. DNS:				Time taken to do the DNS lookup (in seconds)
4. Request-write:	Time taken to write full request (in seconds)
5. Response-delay: 	Time taken to first byte received (in seconds)
6. Response-read:	Time taken to read full response (in seconds)
7. status-code:		HTTP status code of the response (e.g. 200)
8. offset:			The time since the start of the benchmark when the request was started. (in seconds)
*/
package requester

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"text/template"
)

func newTemplate(output string) *template.Template {
	outputTmpl := output
	switch outputTmpl {
	case "":
		outputTmpl = defaultTmpl
	case "csv":
		outputTmpl = csvTmpl
	}
	return template.Must(template.New("tmpl").Funcs(tmplFuncMap).Parse(outputTmpl))
}

var tmplFuncMap = template.FuncMap{
	"formatNumber":    formatNumber,
	"formatNumberInt": formatNumberInt,
	"histogram":       histogram,
	"jsonify":         jsonify,
}

func jsonify(v interface{}) string {
	d, _ := json.Marshal(v)
	return string(d)
}

func formatNumber(duration float64) string {
	return fmt.Sprintf("%4.4f", duration)
}

func formatNumberInt(duration int) string {
	return fmt.Sprintf("%d", duration)
}

func histogram(buckets []Bucket) string {
	max := 0
	for _, b := range buckets {
		if v := b.Count; v > max {
			max = v
		}
	}
	res := new(bytes.Buffer)
	for i := 0; i < len(buckets); i++ {
		var barLen int
		if max > 0 {
			barLen = (buckets[i].Count*40 + max/2) / max
		}
		res.WriteString(fmt.Sprintf("  %4.3f [%v]\t|%v\n", buckets[i].Mark, buckets[i].Count, strings.Repeat(barChar, barLen)))
	}
	return res.String()
}

var (
	defaultTmpl = `
汇总：
  总耗时：\t{{ formatNumber .Total.Seconds }} 秒
  最慢：\t{{ formatNumber .Slowest }} 秒
  最快：\t{{ formatNumber .Fastest }} 秒
  平均：\t{{ formatNumber .Average }} 秒
  每秒请求数：\t{{ formatNumber .Rps }}
  {{ if gt .SizeTotal 0 }}
  总数据量：\t{{ .SizeTotal }} 字节
  平均每请求：\t{{ .SizeReq }} 字节{{ end }}

响应时间直方图：
{{ histogram .Histogram }}

延迟分布：{{ range .LatencyDistribution }}
  {{ .Percentage }}%% 的请求在 {{ formatNumber .Latency }} 秒内{{ end }}

详细信息（平均、最快、最慢）：
  DNS+连接：\t{{ formatNumber .AvgConn }} 秒，{{ formatNumber .ConnMax }} 秒，{{ formatNumber .ConnMin }} 秒
  DNS 查询：\t{{ formatNumber .AvgDNS }} 秒，{{ formatNumber .DnsMax }} 秒，{{ formatNumber .DnsMin }} 秒
  请求写入：\t{{ formatNumber .AvgReq }} 秒，{{ formatNumber .ReqMax }} 秒，{{ formatNumber .ReqMin }} 秒
  等待响应：\t{{ formatNumber .AvgDelay }} 秒，{{ formatNumber .DelayMax }} 秒，{{ formatNumber .DelayMin }} 秒
  响应读取：\t{{ formatNumber .AvgRes }} 秒，{{ formatNumber .ResMax }} 秒，{{ formatNumber .ResMin }} 秒

状态码分布：{{ range $code, $num := .StatusCodeDist }}
  [{{ $code }}]\t{{ $num }} 个响应{{ end }}

{{ if gt (len .ErrorDist) 0 }}错误分布：{{ range $err, $num := .ErrorDist }}
  [{{ $num }}]\t{{ $err }}{{ end }}{{ end }}
`
	csvTmpl = `{{ $connLats := .ConnLats }}{{ $dnsLats := .DnsLats }}{{ $dnsLats := .DnsLats }}{{ $reqLats := .ReqLats }}{{ $delayLats := .DelayLats }}{{ $resLats := .ResLats }}{{ $statusCodeLats := .StatusCodes }}{{ $offsets := .Offsets }}{{ range $i, $v := .Lats }}{{ formatNumber $v }},{{ formatNumber (index $connLats $i) }},{{ formatNumber (index $dnsLats $i) }},{{ formatNumber (index $reqLats $i) }},{{ formatNumber (index $delayLats $i) }},{{ formatNumber (index $resLats $i) }},{{ index $statusCodeLats $i }},{{ formatNumber (index $offsets $i) }}
{{ end }}`
)
