// Package utils 提供 ChronoDraftAEx 的通用工具函数
package utils

import (
	"fmt"
	"strings"

	"ChronoDraftAEx/pkg/models"
)

// EstimateTokens 估算文本的 token 数量，使用 len(text)/3 启发式算法
// 返回估算的 token 数，短文本至少返回 1
func EstimateTokens(text string) int {
	n := len(text) / 3
	if n == 0 && len(text) > 0 {
		return 1
	}
	return n
}

// TruncateToBudget 按行截断文本至指定 token 预算，保留完整行
// 如果文本已经在预算内，直接返回原文本。
// 从末尾逐行移除直到满足预算，始终保留完整的行。
func TruncateToBudget(text string, maxTokens int) string {
	if maxTokens < 0 {
		return ""
	}
	if EstimateTokens(text) <= maxTokens {
		return text
	}
	lines := strings.Split(text, "\n")
	for i := len(lines); i > 0; i-- {
		candidate := strings.Join(lines[:i], "\n")
		if EstimateTokens(candidate) <= maxTokens {
			return candidate
		}
	}
	return ""
}

// CompressEntry 将结构化条目格式化为单行摘要，并按 token 预算截断
// 格式: "Summary: [summary] | Decision: [design_decision] | Impact: [impact_analysis]"
func CompressEntry(entry models.StructuredEntry, maxTokens int) string {
	text := fmt.Sprintf("Summary: %s | Decision: %s | Impact: %s",
		entry.Summary, entry.DesignDecision, entry.ImpactAnalysis)
	return TruncateToBudget(text, maxTokens)
}
