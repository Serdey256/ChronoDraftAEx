// Package kbindex 负责知识库索引更新
// vectordb.go: 向量数据库封装，使用 SQLite 存储向量 + 余弦相似度搜索
// 后续可替换为 LanceDB 等专用向量数据库
package kbindex

import (
	"ChronoDraftAEx/pkg/models"
	"bytes"
	"context"
	"database/sql"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"sort"
	"strings"
	"time"
)

const embeddingDim = 1536 // text-embedding-3-small 维度（仅作降级零向量用，实际维度由 API 返回决定）

// VectorDB 向量数据库封装（基于 SQLite + 余弦相似度）
type VectorDB struct {
	db      *sql.DB
	apiKey  string
	apiBase string
	model   string
	dim     int // 动态维度，首次 API 调用后确定
}

// NewVectorDB 创建向量数据库实例
func NewVectorDB(dbPath string) (*VectorDB, error) {
	db, err := sql.Open("sqlite", dbPath+"?_journal_mode=WAL")
	if err != nil {
		return nil, err
	}
	return &VectorDB{db: db}, nil
}

// SetEmbeddingConfig 配置嵌入模型 API
func (v *VectorDB) SetEmbeddingConfig(apiKey, apiBase, model string) {
	v.apiKey = apiKey
	if apiBase != "" {
		v.apiBase = apiBase
	}
	if model != "" {
		v.model = model
	}
}

// InitSchema 初始化向量表结构
func (v *VectorDB) InitSchema(ctx context.Context) error {
	schema := `
CREATE TABLE IF NOT EXISTS vectors (
	entry_id TEXT PRIMARY KEY,
	session_id TEXT NOT NULL,
	summary TEXT,
	design_decision TEXT,
	impact_analysis TEXT,
	tags TEXT,
	embedding BLOB NOT NULL,
	timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_vectors_session ON vectors(session_id);
CREATE INDEX IF NOT EXISTS idx_vectors_time ON vectors(timestamp);
`
	_, err := v.db.ExecContext(ctx, schema)
	return err
}

