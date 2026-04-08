# MemPalace-Go

> 本项目是 [MemPalace](https://github.com/milla-jovovich/mempalace) 的 Go 语言重写版本

MemPalace 是一个本地 AI 记忆系统，采用"宫殿"（Palace）作为核心隐喻，将记忆存储在 Wings（翼）→ Rooms（房间）→ Drawers（抽屉）的层级结构中，通过语义搜索实现快速检索。

## 特性

- 🏛️ **宫殿式存储结构**：Wing（项目/人物）→ Room（主题）→ Drawer（文本块）
- 🔍 **语义搜索**：基于向量相似度的智能检索
- 📚 **四层记忆栈**：L0（身份）→ L1（核心故事）→ L2（按需）→ L3（深度搜索）
- 🔗 **MCP 服务器**：支持 19 个工具的 Model Context Protocol 实现
- 📦 **Go SDK**：提供公共 API，可直接集成到 Go 项目
- 🗃️ **SQLite 向量存储**：无需外部数据库，内置 FTS5 全文搜索
- 🎯 **AAAK 压缩方言**：实体识别与压缩编码

## 安装

```bash
# 克隆仓库
git clone https://github.com/kinwyb/mempalace-go.git
cd mempalace-go

# 构建
go build -o mempalace ./cmd/mempalace

# 安装到 GOPATH
go install ./cmd/mempalace
```

## 快速开始

### 初始化配置

```bash
# 首次运行设置（交互式配置）
mempalace setup

# 或使用默认配置
mempalace status
```

### 挖掘项目文件

```bash
# 挖掘项目目录
mempalace mine /path/to/project

# 挖掘对话导出文件
mempalace mine /path/to/conversations --mode convos

# 预览模式（不实际存储）
mempalace mine /path/to/project --dry-run
```

### 搜索内容

```bash
# 基本搜索
mempalace search "golang 并发编程"

# 按翼过滤
mempalace search "api 设计" --wing myproject

# 按房间过滤
mempalace search "错误处理" --room technical
```

### 查看状态

```bash
# 显示宫殿状态
mempalace status

# 显示唤醒上下文（L0 + L1）
mempalace wake-up
```

### 启动 MCP 服务器

```bash
# 启动 MCP 服务器（用于 AI 工具集成）
mempalace mcp
```

## 命令参考

| 命令 | 描述 |
|------|------|
| `mempalace init <dir>` | 检测目录结构，生成房间配置 |
| `mempalace mine <dir>` | 挖掘项目文件到宫殿 |
| `mempalace mine <dir> --mode convos` | 挖掘对话导出文件 |
| `mempalace search <query>` | 语义搜索 |
| `mempalace status` | 显示宫殿统计信息 |
| `mempalace wake-up` | 显示 L0 + L1 唤醒上下文 |
| `mempalace mcp` | 启动 MCP 服务器 |
| `mempalace split <file>` | 拆分大型对话文件 |
| `mempalace compress` | AAAK 压缩 |
| `mempalace setup` | 首次运行配置 |

## MCP 工具

MCP 服务器提供以下 19 个工具：

| 工具 | 描述 |
|------|------|
| `search` | 搜索记忆宫殿 |
| `wake_up` | 获取 L0 + L1 唤醒上下文 |
| `add_drawer` | 添加新抽屉 |
| `get_status` | 获取宫殿状态 |
| `list_wings` | 列出所有翼 |
| `list_rooms` | 列出翼下的房间 |
| `check_duplicate` | 检查重复内容 |
| `get_layer_content` | 获取特定层内容 |
| `delete_drawer` | 删除抽屉 |
| `compress_drawer` | 压缩抽屉 |
| `register_entity` | 注册实体 |
| `detect_entities` | 检测实体 |
| `get_taxonomy` | 获取分类体系 |
| `mine_file` | 挖掘单个文件 |
| `batch_add` | 批量添加抽屉 |
| `get_recent` | 获取最近添加的内容 |
| `get_drawer` | 获取特定抽屉 |
| `update_drawer` | 更新抽屉 |
| `detect_room` | 检测内容所属房间 |
| `store_layer` | 存储到特定记忆层 |

## Go SDK 集成

MemPalace 提供了公共 Go API，允许其他 Go 项目直接集成而无需通过 MCP 协议。

### 安装

```bash
go get github.com/kinwyb/mempalace-go/pkg/mempalace
```

### 快速开始

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/kinwyb/mempalace-go/pkg/mempalace"
)

