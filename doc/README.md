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
| `canary` | 手动灰度发布，按指定比例分配流量 |
| `auto` | 自动递进灰度发布 (10%→25%→50%→75%→100%) |
| `promote` | 完成灰度，100% 切换到新版本 |
| `rollback` | 紧急回退到上一个稳定版本 |
| `status` | 查看当前别名状态 |
| `switch` | 极端情况下切换到指定版本 |

### 灰度策略

```
canary0   → 0% 新版本（清除灰度）
canary10  → 10% 新版本
canary25  → 25% 新版本
canary50  → 50% 新版本
canary75  → 75% 新版本
canary100 → 100% 新版本（不更新 previous）
promote   → 100% 新版本（更新 previous）
```

### 自动灰度

```bash
lad auto --wait 5m   # 每阶段等待 5 分钟，自动递进到 100%
```
