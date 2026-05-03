# PRD: PaperPaper — 终端论文阅读 AI 助手

## 1. 产品概述

PaperPaper 是一个基于终端的交互式论文阅读工具，专为深度论文研读场景设计。用户发送论文，AI 首先生成详实总结供用户快速理解，随后进入多轮问答模式。系统采用精简的上下文管理策略——论文全文始终在上下文中，只保留最近 5 轮对话，从根源上避免长对话导致的幻觉问题。

## 2. 技术栈

| 层级 | 技术选择 | 理由 |
|:---|:---|:---|
| 语言 | Go (Golang) | 高性能、单二进制分发、原生并发模型适配流式处理 |
| TUI 框架 | Bubble Tea (charmbracelet) | Elm 架构，天然适配流式渲染与消息驱动 |
| UI 组件 | Bubbles (Viewport, Textarea, List, Spinner) | 开箱即用的终端组件库 |
| Markdown 渲染 | Glamour (charmbracelet) | 终端原生 Markdown 渲染，支持代码高亮与表格 |
| 数据存储 | JSON 文件 | 一个论文一个文件，零依赖，用户可直接查看/编辑/备份 |
| HTTP 客户端 | 标准库 net/http | 调用 OpenAI Chat Completion API |
| Prompt 管理 | `//go:embed` 内置默认值 | 开箱即用，用户可自定义覆盖 |
| Token 估算 | 字符数 / 4 | 轻量估算，无需额外依赖 |

## 3. 上下文管理策略

### 3.1 核心原则

**论文全文始终在上下文中，对话历史只保留最近 5 轮。** 这是整个架构的设计基石——问题不是 context 不够大，而是对话历史淹没了论文细节导致幻觉。

### 3.2 上下文组成（三层）

```
[System Prompt] → [论文全文] → [最近 5 轮完整对话] → [当前用户提问]
```

| 层级 | 内容 | 发送策略 |
|:---|:---|:---|
| L1: 论文全文 | 论文原始内容 | 每轮固定发送 |
| L2: 短期记忆 | 最近 5 轮完整对话 | 每轮动态拼装（基于当前剩余消息） |
| 当前提问 | 用户最新输入 | 每轮发送 |

### 3.3 摘要系统（UI 专用，不进模型上下文）

每轮对话结束后，后台调用轻量模型（如 GPT-4o-mini）生成摘要，**仅用于 UI 导航展示**：

- **摘要内容**：只总结用户提问（不总结 AI 回复），≤50 字
- **Session Title**：从论文前 100 字提取论文标题作为会话名称
- **失败策略**：重试 2-3 次，fallback 到截断的用户原文
- **异步执行**：不阻塞主对话流程，用户无感知

## 4. 状态机：两阶段 Prompt 策略

### 4.1 INIT 阶段（首次总结）

- **触发条件**: 用户首次提供论文内容（文件/粘贴）
- **System Prompt**: 使用配置的 `HEAVY_PROMPT`（长篇详实总结指令）
- **User Message**: 论文全文
- **输出处理**:
  - 流式渲染到 TUI 的 Viewport，供用户快速理解论文
  - 完成后完整内容存入 `papers.initial_summary` 字段
- **注意**: 初始总结**不进入后续对话的模型上下文**，论文全文已在 L1 中，模型可自行阅读原文

### 4.2 CHAT 阶段（问答）

- **触发条件**: 初始总结完成，用户开始提问
- **System Prompt**: 使用 `LIGHT_PROMPT`，内容为"根据论文内容回答用户问题"
- **上下文**: 论文全文 (L1) + 最近 5 轮对话 (L2) + 当前提问
- **注意**: 不包含初始总结，不包含摘要列表，不包含 Tool Calling

## 5. 数据存储设计

### 5.1 存储结构

```
~/.paperpaper/
├── config.yaml          # 配置文件
├── prompts/             # 可选，用户自定义 prompt 覆盖
│   ├── heavy.txt
│   ├── light.txt
│   └── digest.txt
└── papers/              # 会话数据（一个论文一个 JSON 文件）
    ├── 1.json
    ├── 2.json
    └── ...
```

### 5.2 论文 JSON 文件格式