func main() {
    ctx := context.Background()

    // 创建 Palace 实例
    palace, err := mempalace.New(ctx,
        mempalace.WithOllama("http://localhost:11434", "nomic-embed-text"),
        mempalace.WithPalacePath("~/.mempalace/palace"),
    )
    if err != nil {
        log.Fatal(err)
    }
    defer palace.Close()

    // 搜索内容
    result, err := palace.Search(ctx, "数据库连接错误",
        mempalace.WithWing("myproject"),
        mempalace.WithLimit(10),
    )
    if err != nil {
        log.Fatal(err)
    }

    for _, item := range result.Results {
        fmt.Printf("[%s/%s] %s\n", item.Wing, item.Room, item.Content[:100])
    }
}
```

### 配置选项

```go
// Ollama 嵌入模型
mempalace.WithOllama("http://localhost:11434", "nomic-embed-text")

// OpenAI 嵌入模型
mempalace.WithOpenAI("sk-...", "", "text-embedding-3-small")

// 自定义存储路径
mempalace.WithPalacePath("/data/my-palace")

// 从配置文件加载
mempalace.WithConfigFile("~/.mempalace/config.yaml")

// 文本分块参数
mempalace.WithChunkSize(1000, 200, 100)

// 搜索默认参数
mempalace.WithSearchDefaults(20, 0.85)

// 自定义嵌入器
mempalace.WithEmbedder(myCustomEmbedder)
```

### 核心操作

#### 添加内容

```go
// 添加单个内容
result, err := palace.Add(ctx, "重要决策：使用 PostgreSQL 作为主数据库",
    mempalace.WithWingForAdd("myproject"),
    mempalace.WithRoomForAdd("decisions"),
    mempalace.WithMetadata(map[string]any{
        "priority": "high",
        "date":     "2024-01-15",
    }),
)

// 添加文档
doc := mempalace.Document{
    Content: "API 端点设计文档",
    Wing:    "myproject",
    Room:    "api",
    Source:  "docs/api.md",
}
result, err := palace.AddDocument(ctx, doc)

// 批量添加
docs := []mempalace.Document{...}
results, err := palace.AddBatch(ctx, docs)
```

#### 搜索内容

```go
// 基本搜索
result, err := palace.Search(ctx, "用户认证",
    mempalace.WithLimit(10),
)

// 按翼/房间过滤
result, err := palace.Search(ctx, "数据库配置",
    mempalace.WithWing("myproject"),
    mempalace.WithRoom("config"),
    mempalace.WithLimit(5),
)

// 检查重复
dupResult, err := palace.CheckDuplicate(ctx, "内容...", 0.9)
if dupResult.IsDuplicate {
    fmt.Println("内容已存在")
}
```

#### 四层记忆栈

```go
// 存储到 L0（身份层）
err := palace.StoreInLayer(ctx, mempalace.L0, 
    "我是一名 Go 开发者，喜欢简洁的架构设计",
    mempalace.WithWingForLayer("identity"),
)

// 存储到 L1（核心故事层）
err := palace.StoreInLayer(ctx, mempalace.L1,
    "当前正在开发用户认证模块",
    mempalace.WithWingForLayer("myproject"),
)

// 获取唤醒上下文（L0 + L1）
wakeUp, err := palace.WakeUp(ctx)
fmt.Println(wakeUp)

// 自动分类内容到合适的层级
layer := palace.AutoClassify("我目前正在处理...")
fmt.Printf("推荐层级: %s\n", layer)
```

#### 挖掘项目文件

```go
// 挖掘项目目录
result, err := palace.Mine(ctx, "/path/to/project",
    mempalace.WithWingOverride("myproject"),
)

// 挖掘对话文件
result, err := palace.MineConversations(ctx, "/path/to/exports",
    mempalace.WithWingOverride("conversations"),
    mempalace.WithExtractMode("exchange"),
)

// 预览模式（不实际存储）
result, err := palace.Mine(ctx, "/path/to/project",
    mempalace.WithDryRun(),
)
```

#### 统计和管理

```go
// 获取统计信息
stats, err := palace.GetStats(ctx)
fmt.Printf("文档数: %d, 翼数: %d\n", stats.TotalDocuments, stats.TotalWings)

// 获取所有翼
wings, err := palace.GetWings(ctx)

// 获取翼下的房间
rooms, err := palace.GetRooms(ctx, "myproject")

