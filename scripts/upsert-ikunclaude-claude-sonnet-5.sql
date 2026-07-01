BEGIN;

-- Add platform model claude-sonnet-5 and route it through the existing
-- ikunclaude upstream using OpenRouter's model id anthropic/claude-sonnet-5.
--
-- OpenRouter pricing fetched from https://openrouter.ai/api/v1/models:
-- prompt $0.000002/token ($2/M), completion $0.00001/token ($10/M),
-- input_cache_read $0.0000002/token ($0.20/M),
-- input_cache_write $0.0000025/token ($2.50/M).

DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM llm_upstreams WHERE name = 'ikunclaude') THEN
    RAISE EXCEPTION 'llm upstream "%" not found', 'ikunclaude';
  END IF;
END $$;

WITH up AS (
  SELECT id
  FROM llm_upstreams
  WHERE name = 'ikunclaude'
  LIMIT 1
),
platform_model AS (
  INSERT INTO llm_platform_models (
    created_at,
    updated_at,
    name,
    vendor,
    kinds_json,
    capabilities_json,
    access_scope,
    icon,
    description,
    cb_policy_mode,
    status
  )
  VALUES (
    now(),
    now(),
    'claude-sonnet-5',
    'anthropic',
    '["chat"]',
    '{"contextWindow":1000000,"maxOutputTokens":128000}',
    'public',
    'claude',
    'Anthropic Claude Sonnet 5 via OpenRouter',
    'default',
    'active'
  )
  ON CONFLICT (name) DO UPDATE SET
    updated_at = EXCLUDED.updated_at,
    vendor = EXCLUDED.vendor,
    kinds_json = EXCLUDED.kinds_json,
    capabilities_json = EXCLUDED.capabilities_json,
    access_scope = EXCLUDED.access_scope,
    icon = EXCLUDED.icon,
    description = EXCLUDED.description,
    status = EXCLUDED.status
  RETURNING id
),
upstream_model AS (
  INSERT INTO llm_upstream_models (
    created_at,
    updated_at,
    upstream_id,
    binding_code,
    upstream_model_name,
    vendor,
    icon,
    suggested_protocol,
    kinds_json,
    status,
    source,
    last_synced_at,
    raw_json
  )
  SELECT
    now(),
    now(),
    up.id,
    'upm_openrouter_claude_sonnet_5',
    'anthropic/claude-sonnet-5',
    'anthropic',
    'claude',
    'openrouter_responses',
    '["chat"]',
    'active',
    'manual',
    now(),
    '{"id":"anthropic/claude-sonnet-5","owned_by":"anthropic","openrouter_pricing":{"prompt":"0.000002","completion":"0.00001","web_search":"0.01","input_cache_read":"0.0000002","input_cache_write":"0.0000025","input_cache_write_1h":"0.000004"}}'
  FROM up
  ON CONFLICT (upstream_id, upstream_model_name) DO UPDATE SET
    updated_at = EXCLUDED.updated_at,
    vendor = EXCLUDED.vendor,
    icon = EXCLUDED.icon,
    suggested_protocol = EXCLUDED.suggested_protocol,
    kinds_json = EXCLUDED.kinds_json,
    status = EXCLUDED.status,
    source = EXCLUDED.source,
    last_synced_at = EXCLUDED.last_synced_at,
    raw_json = EXCLUDED.raw_json
  RETURNING id
)
INSERT INTO llm_model_routes (
  created_at,
  updated_at,
  platform_model_id,
  upstream_model_id,
  protocol,
  status,
  priority,
  weight,
  source,
  cb_failure_threshold,
  cb_duration_min,
  cb_window_min,
  headers_json
)
SELECT
  now(),
  now(),
  platform_model.id,
  upstream_model.id,
  'openrouter_responses',
  'active',
  1,
  1,
  'manual',
  0,
  0,
  0,
  ''
FROM platform_model
CROSS JOIN upstream_model
ON CONFLICT (platform_model_id, upstream_model_id, protocol) DO UPDATE SET
  updated_at = EXCLUDED.updated_at,
  status = EXCLUDED.status,
  priority = EXCLUDED.priority,
  weight = EXCLUDED.weight,
  source = EXCLUDED.source,
  cb_failure_threshold = EXCLUDED.cb_failure_threshold,
  cb_duration_min = EXCLUDED.cb_duration_min,
  cb_window_min = EXCLUDED.cb_window_min,
  headers_json = EXCLUDED.headers_json;

INSERT INTO billing_model_prices (
  created_at,
  updated_at,
  deleted_at,
  platform_model_name,
  currency,
  is_free,
  pricing_mode,
  input_nanousd_per_m_tokens,
  cache_read_nanousd_per_m_tokens,
  cache_write_nanousd_per_m_tokens,
  output_nanousd_per_m_tokens,
  call_nanousd_per_call,
  duration_nanousd_per_second,
  tiered_pricing_json
) VALUES (
  now(),
  now(),
  NULL,
  'claude-sonnet-5',
  'USD',
  false,
  'token',
  2000000000,
  200000000,
  2500000000,
  10000000000,
  0,
  0,
  '{}'
)
ON CONFLICT (platform_model_name) DO UPDATE SET
  updated_at = EXCLUDED.updated_at,
  deleted_at = NULL,
  currency = EXCLUDED.currency,
  is_free = EXCLUDED.is_free,
  pricing_mode = EXCLUDED.pricing_mode,
  input_nanousd_per_m_tokens = EXCLUDED.input_nanousd_per_m_tokens,
  cache_read_nanousd_per_m_tokens = EXCLUDED.cache_read_nanousd_per_m_tokens,
  cache_write_nanousd_per_m_tokens = EXCLUDED.cache_write_nanousd_per_m_tokens,
  output_nanousd_per_m_tokens = EXCLUDED.output_nanousd_per_m_tokens,
  call_nanousd_per_call = EXCLUDED.call_nanousd_per_call,
  duration_nanousd_per_second = EXCLUDED.duration_nanousd_per_second,
  tiered_pricing_json = EXCLUDED.tiered_pricing_json;

COMMIT;
