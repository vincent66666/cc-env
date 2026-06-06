package tui

import "strings"

// maskSecret 遮罩敏感值，仅保留首尾两位。
func maskSecret(value string) string {
	if len(value) <= 4 {
		return "****"
	}
	return value[:2] + strings.Repeat("*", len(value)-4) + value[len(value)-2:]
}
