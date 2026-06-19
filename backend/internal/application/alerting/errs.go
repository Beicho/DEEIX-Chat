package alerting

import "errors"

// ErrInvalidConfig 表示告警配置校验失败。
var ErrInvalidConfig = errors.New("invalid alerting config")
