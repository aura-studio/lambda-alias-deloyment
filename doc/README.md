# LAD 架构设计文档

Lambda Alias Deployment (LAD) 是一个用于管理 AWS Lambda 函数灰度发布的命令行工具。

## 文档索引

- [状态图](./state-diagram.md) - 别名状态流转
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

### 灰度策略

```
canary10  → 10% 新版本
canary25  → 25% 新版本
canary50  → 50% 新版本
canary75  → 75% 新版本
promote   → 100% 新版本
```
