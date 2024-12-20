---
id: conf
---
# 配置

配置是工程化程序的重要组件, 默认选择采用`ymal`格式,并且对文件型配置支持较为友好.

## 使用方法

```go
import (
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/knadh/koanf/providers/s3"
)

func main() {
	// 以程序运行目录为根目录,默认指向的`etc/app.yaml`
	conf.New()
	// 以appdir目录为根目录的配置,默认指向的`etc/app.yaml`
	conf.New(conf.WithBaseDir("appdir"))
	// 以指定配置文件路径
	conf.New(conf.WithLocalPath("xxx.yaml"))
	// 在默认配置文件并加载其他配置文件
	conf.New(conf.WithIncludeFiles("1.yaml","2.yaml"))
	// 还支持bytes[],map[string]any为数据源的初始化方法
	// conf.NewFromBytes,conf.NewFromStringMap
	// 扩展初始化方法,如使用koanf s3 provider库
	s3config := s3.Config{
		AccessKey: "xxx",
		Secret:   "xxx",
		Bucket:   "bucket",
		Region:   "us-east-1",
		Endpoint: "https://s3.amazonaws.com",
		ObjectKey: "app.yaml",
    }   
	parse := conf.NewParserFromProvider(s3.Provider(s3config))
	conf.NewFromParse()
}

```

> 关于更多的Provider,可参考[Koanf Provider](https://github.com/knadh/koanf?tab=readme-ov-file#bundled-providers)

我们结合通过快速开始创建的项目, 其配置文件大至如下:

```yaml
namespace: default
appName: example
version: 0.0.1
development: true
#... 略
```

>  默认采用程序所在目录为根目录(非当前执行目录),因此需要注意配置文件中相对路径的位置

安装woco后,首先通过无配置快速开始一个woocoo项目.

```go
package main

import (
	"github.com/gin-gonic/gin"
	"github.com/tsingsun/woocoo"
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/web"
	"log"
)

func main() {
	// 构建一个空程序配置
	cnf := conf.New()
	// 将配置传入woocoo.New()函数
	app := woocoo.New(WithAppConfiguration(cnf))
	webSrv := web.New()
	app.RegisterServer(webSrv)
	webSrv.Router().GET("/", func(c *gin.Context) {
		c.String(200, "hello world")
	})
	if err := app.Run(); err != nil {
		log.Panic(err)
	}
}
```

`go run main.go`后`curl localhost:8080`即可看到`hello world`输出.

当然现实工程不会有这么简单的项目.实际上woocoo是一个基于约定配置的框架.
一般我们直接通过以下代码初始化程序就可以,会自动加载配置文件,加载工程过程中所需要的各类型组件.

```go
app := woocoo.New()
```

配置文件内主要区分主程序项配置及组件配置,对于配置和使用是开放式的,只需要依照约定符合基本的组件初始化配置即可.

在制定配置文件时,请注意以下几点:

1. 配置文件针对Map的读取是无序的,这是底层库采用了golang Map.因此如果你的配置是顺序有关的(比如中间件有执行顺序要求),请使用数组.
2. 其他情况建议使用Map会更方便.

## 主程序项

```yaml
# 程序命名空间,在需要命名空间定义的组件,将默认使用此命名空间
namespace: default

# 程序名称
appName: example

# 程序版本
version: 0.0.1

# 是否开发模式,开发模式下,对程序开发更友好.一般生产环境下设定为false
development: true

# 附加配置文件,可以通过此配置项,加载其他配置文件,如:etc/app.dev.yaml 做为本地开发配置文件.
# 附加配置文件将会合并主配置文件中的配置项.但合并有一定的限制.
#   1. Map,Struct类型可正常合并.同Key将会被最后加载的配置文件覆盖.
#   2. Slice类型,将会被最后加载的配置文件覆盖.
#   3. 相对路径以程序运行目录为基准.
includeFiles:
  - etc/app.dev.yaml
  - etc/local.yaml
  - etc/web.yaml
```

## 组件配置

组件配置也比较简单,根据组件的约定确定好组件根路径,按组件约定进行展开.如Web组件,极简配置如下:

```yaml
#... 主程序项配置略
web:
  server:
    addr: :8080
```

各组件的配置将在对应组件文档中说明.

##  嵌套结构

当要为某个组件配置时,经常到遇自定义配置类,嵌套底层组件配置.

```yaml
component:
  addr: :8080
  # 以下底层组件配置
  name: base  
```

```go
type Component struct {
    BaseComponentConfig `,inline`
    Addr string `json:"addr"`
}

var cfg Component
conf.Unmarshal(&cfg)
```
> 通过`inline`标签时,不支持指针类型,需要注意.

## 变量

woocoo支持环境变量配置,在配置文件中使用`${ENV_NAME}`即可引用环境变量.

- 可以在`etc`目录下创建`.env`及`.env.local`文件,程序启动时会自动加载对应的环境变量文件;
- 支持系统环境变量;
- 在配置文件加载前,通过`os.SetEnv`设置的环境变量;

```yaml
web:
  server:
    addr: :${PORT}
```

有时我们会在配置文件中引用其他配置项,这时可以使用YAML自身的引用功能.

```yaml
jwt: &jwt
  secret: secret
  expire: 3600

some:
  jwt:
    <<: *jwt
```

## 最佳实践

### 配置文件规范

- 数组时使用顶格缩进l,有利于阅读.
  ```yaml
    web:
      middleware:
      - cors
  ```