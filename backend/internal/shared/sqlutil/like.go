// Package sqlutil 提供 SQL 相关的通用工具函数。
package sqlutil

import "strings"

// EscapeLIKE 转义 LIKE 模式中的特殊字符（\、%、_），
// 防止用户输入的搜索关键字被解释为通配符（INJ-003）。
func EscapeLIKE(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "%", "\\%")
	s = strings.ReplaceAll(s, "_", "\\_")
	return s
}
