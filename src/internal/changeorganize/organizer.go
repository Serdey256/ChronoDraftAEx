// Package changeorganize 负责将原始变动记录提炼为结构化的知识单元
// 核心功能：调用 AI API 生成代码变更概要、设计决策与理由、潜在影响面
package changeorganize

import (
	"ChronoDraftAEx/pkg/models"
	"ChronoDraftAEx/pkg/utils"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Organizer AI 结构化摘要生成器
type Organizer struct {
	apiKey     string
	apiBaseURL string
	model      string
}

// NewOrganizer 创建一个 Organizer 实例
func NewOrganizer(apiKey, apiBaseURL, model string) *Organizer {
	if model == "" {
		model = "gpt-4o"
	}
	return &Organizer{
		apiKey:     apiKey,
		apiBaseURL: apiBaseURL,
		model:      model,
	}
}

// Organize 接收原始变动记录，返回结构化知识条目
func (o *Organizer) Organize(record *models.ChangeRecord) (*models.StructuredEntry, error) {
	// 构造 AI 提示词
	prompt := o.buildPrompt(record)

	// 调用 AI API 获取摘要
	aiResponse, err := o.callAI(prompt)
	if err != nil {
		return nil, fmt.Errorf("AI 调用失败: %w", err)
	}

	entry := &models.StructuredEntry{
		ID:            utils.GenerateID(),
		Timestamp:     time.Now(),
		SessionID:     record.SessionID,
		AffectedFiles: record.Changes,
	}

	// 解析 AI 返回的 JSON 结构
	// 期望格式: {"summary": "...", "design_decision": "...", "impact_analysis": "...", "tags": [...]}
	var parsed struct {
		Summary        string   `json:"summary"`
		DesignDecision string   `json:"design_decision"`
		ImpactAnalysis string   `json:"impact_analysis"`
		Tags           []string `json:"tags"`
	}
	if err := json.Unmarshal([]byte(aiResponse), &parsed); err != nil {
		// 如果解析失败，将原始响应作为 summary
		entry.Summary = aiResponse
		entry.DesignDecision = "未能自动解析 AI 响应"
		entry.ImpactAnalysis = "未能自动解析 AI 响应"
	} else {
		entry.Summary = parsed.Summary
		entry.DesignDecision = parsed.DesignDecision
		entry.ImpactAnalysis = parsed.ImpactAnalysis
		entry.Tags = parsed.Tags
	}

	return entry, nil
}

// buildPrompt 构建发送给 AI 的结构化提示词
func (o *Organizer) buildPrompt(record *models.ChangeRecord) string {
	filesDesc := ""
	for _, ch := range record.Changes {
		filesDesc += fmt.Sprintf("- [%s] %s\n", ch.ChangeType, ch.Path)
	}

	return fmt.Sprintf(`
你是一名资深软件架构师，负责将代码变动记录提炼为结构化的工程知识。

## 原始变动记录
Session ID: %s
时间: %s

受影响文件:
%s

## 任务
请根据以上变动记录，以 JSON 格式返回以下信息（不要包含 markdown 代码块标记，仅返回纯 JSON）：
{
  "summary": "用 1-2 句话概括本次代码变更的核心内容",
  "design_decision": "分析并说明做出这些变更的设计决策与理由",
  "impact_analysis": "评估这些变更可能影响的模块、接口或功能面",
  "tags": ["相关模块名", "技术标签", "影响范围标签"]
}
`, record.SessionID, record.Timestamp.Format(time.RFC3339), filesDesc)
}

// callAI 调用云端 AI API（支持 OpenAI 兼容格式）
func (o *Organizer) callAI(prompt string) (string, error) {
	// 构造请求体
	reqBody, _ := json.Marshal(map[string]interface{}{
		"model": o.model,
		"messages": []map[string]string{
			{"role": "system", "content": "You are a helpful software architecture assistant."},
			{"role": "user", "content": prompt},
		},
		"temperature": 0.3,
	})

	url := o.apiBaseURL + "/chat/completions"
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+o.apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("AI API 返回非 200 状态码: %d", resp.StatusCode)
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	if len(result.Choices) == 0 {
		return "", fmt.Errorf("AI API 返回空 choices")
	}
	return result.Choices[0].Message.Content, nil
}