// 删除操作
err := palace.Delete(ctx, "document-id")
err := palace.DeleteByWing(ctx, "old-project")
err := palace.DeleteByRoom(ctx, "myproject", "deprecated")
```

### 层级常量

```go
const (
    L0 mempalace.Layer = 0  // 身份层 - 核心身份、关键偏好
    L1 mempalace.Layer = 1  // 核心故事层 - 项目上下文、当前目标
    L2 mempalace.Layer = 2  // 按需层 - 需要时检索
    L3 mempalace.Layer = 3  // 深度搜索层 - 全面搜索
)
```

### 错误处理

```go
result, err := palace.Search(ctx, "query")
if err != nil {
    if mempalace.Is(err, mempalace.ErrClosed) {
        // Palace 已关闭
    } else if mempalace.Is(err, mempalace.ErrSearch) {
        // 搜索错误
    }
    log.Fatal(err)
}
```

### 使用 Mock Embedder 测试

```go
import "github.com/kinwyb/mempalace-go/pkg/embedding"

// 创建测试用 Palace
palace, err := mempalace.New(ctx,
    mempalace.WithPalacePath(t.TempDir()),
    mempalace.WithEmbedder(embedding.NewMockEmbedder(768)),
)
```

## 项目结构

```
mempalace-go/
├── cmd/mempalace/          # CLI 入口
│   └── main.go
├── internal/
│   ├── config/             # 配置管理
│   ├── convominer/         # 对话挖掘
│   ├── dialect/            # AAAK 压缩方言
│   ├── entity/             # 实体检测
│   ├── kg/                 # 知识图谱
│   ├── layers/             # 四层记忆栈
│   ├── mcp/                # MCP 服务器
│   ├── miner/              # 项目文件挖掘
│   ├── normalize/          # 对话格式标准化
│   ├── onboarding/         # 首次运行引导
│   ├── palace/             # 宫殿图遍历
│   ├── searcher/           # 语义搜索
│   └── split/              # 文件拆分
├── pkg/
│   ├── embedding/          # 嵌入模型接口
│   ├── mempalace/          # 公共 Go SDK
│   └── vector/             # 向量存储接口
├── go.mod
└── README.md
```

## 四层记忆栈

| 层级 | 名称 | 描述 | 用途 |
|------|------|------|------|
| L0 | 身份层 | 核心身份、关键偏好 | 始终激活 |
| L1 | 核心故事 | 项目上下文、当前目标 | 上下文窗口 |
| L2 | 按需层 | 需要时检索 | 搜索触发 |
| L3 | 深度搜索 | 全面搜索完整上下文 | 深度分析 |

## 配置

配置文件位于 `~/.mempalace/config.yaml`，支持以下选项：

```yaml
# 宫殿存储路径
palace_path: ~/.mempalace/palace

# 嵌入模型配置
embedding_model: nomic-embed-text
ollama_host: http://localhost:11434

# 文本处理
chunk_size: 800
chunk_overlap: 100
min_chunk_size: 50

# 搜索配置
search_limit: 10
similarity_threshold: 0.9

# 日志级别
log_level: info
```

### 环境变量

配置可通过环境变量覆盖：

```bash
MEMPALACE_PALACE_PATH=/custom/path
MEMPALACE_EMBEDDING_MODEL=text-embedding-3-small
MEMPALACE_OLLAMA_HOST=http://localhost:11434
MEMPALACE_LOG_LEVEL=debug
```

## 项目配置文件

在项目目录中创建 `mempalace.yaml` 来自定义翼和房间：

```yaml
wing: my-project

rooms:
  - name: api
    description: API 相关代码
    keywords:
      - endpoint
      - handler
      - route
  
  - name: database
    description: 数据库相关
    keywords:
      - sql
      - query
      - migration
  
  - name: tests
    description: 测试文件
    keywords:
      - test
      - spec
      - mock
```

## 运行测试

```bash
# 运行所有测试
go test ./...

# 运行特定包的测试
go test ./internal/miner/...

# 带覆盖率
go test -cover ./...
```

## 与原版 Python 实现的差异

| 特性 | Python 版本 | Go 版本 |
|------|------------|---------|
| 向量数据库 | ChromaDB | SQLite + FTS5 |
| 日志 | Python logging | Go slog |
| CLI 框架 | Click | Cobra |
| 配置格式 | YAML | YAML |
| MCP 工具数 | 19 | 19 |
| Go SDK | 无 | ✅ 完整支持 |

## 依赖

- [github.com/spf13/cobra](https://github.com/spf13/cobra) - CLI 框架
- [modernc.org/sqlite](https://pkg.go.dev/modernc.org/sqlite) - 纯 Go SQLite 驱动
- [gopkg.in/yaml.v3](https://gopkg.in/yaml.v3) - YAML 解析

## 开发

```bash
# 格式化代码
go fmt ./...

# 静态检查
go vet ./...

# 构建
go build -o mempalace ./cmd/mempalace
```

## 许可证

MIT License

## 致谢

本项目是对 [MemPalace](https://github.com/milla-jovovich/mempalace) 的 Go 语言重写实现。