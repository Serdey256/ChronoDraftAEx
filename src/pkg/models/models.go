// Package models 定义 ChronoDraftAEx 的核心数据结构
package models

import "time"

// FileChange 表示单个文件的变更记录
type FileChange struct {
	Path       string `json:"path"`
	ChangeType string `json:"change_type"` // add, modify, delete
	Diff       string `json:"diff,omitempty"`
}

// ChangeRecord 是一次对话结束后捕获的原始变动记录
type ChangeRecord struct {
	ID        string       `json:"id"`
	Timestamp time.Time    `json:"timestamp"`
	SessionID string       `json:"session_id"`
	Changes   []FileChange `json:"changes"`
	RawPrompt string       `json:"raw_prompt,omitempty"` // 原始提示词（可选）
}

// StructuredEntry 是 AI 提炼后的结构化知识单元
type StructuredEntry struct {
	ID              string       `json:"id"`
	Timestamp       time.Time    `json:"timestamp"`
	SessionID       string       `json:"session_id"`
	Summary         string       `json:"summary"`          // 代码变更概要
	DesignDecision  string       `json:"design_decision"`  // 设计决策与理由
	ImpactAnalysis  string       `json:"impact_analysis"`  // 潜在影响面
	AffectedFiles   []FileChange `json:"affected_files"`
	Tags            []string     `json:"tags"`
	EmbeddingVector []float32    `json:"-"`                // 向量嵌入（不序列化到 JSON）
}

// ProjectSnapshot 表示项目在某一时刻的快照
type ProjectSnapshot struct {
	ID          string            `json:"id"`
	Timestamp   time.Time         `json:"timestamp"`
	Version     string            `json:"version"`
	Dependencies []string         `json:"dependencies"`
	Metadata    map[string]string `json:"metadata"`
}

// KnowledgeNode 是知识图谱中的节点
type KnowledgeNode struct {
	ID       string            `json:"id"`
	Label    string            `json:"label"`
	Type     string            `json:"type"` // file, module, decision, concept
	Metadata map[string]string `json:"metadata"`
}

// KnowledgeEdge 是知识图谱中的关系边
type KnowledgeEdge struct {
	SourceID string `json:"source_id"`
	TargetID string `json:"target_id"`
	Relation string `json:"relation"` // depends_on, implements, relates_to, caused_by
}

// SearchResult 是知识库搜索的返回结果
type SearchResult struct {
	Entry    StructuredEntry `json:"entry"`
	Score    float64         `json:"score"`
	NodePath []KnowledgeNode `json:"node_path,omitempty"`
}
