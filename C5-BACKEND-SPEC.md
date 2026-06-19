# C5 渠道故障告警 - 后端实现规格（Codex 任务）

## 2026-06-19

### 任务概述
为 DEEIX-Chat 实现渠道熔断告警和公开状态页后端支持。

### 1. 公开状态页 API（Priority 1 - MVP）

**端点**: `GET /api/v1/status/models`

**响应格式**:
```json
{
  "overallStatus": "operational",  // "operational" | "degraded" | "down"
  "lastUpdated": "2026-06-19T10:30:00Z",
  "models": [
    {
      "modelName": "GPT-4",
      "availability": 0.99,  // 0-1 (0.99 = 99%)
      "status": "operational",  // "operational" | "degraded" | "down"
      "lastChecked": "2026-06-19T10:30:00Z"
    }
  ]
}
```

**数据源**:
- 从现有调用日志（`conversation_logs` 或类似表）聚合
- 时间窗口：最近 24 小时
- 按模型分组：`model_name` 或 `model_id`
- 成功率计算：`成功请求数 / 总请求数`

**状态判定规则**:
- `availability >= 0.95`: `operational` ✅
- `0.80 <= availability < 0.95`: `degraded` ⚠️
- `availability < 0.80`: `down` ❌
- `overallStatus`: 所有模型中最差状态

**安全红线**:
- ❌ **绝不暴露**：渠道名称、上游商标识、渠道 ID、内部路由信息
- ✅ **仅暴露**：模型名称（用户可见的公开名称）、可用性百分比、状态

**性能要求**:
- 响应时间 < 500ms（建议预聚合 + 缓存）
- 无需鉴权（公开端点）
- 支持 HTTP 缓存头（`Cache-Control: public, max-age=60, stale-while-revalidate=120`）

**实现建议**:
1. 定时任务（每 5 分钟）预聚合模型可用性数据到缓存表或 Redis
2. API 直接读缓存，避免实时聚合大量日志
3. 如果没有调用日志，可以先返回默认值（所有模型 `operational`）

---

### 2. 熔断告警通知（Priority 2）

**触发时机**:
- 渠道熔断器状态变化：`CLOSED` → `OPEN`（熔断）
- 渠道恢复：`OPEN` → `CLOSED`（恢复）
- 半开尝试失败/成功

**告警去抖**:
- 同一渠道 5 分钟内只发一次告警
- 使用 Redis 或内存 map 记录最近告警时间

**Notifier 接口**:
```go
type Notifier interface {
    Notify(ctx context.Context, event AlertEvent) error
}

type AlertEvent struct {
    Type        string    // "circuit_open" | "circuit_closed"
    ChannelID   int       
    ChannelName string
    ModelNames  []string  // 受影响的模型
    Timestamp   time.Time
    Message     string    // 人类可读消息
}
```

**实现者**:
1. **TelegramNotifier** (Priority 2.1)
   - 配置：`settings.alerting.telegram_bot_token`、`settings.alerting.telegram_chat_id`
   - 消息格式：
     ```
     🔴 渠道熔断告警
     渠道: [渠道名]
     影响模型: GPT-4, GPT-4o
     时间: 2026-06-19 10:30:00
     ```
   - 测试端点：`POST /api/v1/admin/alerting/test-telegram`

2. **WebhookNotifier** (Priority 2.2)
   - 配置：`settings.alerting.webhook_url`
   - POST JSON payload：
     ```json
     {
       "type": "circuit_open",
       "channel_name": "OpenAI Official",
       "models": ["GPT-4", "GPT-4o"],
       "timestamp": "2026-06-19T10:30:00Z",
       "message": "..."
     }
     ```

3. **SMTPNotifier** (Priority 2.3 - 可选)
   - 配置：`settings.alerting.smtp_*`（host/port/user/pass/to）
   - 邮件主题：`[DEEIX-Chat] 渠道熔断告警 - [渠道名]`

**集成点**:
- 在现有熔断器（circuit breaker）状态变化处挂 hook
- 如果没有熔断器，可以先跳过此部分，Phase 1 公开状态页是独立的

---

### 3. Admin 配置接口（Priority 3）

**端点**:
- `GET /api/v1/admin/alerting/config` - 查询告警配置
- `PATCH /api/v1/admin/alerting/config` - 更新配置
- `POST /api/v1/admin/alerting/test-telegram` - 测试 Telegram 通知
- `POST /api/v1/admin/alerting/test-webhook` - 测试 Webhook

**配置存储**:
- 使用现有 `settings` 表
- Namespace: `alerting`
- Keys:
  - `telegram_bot_token`
  - `telegram_chat_id`
  - `webhook_url`
  - `smtp_host`/`smtp_port`/`smtp_user`/`smtp_pass`/`smtp_to`
  - `enabled_notifiers` (JSON array: `["telegram", "webhook"]`)

---

### 4. 探活（Priority 4 - 可选）

**功能**:
- 定时器（每 15 分钟）主动调用各模型 TestModel
- 记录探测结果（成功/失败）
- 补充"无人调用"时段的可用性数据

**实现**:
- 复用 admin 的 TestModel 能力
- 可选：只探测关键模型（GPT-4/Claude/Gemini）

---

## 验收标准

### Phase 1（MVP）:
- ✅ `GET /api/v1/status/models` 返回正确格式的 JSON
- ✅ 模型可用性数据基于真实调用日志聚合
- ✅ 响应时间 < 500ms
- ✅ 前端 `/status` 页面能正确展示数据

### Phase 2:
- ✅ 渠道熔断时发送 Telegram 告警
- ✅ 同一渠道 5 分钟去抖生效
- ✅ Admin 可配置 Telegram token 并测试发送

### Phase 3:
- ✅ Webhook notifier 正常工作
- ✅ Admin 配置页可管理所有告警通道

---

## 安全注意事项
1. **公开状态页**：绝不暴露渠道/上游信息
2. **Admin 配置**：Telegram token/SMTP password 等敏感信息加密存储
3. **告警消息**：不在告警中暴露用户数据或请求内容

---

## 工作量估算
- Phase 1（状态页 API）：~4h（含聚合逻辑 + 缓存）
- Phase 2（Telegram 告警）：~3h（hook + notifier + 去抖）
- Phase 3（Admin 配置页后端）：~2h
- Phase 4（探活）：~2h（可选）

**总计**: ~11h（不含探活 ~9h）

---

## 实现顺序建议
1. Phase 1 状态页 API（独立，前端已完成 mock）
2. Phase 2 Telegram 告警（最实用）
3. Phase 3 Admin 配置
4. Phase 4 探活（锦上添花）
