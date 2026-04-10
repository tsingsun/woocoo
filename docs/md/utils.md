---
title: 工具包
---

# HTTP Client

内置了支持配置化的 HTTP Client，用于方便的进行 HTTP 服务的接入与请求。特别适应于一些开放平台的接入。

## 配置说明

### 基础配置

```yaml
http:
  timeout: 30s                    # 请求超时时间
  basicAuth:                      # HTTP Basic Auth (可选)
    username: "user"
    password: "pass"
```

### OAuth2 自动 Token 支持

支持 OAuth2 协议，自动获取和管理 Token，适用于开放平台 API 接入。

#### 1. Client Credentials 模式（默认）

```yaml
http:
  timeout: 30s
  oauth2:
    clientID: "your-client-id"
    clientSecret: "your-client-secret"
    scopes:
      - "read"
      - "write"
    endpoint:
      tokenUrl: "https://auth.example.com/oauth/token"
    # 可选：获取 Token 时的自定义 Header
    tokenHeader:
      X-API-Key: "your-api-key"
    # 可选：Token 缓存配置
    storeKey: "redis"
```

#### 2. Password 模式

```yaml
http:
  timeout: 30s
  oauth2:
    clientID: "your-client-id"
    clientSecret: "your-client-secret"
    scopes:
      - "read"
    endpoint:
      tokenUrl: "https://auth.example.com/oauth/token"
    endpointParams:
      grant_type: "password"      # 指定为 password 模式
      scope: "user:read"          # 可选：其他 token endpoint 参数
      audience: "https://api.example.com"  # 可选：目标 API 标识
    # 用户凭证（password 模式必需）
    username: "testuser"
    password: "testpass"
    # 可选：获取 Token 时的自定义 Header
    tokenHeader:
      X-API-Key: "your-api-key"
```

**说明：**
- `endpointParams` 用于传递 token endpoint 的额外参数，如 `grant_type`、`scope`、`audience` 等
- `username` 和 `password` 是独立字段，用于 password 模式的用户凭证
- `clientID` 和 `clientSecret` 通过 HTTP Basic Auth 发送
- `username` 和 `password` 作为表单参数发送

#### 3. 自定义 Token 请求 Header

默认情况下，OAuth2 Token 会通过 `Authorization: Bearer <token>` 发送。
某些 API 可能需要自定义 Header 名称或格式：

```yaml
http:
  timeout: 30s
  # 自定义 API 请求的认证 Header
  authorization:
    headerName: "X-Custom-Auth"   # 自定义 Header 名称（默认：Authorization）
    headerPrefix: "ApiKey"        # Token 前缀，会自动添加空格（默认：Bearer）
  oauth2:
    clientID: "your-client-id"
    clientSecret: "your-client-secret"
    endpoint:
      tokenUrl: "https://auth.example.com/oauth/token"
```

**示例效果：**
- 默认（不配置 authorization）：`Authorization: Bearer <token>`
- 自定义 Header：`X-Custom-Auth: <token>`
- 自定义前缀：`Authorization: ApiKey <token>`
- 空前缀（`headerPrefix: ""`）：`Authorization: <token>`

#### 4. 自定义 TokenSource

如果标准的 OAuth2 流程无法满足需求，可以实现自定义的 `TokenSource`：

```go
import (
    "context"
    "golang.org/x/oauth2"
    "github.com/tsingsun/woocoo/pkg/httpx"
)

// 自定义 TokenSource 实现
type customTokenSource struct {
    token *oauth2.Token
}

func (c *customTokenSource) Token() (*oauth2.Token, error) {
    // 自定义 token 获取逻辑
    return c.token, nil
}

// 方式 1: 使用 WithTokenSource Option
cfg, err := httpx.NewClientConfig(conf.New(), 
    httpx.WithTokenSource(&customTokenSource{
        token: &oauth2.Token{
            AccessToken: "my-custom-token",
            TokenType:   "Bearer",
        },
    }),
)
if err != nil {
    return err
}

client, err := cfg.Client(context.Background(), nil)
if err != nil {
    return err
}

// 方式 2: 配置创建后设置 TokenSource
cfg, err := httpx.NewClientConfig(conf.New())
if err != nil {
    return err
}

// 设置自定义 TokenSource
cfg.OAuth2.SetTokenSource(&customTokenSource{
    token: &oauth2.Token{
        AccessToken: "my-custom-token",
        TokenType:   "Bearer",
    },
})

client, err := cfg.Client(context.Background(), nil)
```

**适用场景：**
- 使用第三方提供的固定 Token
- 自定义 Token 刷新逻辑
- 集成其他认证系统（如 JWT、API Key 等）

#### 5. 处理包装的 Token 响应

某些 OAuth2 服务端会在标准 token 响应外再包装一层数据：

```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "access_token": "1234567890",
    "token_type": "Bearer",
    "expires_in": 3600
  }
}
```

参考示例代码实现自定义解析逻辑：