```json
{
  "id": 1,
  "title": "Attention Is All You Need",
  "source_url": "",
  "content": "论文全文...",
  "initial_summary": "初始详实总结...",
  "model_used": "gpt-4o",
  "total_tokens_used": 12345,
  "created_at": "2026-05-02T10:00:00Z",
  "updated_at": "2026-05-02T10:30:00Z",
  "messages": [
    {
      "round_number": 0,
      "role": "user",
      "content": "用户提问...",
      "digest": "一句话摘要...",
      "token_count": 120,
      "created_at": "2026-05-02T10:05:00Z"
    },
    {
      "round_number": 0,
      "role": "assistant",
      "content": "AI 回复...",
      "token_count": 800,
      "created_at": "2026-05-02T10:05:30Z"
    }
  ]
}
```

### 5.3 删除逻辑

删除论文即删除对应的 JSON 文件。

## 6. 功能需求清单

### 6.1 论文输入

| 输入方式 | 实现要求 | 优先级 |
|:---|:---|:---|
| 命令行参数 | `paperpaper ./paper.txt` | P0 |
| 粘贴长文本 | 多行输入模式，Ctrl+D 或 Esc+Enter 结束 | P0 |
| URL 链接 | 解析 HTML 页面转 Markdown，失败回退至 latex | P1 |

**URL 解析器说明**: 用户已有的 Rust 实现可编译为独立子进程 arxiv2text，Go 主程序通过 `os/exec` 调用并获取 Stdout。P1 保持纯 Go，可重构为 `net/http` + `goquery/html2md` 实现，Rust 源代码位于 /Users/hanzerui/joyspace/zenflow/arxiv2text/src/。

### 6.2 交互命令系统

| 命令 | 功能 | 说明 |
|:---|:---|:---|
| `/new [url/path]` | 新建会话 | 重置上下文，载入新论文 |
| `/list` | 会话列表 | 弹出交互式列表，显示历史论文标题 |
| `/open [id/title]` | 加载历史会话 | 恢复论文上下文，加载最近 5 轮消息 |
| `/delete` | 删除当前会话 | 弹出确认框，确认后删除 JSON 文件 |
| `/summarize` | 对话元总结 | 触发当前论文全文总结（见 6.3） |
| `/export` | 导出到 Obsidian | 导出所有积累内容（见 6.4） |
| `/model [name]` | 模型切换 | 动态切换主模型 |
| `/config` | 配置管理 | 动态修改 API Key、BaseURL |
| `/edit` | 编辑最近问题 | 回填最近一条用户提问，修改后删除 AI 回复并重新生成 |
| `/del [round]` | 删除指定轮次 | 成对删除（问题+回答），L2 动态重组 |
| `/quit` 或 `Ctrl+C` | 退出 | 自动保存当前会话状态 |

### 6.3 对话元总结 (`/summarize`)

- **触发**: 用户手动输入 `/summarize`
- **输入内容**: `Initial Summary` + `All Chat History`（完整对话记录，不包含论文原文）
- **System Prompt**: 专门的 `DIGEST_PROMPT`，聚焦于：
  - 问答中讨论的深层细节
  - 用户追问的重点方向
  - 存在的疑点与争议
- **输出**: 结构化 Markdown，包含论文核心思想 + 对话挖掘出的深层内容
- **模型**: 主模型（与对话相同的模型）
- **Token 管理**: 预估总 token，如超过百万级别则提示用户或分段总结
- **不自动触发导出**，用户自行决定何时 `/export`

### 6.4 Obsidian 导出 (`/export`)

- **导出内容**: 所有积累内容，包括：
  - 初始总结（如果存在）
  - 对话摘要列表
  - `/summarize` 结果（如果已生成）
- **文件格式**: `{Title}_session.md`
- **存储路径**: 用户配置的 Obsidian Vault 目录
- **YAML Frontmatter**: 自动添加 `title`, `date`, `source_url`, `tags`（默认 `#paper #reading`）
- **模板支持**: 用户可自定义导出模板，变量包括 `{{Title}}`, `{{Date}}`, `{{Summary}}`, `{{QnA}}`

### 6.5 消息编辑与删除

#### 编辑最近问题 (`/edit` 或 UI 按钮)

- 最近一轮用户提问的右侧显示 `[✏️ Edit]` 按钮
- 触发后，底部 Textarea 回填最近一条 user message
- 用户修改后提交，删除最近一条 assistant message + 该轮 digest
- 基于当前 L2（剩余最近 5 轮）重新请求模型生成

#### 删除任意一轮 (`/del [round]` 或 UI 按钮)

