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
    # 可选：获取 Token 时的自定义 Header
    tokenHeader:
      X-API-Key: "your-api-key"
```

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

### 完整配置示例

```yaml
http:
  timeout: 30s
  # 基础认证（与 OAuth2 互斥）
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
      grant_type: "password"
    tokenHeader:
      X-API-Key: "platform-api-key"
    storeKey: "redis"
  
  # 自定义 Token 请求 Header（仅与 OAuth2 配合使用）
  authorization:
    headerName: "Authorization"
    headerPrefix: "Bearer"
```

## 代码使用

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

## 特性

- ✅ 支持 Client Credentials 授权模式
- ✅ 支持 Password 授权模式
- ✅ 自动 Token 管理和刷新
- ✅ 支持 Token 缓存（Redis）
- ✅ 支持获取 Token 时的自定义 Header
- ✅ 支持 API 请求的自定义 Token Header
- ✅ 支持 HTTPS/TLS 配置
- ✅ 支持代理配置
- ✅ 支持请求超时