```go
import (
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "net/url"
    "golang.org/x/oauth2"
)

// 完整的自定义 TokenSource 实现
type wrappedTokenSource struct {
    clientID     string
    clientSecret string
    tokenURL     string
    username     string  // password grant 模式
    password     string  // password grant 模式
    httpClient   *http.Client
}

func (w *wrappedTokenSource) Token() (*oauth2.Token, error) {
    // Step 1: 准备请求
    form := url.Values{}
    form.Set("grant_type", "password")
    form.Set("client_id", w.clientID)
    form.Set("client_secret", w.clientSecret)
    form.Set("username", w.username)
    form.Set("password", w.password)

    req, _ := http.NewRequest("POST", w.tokenURL, strings.NewReader(form.Encode()))
    req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

    // Step 2: 发送请求
    resp, err := w.httpClient.Do(req)
    if err != nil {
        return nil, fmt.Errorf("request failed: %w", err)
    }
    defer resp.Body.Close()

    // Step 3: 读取响应
    body, _ := io.ReadAll(resp.Body)

    // Step 4: 解析包装的响应
    var wrapped struct {
        Code int             `json:"code"`
        Data json.RawMessage `json:"data"`
    }
    if err := json.Unmarshal(body, &wrapped); err == nil && wrapped.Data != nil {
        var token oauth2.Token
        json.Unmarshal(wrapped.Data, &token)
        return &token, nil
    }

    // 标准格式
    var token oauth2.Token
    json.Unmarshal(body, &token)
    return &token, nil
}
```

**说明：**
- 示例展示了完整的 HTTP 请求 → 解析响应 → 提取 token 流程
- 使用匿名结构体解析包装格式，避免暴露内部实现
- 同时兼容标准格式的响应

**Example 示例代码：**
- `Example_oauth2WrappedResponse`：演示完整的 OAuth2 包装响应处理流程
- `Example_customTokenSource`：演示完整的自定义 TokenSource 实现（含 HTTP 请求）
- `Example_extractWrappedToken`：演示简化的 token 解析函数

查看和运行示例：
```bash
# 查看所有 Example
go test -list Example ./pkg/httpx/...

# 运行特定 Example
go test -v -run Example_oauth2WrappedResponse ./pkg/httpx/...
go test -v -run Example_customTokenSource ./pkg/httpx/...
```

### 完整配置示例

```yaml
http:
  timeout: 30s
  # 基础认证（可与 OAuth2 共存，注意 Authorization header 冲突）
  basicAuth:
    username: "user"
    password: "pass"

  # 或 OAuth2 认证
  oauth2:
    clientID: "client-123"
    clientSecret: "secret-456"
    scopes:
      - "api:read"
      - "api:write"
    endpoint:
      tokenUrl: "https://open.platform.com/oauth/token"
    endpointParams:
      grant_type: "password"    # 可选：默认 client_credentials
      audience: "https://api.example.com"
    # password 模式必需的用户凭证
    username: "testuser"
    password: "testpass"
    tokenHeader:
      X-API-Key: "platform-api-key"
    storeKey: "redis"

  # 自定义 Token 请求 Header（仅与 OAuth2 配合使用）
  authorization:
    headerName: "Authorization"
    headerPrefix: "Bearer"
```

## 代码使用

### 基础用法

```go
import "github.com/tsingsun/woocoo/pkg/httpx"

// 从配置创建 Client
cfg, err := httpx.NewClientConfig(conf.New())
if err != nil {
    return err
}

client, err := cfg.Client(context.Background(), nil)
if err != nil {
    return err
}

// 使用 HTTP Client
resp, err := client.Get("https://api.example.com/data")
```

### 自定义 TokenSource

```go
import (
    "context"
    "golang.org/x/oauth2"
    "github.com/tsingsun/woocoo/pkg/httpx"
)

// 方式 1: 使用 WithTokenSource Option
cfg, err := httpx.NewClientConfig(conf.New(), 
    httpx.WithTokenSource(&customTokenSource{...}),
)

// 方式 2: 配置创建后设置
cfg, err := httpx.NewClientConfig(conf.New())
cfg.OAuth2.SetTokenSource(&customTokenSource{...})
client, err := cfg.Client(context.Background(), nil)
```

## 特性

- ✅ 支持 Client Credentials 授权模式
- ✅ 支持 Password 授权模式
- ✅ 支持 OAuth2 Grant Types 常量 (`GrantTypePassword`, `GrantTypeClientCredentials` 等)
- ✅ 自动 Token 管理和刷新
- ✅ 支持 Token 缓存（Redis）
- ✅ 支持获取 Token 时的自定义 Header
- ✅ 支持 API 请求的自定义 Token Header
- ✅ 支持自定义 TokenSource
- ✅ 提供完整示例代码：
  - `Example_oauth2WrappedResponse`：完整 OAuth2 包装响应处理
  - `Example_customTokenSource`：含 HTTP 请求的自定义 TokenSource
  - `Example_extractWrappedToken`：简化的 token 解析函数
- ✅ 提供 Go Example 示例 (可通过 `go test -list Example` 查看)
- ✅ 支持 HTTPS/TLS 配置
- ✅ 支持代理配置
- ✅ 支持请求超时