- 每轮消息鼠标悬停时显示 `[🗑 Delete]` 按钮
- 成对删除：该轮的 user message + assistant message + digest
- 删除后弹出确认提示
- L2 自动从剩余消息中取最近 5 轮

#### 操作按钮交互

- **鼠标悬停**到某轮消息时显示操作按钮
- 按钮：`[📋 Copy]` / `[🗑 Delete]` / `[✏️ Edit]`（Edit 仅最近一轮）
- Copy 复制 AI 回复内容到剪贴板
- 同时保留斜杠命令作为键盘操作备选

### 6.6 Token 显示

- 每条消息下方显示预估 token 数
- AI 回复的 token 来自 API 返回的 `usage.completion_tokens`（精确值）
- 用户提问的 token 通过字符数 / 4 估算
- 底部状态栏显示当前论文累计 token 消耗

### 6.7 模式切换

- **Normal Mode**: 浏览模式，滚动查看历史内容，键盘 `j/k` 或鼠标滚动
- **Input Mode**: 输入模式，底部 Textarea 激活，支持多行输入
- 切换方式: `i` 进入 Input 模式，`Esc` 返回 Normal 模式（Vim 风格）
- **选择模式**: `/list` 进入，键盘 `j/k` 上下选择，`Enter` 确认，支持鼠标点击

### 6.8 流式渲染与中断处理

#### 流式渲染要求

- AI 响应通过 Bubble Tea 消息循环逐步推送
- 每个 chunk 到达后触发 `Update()` 更新 Model
- `View()` 实时通过 Glamour 渲染当前累积文本
- Viewport 自适应滚动到底部

#### 中断处理策略

- **网络/API 临时错误**: 自动重试 1 次
- **重试失败**: 保留已生成的部分，在末尾追加 `[生成中断]` 标记，存入 messages
- **用户主动打断** (Esc/Ctrl+C): 立即停止，保留已生成部分，存入 messages

## 7. UI 布局设计

```
┌─────────────────────────────────────────────────────────┐
│ [📄 Attention is All You Need]  [GPT-4o]  [🟢 Chat]     │  ← 顶部状态栏
├─────────────────────────────────────────────────────────┤
│                                                         │
│  ## 核心贡献摘要                                         │
│  - 提出了 Transformer 架构，完全基于注意力机制...          │  ← Viewport 区
│                                                         │  (Markdown 渲染)
│  > **你**: 多头注意力的计算复杂度是多少？ [📋] [🗑] [✏️]  │
│    [Tokens: ~120]                                        │
│  > **AI**: 多头注意力的计算复杂度为 O(n²d)...  [📋] [🗑]  │
│    [Tokens: 800]                                         │
│                                                         │
├─────────────────────────────────────────────────────────┤
│ > 请详细解释 Query, Key, Value 的物理意义...              │  ← 输入区
├─────────────────────────────────────────────────────────┤
│ [Tokens: 1,234] [Rounds: 12]                             │  ← 底部状态栏
└─────────────────────────────────────────────────────────┘
```

### 7.1 鼠标支持

- Bubble Tea 启用 `WithMouseCellMotion()`
- 鼠标滚轮滚动 Viewport
- 鼠标点击列表项进行会话选择
- 鼠标点击输入区激活编辑
- 鼠标悬停消息显示操作按钮（Copy/Delete/Edit）

## 8. 配置与首次启动

### 8.1 首次启动流程

1. 检查 `~/.paperpaper/config.yaml` 是否存在，存在则使用
2. 不存在，检查 `$OPENAI_API_KEY` 环境变量，有则用默认配置启动
3. 都没有，报错提示用户配置，并告知两种配置方式

### 8.2 配置文件格式 (config.yaml)

```yaml
api:
  base_url: "https://api.openai.com/v1"
  api_key: "${OPENAI_API_KEY}"
  default_model: "gpt-4o"
  light_model: "gpt-4o-mini"  # 用于摘要生成和标题提取

obsidian:
  vault_path: "~/Documents/Obsidian/MyVault"
  export_folder: "Papers"

ui:
  max_recent_rounds: 5    # L2 短期记忆轮数
```

### 8.3 API 兼容性

只支持 **OpenAI Chat Completion API** 格式。其他提供商（包括 Anthropic）需通过兼容代理（如 OpenRouter）中转，用户修改 `base_url` 即可。

### 8.4 Prompt 管理

