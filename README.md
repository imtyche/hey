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
  -f  从 .http 文件读取请求（方法、URL、请求头、正文）。
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

### 从文件读取 HTTP 请求正文

使用 `-D` 参数可以从文件读取请求正文，适合正文内容较长或包含换行、引号等不便在命令行中直接书写的场景（如 JSON、XML、表单模板文件）。

- `-D` 接收文件路径，支持绝对路径（如 `/home/user/body.json`）和相对路径（如 `./body.json`）。
- 文件内容会整体作为请求正文，每个请求都会发送相同的内容。
- 需配合 `-m POST`（或 PUT 等方法）使用，并用 `-T` 指定正确的 Content-Type；`-T` 默认为 `text/html`。
- 若同时指定 `-d` 和 `-D`，`-D` 的文件内容优先生效。
- 文件不存在或不可读时会报错退出。

示例：

准备正文文件 `payload.json`：

```json
{
  "name": "hey",
  "type": "loadtest",
  "items": [1, 2, 3]
}
```

以 JSON 正文发起 POST 压测：

```bash
hey -m POST -T application/json -D ./payload.json -n 1000 -c 100 https://example.com/api
```

以表单文件正文发起 POST 压测：

```bash
hey -m POST -T "application/x-www-form-urlencoded" -D ./form.txt https://example.com/submit
```

以 XML 正文发起 PUT 压测：

```bash
hey -m PUT -T application/xml -D /home/user/data.xml https://example.com/api/resource
```

### 从 .http 文件读取请求

使用 `-f` 参数可以从 `.http` 文件读取完整的请求定义（方法、URL、请求头、正文），文件格式兼容 JetBrains HTTP Client / VS Code REST Client。

`payload.http` 示例：

```http
### 创建用户
POST https://example.com/api/users HTTP/1.1
Content-Type: application/json
X-Trace: abc123

{
  "name": "hey",
  "items": [1, 2, 3]
}
```

发起压测（URL 可省略，默认取文件中的 URL）：

```bash
hey -f ./payload.http -n 1000 -c 100
```

规则说明：

- 文件中**必须有且仅有一个请求**；包含多个请求块（以 `###` 分隔）时会报错退出：`只允许包含一个请求，但文件中包含 N 个`。
- `#` 或 `//` 开头的行为注释。
- 命令行显式指定的 `-m`、`-H`、`-T`、`-d`、`-D` 会覆盖文件中的对应内容；URL 位置参数若提供，也会覆盖文件中的 URL。
- 文件中已包含 `Content-Type` 且未显式指定 `-T` 时，以文件中的值为准。

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
