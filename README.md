# PaperPaper

终端论文阅读 AI 助手。粘贴论文，AI 生成详实总结，随后进入多轮问答。

## 安装

```bash
go install github.com/happyTonakai/paper-paper@latest
```

或从源码构建：

```bash
git clone https://github.com/happyTonakai/paper-paper.git
cd paperpaper
go build -o paperpaper .
```

## 配置

支持三种方式（优先级从高到低）：

### 1. 环境变量（推荐）

```bash
export OPENAI_API_KEY="sk-..."
export OPENAI_BASE_URL="https://api.openai.com/v1"   # 可选
export OPENAI_MODEL_NAME="gpt-4o"                     # 可选
```

### 2. 配置文件

```bash
mkdir -p ~/.paperpaper
cat > ~/.paperpaper/config.yaml << 'EOF'
api:
  base_url: "https://api.openai.com/v1"
  api_key: "${OPENAI_API_KEY}"
  default_model: "gpt-4o"
  light_model: "gpt-4o-mini"
obsidian:
  vault_path: "~/Documents/Obsidian/MyVault"
  export_folder: "Papers"
ui:
  max_recent_rounds: 5
EOF
```

### 3. 自定义 Prompt

在 `~/.paperpaper/prompts/` 下放置文件覆盖默认 prompt：

- `heavy.txt` — 初始总结指令
- `light.txt` — 问答指令
- `digest.txt` — 对话元总结指令

## 使用

```bash
# 从文件加载
paperpaper ./paper.txt

# 从 URL 加载
paperpaper https://arxiv.org/abs/2301.00001

# 直接启动，粘贴论文内容
paperpaper
```

## 快捷键

| 按键 | 功能 |
|---|---|
| `i` | 进入输入模式 |
| `Esc` | 返回浏览模式 |
| `j` / `k` | 上下滚动 |
| `Ctrl+D` | 发送 / 半页下滚 |
| `Alt+Enter` | 发送 |
| `q` | 退出 |

## 命令

| 命令 | 功能 |
|---|---|
| `/new [url/path]` | 新建会话 |
| `/list` | 会话列表（j/k 选择，Enter 打开） |
| `/open <id>` | 加载历史会话 |
| `/delete` | 删除当前会话 |
| `/edit` | 编辑最近问题并重新生成 |
| `/del <round>` | 删除指定轮次 |
| `/summarize` | 对话元总结 |
| `/export` | 导出到 Obsidian |
| `/model [name]` | 切换模型 |
| `/config` | 查看配置 |
| `/help` | 帮助 |
| `/quit` | 退出 |

## 架构

```
[System Prompt] → [论文全文] → [最近 5 轮对话] → [当前提问]
```

- **INIT 阶段**：论文全文 → 详实总结（不进入后续上下文）
- **CHAT 阶段**：论文全文 + 最近 5 轮 → 回答问题
- 每轮异步生成 per-question 摘要（仅 UI 展示）

## 测试

```bash
# 全部测试（含真实 API 调用）
go test ./... -v

# 仅单元测试
go test ./internal/config/ ./internal/session/ ./internal/prompt/ ./internal/urlparse/ ./internal/export/ -v
```

## 许可

MIT