- 默认 Prompt 通过 `//go:embed` 编译进二进制
- 用户可在 `~/.paperpaper/prompts/` 下放置自定义文件覆盖：
  - `heavy.txt` — INIT 阶段的详实总结指令
  - `light.txt` — CHAT 阶段的问答指令
  - `digest.txt` — 对话元总结指令

## 9. 非功能需求

| 需求 | 目标 | 说明 |
|:---|:---|:---|
| 启动时间 | < 100ms (不包含网络请求) | 纯 Go 编译 + 本地文件 |
| 分发方式 | 单静态二进制文件 | `go install` 安装，无依赖 |
| 内存占用 | < 100MB (论文 + 对话) | 内存管理可控 |
| 异常处理 | 全覆盖 | 网络超时重试、API Key 缺失提示、流式中断处理 |
| 并发控制 | 无竞态 | 异步摘要 goroutine 需加锁保护共享状态 |

## 10. 项目结构

```
paperpaper/
├── main.go                  # 入口
├── go.mod
├── go.sum
├── internal/
│   ├── config/              # 配置加载
│   │   └── config.go
│   ├── api/                 # OpenAI Chat Completion 客户端 + 流式 SSE
│   │   └── client.go
│   ├── prompt/              # Prompt 管理（//go:embed + 用户覆盖）
│   │   ├── embed.go
│   │   └── prompts/
│   │       ├── heavy.txt
│   │       ├── light.txt
│   │       └── digest.txt
│   ├── session/             # 会话数据模型 + JSON 持久化
│   │   └── session.go
│   └── tui/                 # Bubble Tea TUI
│       ├── model.go         # 主 Model
│       ├── view.go          # View 渲染
│       ├── update.go        # Update 消息处理
│       ├── viewport.go      # Viewport 组件
│       ├── textarea.go      # Textarea 组件
│       └── statusbar.go     # 底部状态栏
├── config.yaml              # 示例配置（不打包，仅供参考）
└── README.md
```

## 11. 开发里程碑

### M1: 核心对话

目标：能与单篇论文完整对话。

分为 4 个子步骤，按顺序推进：

| 步骤 | 内容 | 可验证目标 |
|:---|:---|:---|
| M1.1: 项目骨架 | `go mod init`、目录结构、config 加载、`//go:embed` prompt、首次启动引导 | 能启动程序、读取配置、打印 prompt 内容 |
| M1.2: API 客户端 | OpenAI Chat Completion 调用 + 流式 SSE 解析（纯终端 print，无 TUI） | 能在终端里流式打印 AI 回复 |
| M1.3: TUI 框架 | Bubble Tea 模型搭建、Viewport + Textarea 布局、Vim 模式切换 | 终端 UI 能显示、能输入、能滚动 |
| M1.4: 串联 | 流式渲染到 Viewport + INIT/CHAT 两阶段 Prompt 切换 + Token 显示 | 完整的论文对话流程跑通 |

### M2: 会话管理

目标：支持多论文切换与历史恢复。

| 内容 | 说明 |
|:---|:---|
| JSON 持久化 | 会话数据读写、论文列表扫描 |
| /list | 交互式会话列表，支持键盘选择和鼠标点击 |
| /open | 加载历史会话，恢复最近 5 轮到 L2 |
| /delete | 删除当前会话，确认提示 |
| 摘要系统 | 轻量模型异步生成 per-round 摘要 + session title |

### M3: 消息操作

目标：消息可编辑、可删除、L2 动态重组。

| 内容 | 说明 |
|:---|:---|
| /edit | 编辑最近问题，删除 AI 回复并重新生成 |
| /del | 成对删除指定轮次 |
| 操作按钮 | 鼠标悬停显示 Copy/Delete/Edit 按钮 |
| Token 显示 | 每条消息下方显示 token 数 |

### M4: 命令系统

目标：完整的交互闭环。

| 内容 | 说明 |
|:---|:---|
| /summarize | 对话元总结，使用主模型 |
| /export | 导出所有积累内容到 Obsidian |
| /model | 动态切换模型 |
| /config | 动态修改配置 |

### M5: 增强输入

目标：开箱即用的用户体验。

| 内容 | 说明 |
|:---|:---|
| URL 解析 | arxiv2text 子进程 + 纯 Go fallback |
| Obsidian 导出模板 | 自定义模板变量 |
| 配置管理完善 | 完整的 config.yaml 选项 |