// InsertEntry 将结构化知识条目插入向量库
func (v *VectorDB) InsertEntry(ctx context.Context, entry *models.StructuredEntry) error {
	// 生成嵌入向量
	text := entry.Summary + " " + entry.DesignDecision + " " + entry.ImpactAnalysis
	vec, err := v.generateEmbedding(ctx, text)
	if err != nil {
		// 嵌入失败时使用零向量（降级处理）
		dim := v.dim
		if dim == 0 {
			dim = embeddingDim // 未初始化时用默认值
		}
		vec = make([]float32, dim)
	}

	embeddingBlob := encodeVector(vec)
	tagsJSON, _ := json.Marshal(entry.Tags)

	_, err = v.db.ExecContext(ctx,
		`INSERT OR REPLACE INTO vectors (entry_id, session_id, summary, design_decision, impact_analysis, tags, embedding, timestamp)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		entry.ID, entry.SessionID, entry.Summary, entry.DesignDecision,
		entry.ImpactAnalysis, string(tagsJSON), embeddingBlob,
		entry.Timestamp.Format(time.RFC3339),
	)
	return err
}

// DeleteEntry 删除指定知识条目的向量记录
func (v *VectorDB) DeleteEntry(ctx context.Context, entryID string) error {
	_, err := v.db.ExecContext(ctx, `DELETE FROM vectors WHERE entry_id = ?`, entryID)
	if err != nil {
		return fmt.Errorf("删除向量条目失败: %w", err)
	}
	return nil
}

// Search 语义搜索向量库
func (v *VectorDB) Search(ctx context.Context, query string, topK int) ([]models.SearchResult, error) {
	// 生成查询向量
	queryVec, err := v.generateEmbedding(ctx, query)
	if err != nil {
		// 嵌入 API 不可用时回退到关键词匹配
		return v.fallbackKeywordSearch(ctx, query, topK)
	}

	rows, err := v.db.QueryContext(ctx,
		`SELECT entry_id, session_id, summary, design_decision, impact_analysis, tags, embedding, timestamp
		 FROM vectors ORDER BY timestamp DESC LIMIT 1000`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []models.SearchResult
	for rows.Next() {
		var entryID, sessionID, summary, decision, impact, tagsJSON, ts string
		var embeddingBlob []byte
		if err := rows.Scan(&entryID, &sessionID, &summary, &decision, &impact, &tagsJSON, &embeddingBlob, &ts); err != nil {
			continue
		}

		vec := decodeVector(embeddingBlob)
		score := cosineSimilarity(queryVec, vec)

		parsedTime, _ := time.Parse(time.RFC3339, ts)
		var tags []string
		json.Unmarshal([]byte(tagsJSON), &tags)

		results = append(results, models.SearchResult{
			Score: score,
			Entry: models.StructuredEntry{
				ID:             entryID,
				SessionID:      sessionID,
				Timestamp:      parsedTime,
				Summary:        summary,
				DesignDecision: decision,
				ImpactAnalysis: impact,
				Tags:           tags,
			},
		})
	}

	// 按相似度降序排序
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	if len(results) > topK {
		results = results[:topK]
	}
	return results, nil
}

// fallbackKeywordSearch 当嵌入 API 不可用时的关键词回退搜索
func (v *VectorDB) fallbackKeywordSearch(ctx context.Context, query string, topK int) ([]models.SearchResult, error) {
	rows, err := v.db.QueryContext(ctx,
		`SELECT entry_id, session_id, summary, design_decision, impact_analysis, tags, timestamp
		 FROM vectors
		 WHERE summary LIKE '%' || ? || '%'
		    OR design_decision LIKE '%' || ? || '%'
		    OR impact_analysis LIKE '%' || ? || '%'
		 ORDER BY timestamp DESC LIMIT ?`,
		query, query, query, topK,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []models.SearchResult
	for rows.Next() {
		var entryID, sessionID, summary, decision, impact, tagsJSON, ts string
		if err := rows.Scan(&entryID, &sessionID, &summary, &decision, &impact, &tagsJSON, &ts); err != nil {
			continue
		}
		parsedTime, _ := time.Parse(time.RFC3339, ts)
		var tags []string
		json.Unmarshal([]byte(tagsJSON), &tags)
		results = append(results, models.SearchResult{
			Score: 0.5,
			Entry: models.StructuredEntry{
				ID:             entryID,
				SessionID:      sessionID,
				Timestamp:      parsedTime,
				Summary:        summary,
				DesignDecision: decision,
				ImpactAnalysis: impact,
				Tags:           tags,
			},
		})
	}
	return results, rows.Err()
}

// Close 关闭向量数据库连接
func (v *VectorDB) Close() error {
	return v.db.Close()
}

// generateEmbedding 调用嵌入模型 API 生成向量
func (v *VectorDB) generateEmbedding(ctx context.Context, text string) ([]float32, error) {
	if v.apiKey == "" {
		return nil, fmt.Errorf("未配置 AI API Key")
	}

	reqBody, _ := json.Marshal(map[string]interface{}{
		"model": v.embeddingModel(),
		"input": text,
	})

	url := strings.TrimRight(v.apiBase, "/") + "/embeddings"
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+v.apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		errBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("embedding API 返回状态码 %d: %s", resp.StatusCode, string(errBody))
	}

	var result struct {
		Data []struct {
			Embedding []float32 `json:"embedding"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	if len(result.Data) == 0 {
		return nil, fmt.Errorf("embedding API 返回空数据")
	}
	vec := result.Data[0].Embedding
	// 动态记录向量维度（兼容 OpenAI 1536 / DeepSeek 1024 等不同模型）
	if v.dim == 0 && len(vec) > 0 {
		v.dim = len(vec)
	}
	return vec, nil
}

func (v *VectorDB) embeddingModel() string {
	if v.model != "" {
		return v.model
	}
	// 默认：常见嵌入模型，按优先级尝试
	return "text-embedding-3-small"
}

// encodeVector 将 float32 切片编码为字节（小端序）
func encodeVector(vec []float32) []byte {
	buf := new(bytes.Buffer)
	for _, v := range vec {
		binary.Write(buf, binary.LittleEndian, v)
	}
	return buf.Bytes()
}

// decodeVector 将字节解码为 float32 切片
func decodeVector(data []byte) []float32 {
	vec := make([]float32, len(data)/4)
	buf := bytes.NewReader(data)
	for i := range vec {
		binary.Read(buf, binary.LittleEndian, &vec[i])
	}
	return vec
}

// cosineSimilarity 计算两个向量的余弦相似度
func cosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}
	var dot, normA, normB float64
	for i := range a {
		fa, fb := float64(a[i]), float64(b[i])
		dot += fa * fb
		normA += fa * fa
		normB += fb * fb
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}
