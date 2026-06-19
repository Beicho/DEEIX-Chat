# C2 模型竞技场（Arena）- 后端实现规格（Codex 任务）

## 2026-06-19

## 任务概述
实现"同一问题并排发给 2-4 个模型对比"的竞技场后端支持。

## 现状调研结论
- 分支树已支持同一 parent 多 assistant 分支（`ParentMessageID` + `BranchReason`）
- 现有 stream 提交：`StreamMessage` / `SendMessage`（service_message_send.go）
- 现有 branchReason 值：`default` / `retry`
- 后端目前**无** arena/并行多模型支持

## Phase 1：竞技场消息（核心）

### 数据模型
新增 `arena_votes` 表：
```sql
CREATE TABLE arena_votes (
  id BIGSERIAL PRIMARY KEY,
  message_group_id VARCHAR(64) NOT NULL,  -- 同一轮竞技的标识（同 parent 的一组 assistant 分支）
  conversation_id BIGINT NOT NULL,
  voter_user_id BIGINT NOT NULL,
  winner_model VARCHAR(128) NOT NULL,     -- 投票选中的模型
  blind_mode BOOLEAN DEFAULT FALSE,        -- 是否盲投
  created_at TIMESTAMP DEFAULT NOW(),
  UNIQUE(message_group_id, voter_user_id)  -- 每组每人只能投一次
);
CREATE INDEX idx_arena_votes_group ON arena_votes(message_group_id);
CREATE INDEX idx_arena_votes_winner ON arena_votes(winner_model);
```

### API 端点
1. **发起竞技场对比**：扩展现有 stream 端点或新增
   - `POST /api/v1/conversations/:id/arena`
   - 请求：`{ content, models: ["gpt-4", "claude-3.5", ...], blindMode: bool }`（2-4 个模型）
   - 行为：对同一 user message，并行创建 N 个 assistant 分支（branchReason="arena"）
   - 每个分支用不同模型，正常计费
   - 返回 N 路 SSE 流（或复用现有 stream 机制，前端发 N 次请求亦可）

2. **投票**：
   - `POST /api/v1/conversations/:id/arena-vote`
   - 请求：`{ messageGroupID, winnerModel, blindMode }`
   - 幂等：同组同用户只能投一次（UNIQUE 约束）

### 计费
- 每个模型按正常单价扣费
- 发起前后端返回预估（前端 UI 明示"将按 N 个模型分别计费"）

### 流式
- N 路并行，单路失败不影响其余
- 失败的分支标记错误状态，前端显示重试按钮

## Phase 2：偏好榜（Admin）
- `GET /api/v1/admin/arena/leaderboard`
- 聚合 arena_votes：每模型 win_count / total_battles / win_rate
- 按 win_rate 排序

## 安全/红线
- 盲投模式：后端在投票完成前不向前端暴露模型名（用占位符 modelA/modelB）
- 计费透明：发起前明示将扣 N 次费

## 验收
- 3 模型对比一次成功创建 3 个分支
- 投票入库，重复投票被拒
- 偏好榜正确聚合

## 工作量：L（大）

## 实现建议
考虑到复杂度，Phase 1 可以简化为：
- 前端发起 N 次独立 stream 请求（每个不同 model + 相同 branchReason="arena" + 相同 messageGroupID）
- 后端只需：①接受 branchReason="arena" + messageGroupID 参数 ②arena_votes 表 + 投票端点
- 这样避免改动核心 stream 逻辑，风险最小

## 注意
- PRoot 环境 go test 段错误，用 go build 验证编译
- 推 beicho/dev
