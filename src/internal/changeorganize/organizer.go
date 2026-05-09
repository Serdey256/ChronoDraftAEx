// Package changeorganize 负责将原始变动记录提炼为结构化的知识单元
// 核心功能：调用 AI API 生成代码变更概要、设计决策与理由、潜在影响面
package changeorganize

import (
	"ChronoDraftAEx/pkg/models"
	"ChronoDraftAEx/pkg/utils"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
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

// AnnotateEntities 使用 AI 为代码实体生成简短中文描述（15字以内）
// 接收代码实体列表，为每个尚无描述的实体生成描述并更新其 Metadata 字段
// 无 API Key 时直接返回原列表（不报错）
func (o *Organizer) AnnotateEntities(entities []models.CodeEntity) ([]models.CodeEntity, error) {
	if o.apiKey == "" {
		return entities, nil
	}

	// 过滤出需要标注的实体（Metadata 中尚无 description）
	var toAnnotate []models.CodeEntity
	for _, e := range entities {
		if e.Metadata == "" || !strings.Contains(e.Metadata, "\"description\"") {
			toAnnotate = append(toAnnotate, e)
		}
	}
	if len(toAnnotate) == 0 {
		return entities, nil
	}

	// 构建提示词
	var sb strings.Builder
	sb.WriteString("为以下代码实体各生成一句简短的中文描述（15字以内），说明其功能。")
	sb.WriteString("返回 JSON 数组，格式：[{\"name\":\"实体名\",\"entity_type\":\"类型\",\"description\":\"中文描述\"}]，不要包含 markdown 代码块标记：\n")
	for _, e := range toAnnotate {
		sb.WriteString(fmt.Sprintf("- %s %s", e.EntityType, e.Name))
		if e.Signature != "" {
			sb.WriteString(fmt.Sprintf(": %s", e.Signature))
		}
		sb.WriteString(fmt.Sprintf(" (file: %s)\n", e.FilePath))
	}

	aiResponse, err := o.callAI(sb.String())
	if err != nil {
		return entities, fmt.Errorf("AI 标注调用失败: %w", err)
	}

	// 清理响应：移除可能的 markdown 代码块标记
	cleaned := strings.TrimSpace(aiResponse)
	cleaned = strings.TrimPrefix(cleaned, "```json")
	cleaned = strings.TrimPrefix(cleaned, "```")
	cleaned = strings.TrimSuffix(cleaned, "```")
	cleaned = strings.TrimSpace(cleaned)

	var annotations []struct {
		Name        string `json:"name"`
		EntityType  string `json:"entity_type"`
		Description string `json:"description"`
	}
	if err := json.Unmarshal([]byte(cleaned), &annotations); err != nil {
		return entities, fmt.Errorf("解析 AI 标注响应失败: %w", err)
	}

	// 构建查找表
	type entityKey struct{ name, entityType string }
	descMap := make(map[entityKey]string)
	for _, a := range annotations {
		descMap[entityKey{a.Name, a.EntityType}] = a.Description
	}

	// 更新实体 Metadata
	for i := range entities {
		key := entityKey{entities[i].Name, entities[i].EntityType}
		desc, ok := descMap[key]
		if !ok || desc == "" {
			continue
		}
		// 如果 Metadata 已有内容，追加 description；否则新建
		meta := make(map[string]string)
		if entities[i].Metadata != "" {
			_ = json.Unmarshal([]byte(entities[i].Metadata), &meta)
		}
		meta["description"] = desc
		metaBytes, _ := json.Marshal(meta)
		entities[i].Metadata = string(metaBytes)
	}

	return entities, nil
}

// callAI 调用云端 AI API（支持 OpenAI 兼容格式）
func (o *Organizer) callAI(prompt string) (string, error) {
	if o.apiKey == "" {
		return o.generateLocalSummary(prompt), nil
	}

	// 构造请求体
	reqBody, _ := json.Marshal(map[string]interface{}{
		"model": o.model,
		"messages": []map[string]string{
			{"role": "system", "content": "You are a helpful software architecture assistant."},
			{"role": "user", "content": prompt},
		},
		"temperature": 0.3,
	})

	url := strings.TrimRight(o.apiBaseURL, "/") + "/chat/completions"
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
		errBody, _ := io.ReadAll(resp.Body)
		fmt.Fprintf(os.Stderr, "[changeorganize] AI API 错误 (status %d): %s\n", resp.StatusCode, string(errBody))
		return o.generateLocalSummary(prompt), nil
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

// generateLocalSummary 当 AI API 不可用时生成本地摘要
func (o *Organizer) generateLocalSummary(prompt string) string {
	return `{"summary": "代码变更已记录（AI 摘要服务暂不可用）", "design_decision": "手动代码修改", "impact_analysis": "具体影响需人工评估", "tags": ["manual-change"]}`
}
