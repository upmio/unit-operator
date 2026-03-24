# Milvus 日志转发到标准输出 设计文档

## 需求背景

当服务类型是 Milvus 时，需要把其日志打印到标准输出，目的是让 Java 程序能够读取容器内标准输出的 Milvus 运行日志，从而在页面进行展示。

## 现状分析

### Milvus 进程管理
- Milvus 进程由 **supervisord** 管理（进程名：`unit_app`）
- Agent 通过 XML-RPC 与 supervisord 通信来启停 Milvus

### 当前日志配置

#### 1. milvusTemplate.tpl (Milvus Server 配置)
```yaml
log:
  file:
    maxAge: 10
    maxBackups: 20
    maxSize: 300
    rootPath: null
  format: text
  level: info
  stdout: true     # 已配置输出到标准输出
```

#### 2. supervisord.conf
```ini
[program:unit_app]
command=/usr/local/milvus/bin/milvus run %(ENV_ARCH_MODE)s
stderr_logfile=%(ENV_LOG_MOUNT)s/unit_app.err.log
stdout_logfile=%(ENV_LOG_MOUNT)s/unit_app.out.log
autostart=false
```

### 问题分析

即使 Milvus 配置了 `stdout: true`，但 supervisord 捕获了子进程的 stdout/stderr 并重定向到日志文件，导致日志无法到达容器标准输出。

## 方案选型

### 方案对比

| 方案 | 描述 | 优点 | 缺点 |
|-----|------|------|------|
| 方案 A | 修改 supervisord.conf 输出到 /dev/stdout | 最简单 | 影响现有日志文件设计 |
| 方案 B | Milvus 原生支持同时输出 | 配置简单 | supervisord 仍会拦截 stdout |
| 方案 C | Agent 新增 DaemonApp 转发日志 | 不修改其他仓库 | 需要新增代码 |

### 最终选择

**选择方案 C：在 agent 中新增 DaemonApp 转发日志**

原因：
1. 不需要修改 upm-packages 仓库
2. 不影响现有 supervisord 日志文件设计
3. 实现模式成熟（参考 rediscluster）

## 实现方案

### 1. 新增文件

```
pkg/agent/app/milvus/
├── impl.go              # 现有实现
├── logtail.go          # 新增：日志转发 DaemonApp
└── logtail_test.go     # 新增：测试
```

### 2. 架构设计

```
┌─────────────────────────────────────────────────────────┐
│  Milvus Process                                        │
│    stdout → unit_app.out.log                          │
│    stderr → unit_app.err.log                           │
└─────────────────────────────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────┐
│  DaemonApp (logtail)                                   │
│    tail -f unit_app.out.log → /dev/stdout              │
│    tail -f unit_app.err.log → /dev/stderr              │
└─────────────────────────────────────────────────────────┘
                           │
                           ▼
                    Container stdout
                           │
                           ▼
                    Java Application
```

### 3. 实现要点

#### 3.1 文件路径
```go
const (
    outLogFile = "unit_app.out.log"
    errLogFile = "unit_app.err.log"
)
```

#### 3.2 两种实现方式（可选）

**方式一：Go 原生实现（推荐）**
```go
func (lt *logtail) tailFile(path string, output io.Writer) {
    file, err := os.Open(path)
    // 使用 bufio.Scanner 读取增量内容
    // 处理日志轮转（文件被 rename 后重新打开）
}
```

**方式二：exec.Command 调用 tail（已注释）**
```go
// 已注释，使用 Go 原生实现
// func (lt *logtail) tailFileByCommand(path string, output io.Writer) {
//     cmd := exec.Command("tail", "-f", "-n", "+1", path)
//     cmd.Stdout = output
//     cmd.Run()
// }
```

#### 3.3 环境变量控制（已注释）
```go
// 已注释，当前设计始终启用
// const (
//     LogStdoutEnableEnvKey = "LOG_STDOUT_ENABLE"
// )
```

### 4. 核心功能

#### 4.1 日志轮转处理
- 使用 fsnotify 监控日志文件所在目录
- 检测到文件被 rename/create 后，重新打开文件继续 tail

#### 4.2 启动时文件不存在处理
- 文件不存在时等待文件创建
- 使用定期检查 + fsnotify 结合的方式

### 5. 注册方式

```go
func init() {
    app.RegistryDaemonApp(logtailDaemon)
}
```

## 资源消耗评估

| 资源 | 消耗量 | 说明 |
|-----|-------|------|
| CPU | ~0% | 空闲时几乎为 0 |
| 内存 | ~5-10MB | tail 进程本身非常轻量 |
| 文件描述符 | 2-3 个 | 监控的文件 + stdin/stdout |
| 磁盘 I/O | 很低 | 只读增量日志 |

## 潜在问题及处理

| 问题 | 处理方式 |
|-----|---------|
| 日志轮转 | 使用 fsnotify 监控目录，检测到文件变化后重新打开 |
| 文件不存在 | 启动时等待文件创建 |
| stderr 转发 | 同时 tail unit_app.err.log 到 /dev/stderr |

## 实现清单

- [x] 创建 logtail.go，实现 DaemonApp 接口
- [x] 实现 Go 原生 tail 功能
- [x] 处理日志轮转
- [x] 同时转发 stdout 和 stderr
- [x] 启动时文件不存在处理
- [x] 添加 exec.Command tail 方式（已注释）
- [x] 添加环境变量控制方式（已注释）
- [x] 添加单元测试

## 讨论时间线

1. 初始需求：Milvus 日志打印到标准输出
2. 发现 backup.tmpl 是 milvus-backup 配置，不是 Milvus Server 配置
3. 分析 supervisord.conf 发现 stdout 被重定向到文件
4. 验证 milvusTemplate.tpl 的 stdout: true 配置
5. 确定方案 C：Agent 新增 DaemonApp
6. 评估资源消耗和潜在问题
7. 确定最终实现方案
