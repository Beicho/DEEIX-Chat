# C5 渠道故障告警 + 公开状态页 实现计划

## 2026-06-19

### 需求（from master plan）
- **告警通知**：渠道熔断/恢复时通知站长（Telegram bot/邮件/webhook）
- **公开状态页**：`/status` 路由（无需登录），展示各模型最近 24h 可用性
- **探活**：可选主动探测（避免"无人调用=显示正常"盲区）

### 架构设计

#### 1. 后端（Codex）
- 熔断状态机 hook：在 circuit breaker open/close 事件挂 notifier
- Notifier 接口：Telegram/Webhook/SMTP 三实现
- 配置：settings namespace `alerting.*`（telegram_token/webhook_url/smtp_config）
- 去抖：同渠道 5 分钟聚合一条告警
- 状态页 API：`GET /api/v1/status/models`（聚合最近 24h 成功率）
- 数据源：现有调用日志（按模型小时聚合）
- 探活（可选）：定时器跑 TestModel，记录探测结果

#### 2. 前端（Claude）
- `/status` 公开页面（无需登录，SSR 友好）
- 展示：模型名 + 可用性指示器（绿/黄/红）
- 绝不暴露：渠道名、上游商、内部 ID
- 响应式布局（移动端友好）
- CF 缓存友好（stale-while-revalidate）

#### 3. Admin 管理页
- `/admin/alerting`：配置告警通道
- Telegram bot token 输入 + 测试发送
- Webhook URL + 测试 ping
- SMTP 配置 + 测试邮件
- 告警历史记录

### 分阶段
1. **Phase 1**（MVP）：公开状态页 `/status` + 模型可用性展示
2. **Phase 2**：后端熔断告警 hook + Telegram 通知
3. **Phase 3**：Admin 配置页 + Webhook/SMTP 支持
4. **Phase 4**：探活（可选）

### 实现顺序
1. 前端：`/status` 页面（先做静态 mock，等后端 API）
2. 后端（Codex）：状态页 API + 模型可用性聚合
3. 前端：接入真实 API
4. 后端（Codex）：熔断告警 hook + Telegram notifier
5. 前端：Admin 配置页
6. 后端（Codex）：Webhook/SMTP notifier

当前优先：**Phase 1 前端 `/status` 页面**（等签到部署完成后立即开工）
