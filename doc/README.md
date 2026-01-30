# LAD 架构设计文档

Lambda Alias Deployment (LAD) 是一个用于管理 AWS Lambda 函数灰度发布的命令行工具。

## 文档索引

- [状态图](./state-diagram.md) - 别名状态流转
- [状态转换检查](./state-transitions.md) - 命令状态转换矩阵和安全检查
- [流程图](./workflow.md) - 完整工作流程
- [SAM 协同图](./sam-integration.md) - 与 SAM 的协作关系
- [别名变化图](./alias-changes.md) - 各命令对别名的影响

## 核心概念

### 三个别名

| 别名 | 用途 | 流量来源 |
|------|------|----------|
| `live` | 生产流量 | API Gateway、Schedule 等触发器 |
| `latest` | 最新版本 | 测试验证 |
| `previous` | 回退版本 | 紧急回退时使用 |

### 命令列表

| 命令 | 功能 |
|------|------|
| `patch` | 给 template.yaml 添加 Version 和 Alias 资源 |
| `unpatch` | 移除补丁内容，还原 template.yaml |
| `deploy` | 部署新版本，更新 latest 别名 |
| `canary` | 手动灰度发布，按指定百分比分配流量 |
| `auto` | 自动递进灰度发布 |
| `promote` | 完成灰度，100% 切换到新版本 |
| `rollback` | 紧急回退到上一个稳定版本 |
| `status` | 查看当前别名状态 |
| `switch` | 极端情况下切换到指定版本 |

### canary 命令

使用 `--percent` 参数指定新版本流量百分比 (0-100)：

```bash
lad canary --env test --percent 10   # 10% 流量到新版本
lad canary --env test --percent 50   # 50% 流量到新版本
lad canary --env test --percent 0    # 清除灰度配置
```

### auto 命令

自动递进灰度发布，支持自定义步长和等待时间：

```bash
lad auto --env test                           # 默认: 每次 +10%，每阶段等待 5 分钟
lad auto --env test --percent 25 --wait 1h    # 每次 +25%，每阶段等待 1 小时
```

参数说明：
- `--percent`: 每次增加的百分比 (默认 10)
- `--wait`: 每阶段等待时间 (默认 5m)
