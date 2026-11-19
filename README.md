![hey](http://i.imgur.com/szzD9q0.png)

hey 是一个向 Web 应用发送负载的小工具。

hey 原名 boom，受 Tarek Ziade 的 [boom](https://github.com/tarekziade/boom) 工具启发。由于与原项目重名导致二进制名称冲突，为保留原项目名称，本项目更名为 hey。

## 安装

- Linux (amd64)：https://storage.googleapis.com/hey-releases/hey_linux_amd64
- macOS (amd64)：https://storage.googleapis.com/hey-releases/hey_darwin_amd64
- Windows (amd64)：https://storage.googleapis.com/hey-releases/hey_windows_amd64

### 包管理器

macOS 用户可通过 Homebrew 安装：

- `brew install hey`

## 用法

hey 会以指定的并发级别运行指定数量的请求，并输出中文统计结果。

同时支持 HTTP/2 端点。

```text
用法：hey [选项...] <网址>

选项：
  -n  要执行的请求次数，默认 200。
  -c  并发工作线程数，默认 50。
  -q  每个工作线程的每秒请求数（QPS）限速。
  -z  持续压测时间；指定后忽略 -n。
  -o  输出格式，默认为中文汇总；可使用 csv。

  -m  HTTP 方法。
  -H  自定义 HTTP 请求头，可重复指定。
  -t  每个请求的超时时间（秒），默认 20，0 表示无限超时。
  -A  HTTP Accept 请求头。
  -d  HTTP 请求正文。
  -D  从文件读取 HTTP 请求正文。
  -T  Content-Type，默认为 text/html。
  -U  User-Agent。
  -a  基本认证，格式为 用户名:密码。
  -x  HTTP 代理地址。
  -h2 启用 HTTP/2。
  -host HTTP Host 请求头。
  -disable-compression  禁用压缩。
  -disable-keepalive    禁用 keep-alive。
  -disable-redirects    禁止跟随重定向。
  -cpus                 使用的 CPU 核心数。
```

## 示例

使用默认设置发起请求：

```bash
hey https://google.com
```

发起 1000 次请求，100 个并发：

```bash
hey -n 1000 -c 100 https://google.com
```

持续压测 30 秒：

```bash
hey -z 30s https://google.com
```

发送自定义 POST 请求：

```bash
hey -m POST -d "param1=value1&param2=value2" https://google.com
```

添加自定义请求头：

```bash
hey -H "Accept: application/json" -H "Authorization: Bearer token" https://google.com
```

测试 HTTP/2：

```bash
hey -h2 https://google.com
```

每个工作线程限制为每秒 10 个请求：

```bash
hey -q 10 -c 5 -z 30s https://google.com
```
