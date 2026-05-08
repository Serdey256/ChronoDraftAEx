// Package kbindex 负责知识库索引更新
// graphdb.go: 图数据库封装，使用 SQLite 存储节点和边实现关系推理
// 后续可替换为 Kuzu 等专用图数据库
package kbindex

import (
	"ChronoDraftAEx/pkg/models"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

// GraphDB 图数据库封装（基于 SQLite 邻接表）
type GraphDB struct {
	db *sql.DB
}

// NewGraphDB 创建图数据库实例
func NewGraphDB(dbPath string) (*GraphDB, error) {
	db, err := sql.Open("sqlite", dbPath+"?_journal_mode=WAL")
	if err != nil {
		return nil, err
	}
	return &GraphDB{db: db}, nil
}

// InitSchema 初始化图数据库 Schema
func (g *GraphDB) InitSchema(ctx context.Context) error {
	schema := `
CREATE TABLE IF NOT EXISTS nodes (
	id TEXT PRIMARY KEY,
	label TEXT NOT NULL,
	type TEXT NOT NULL,
	metadata TEXT,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS edges (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	source_id TEXT NOT NULL,
	target_id TEXT NOT NULL,
	relation TEXT NOT NULL,
	metadata TEXT,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	FOREIGN KEY (source_id) REFERENCES nodes(id),
	FOREIGN KEY (target_id) REFERENCES nodes(id),
	UNIQUE(source_id, target_id, relation)
);

CREATE INDEX IF NOT EXISTS idx_edges_source ON edges(source_id);
CREATE INDEX IF NOT EXISTS idx_edges_target ON edges(target_id);
CREATE INDEX IF NOT EXISTS idx_edges_relation ON edges(relation);
CREATE INDEX IF NOT EXISTS idx_nodes_type ON nodes(type);
`
	_, err := g.db.ExecContext(ctx, schema)
	return err
}

// InsertEntry 将结构化知识条目写入图数据库
func (g *GraphDB) InsertEntry(ctx context.Context, entry *models.StructuredEntry) error {
	tx, err := g.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 1. 创建 Entry 节点
	_, err = tx.ExecContext(ctx,
		`INSERT OR REPLACE INTO nodes (id, label, type, metadata) VALUES (?, ?, ?, ?)`,
		entry.ID, entry.Summary, "entry",
		fmt.Sprintf(`{"session_id":"%s","timestamp":"%s"}`, entry.SessionID, entry.Timestamp.Format(time.RFC3339)),
	)
	if err != nil {
		return fmt.Errorf("插入 Entry 节点失败: %w", err)
	}

	// 2. 创建 File 节点并建立 AFFECTS 边
	for _, fc := range entry.AffectedFiles {
		fileID := "file:" + fc.Path
		_, err = tx.ExecContext(ctx,
			`INSERT OR IGNORE INTO nodes (id, label, type) VALUES (?, ?, ?)`,
			fileID, fc.Path, "file",
		)
		if err != nil {
			return fmt.Errorf("插入 File 节点失败: %w", err)
		}

		_, err = tx.ExecContext(ctx,
			`INSERT OR REPLACE INTO edges (source_id, target_id, relation) VALUES (?, ?, ?)`,
			entry.ID, fileID, "AFFECTS",
		)
		if err != nil {
			return fmt.Errorf("插入 AFFECTS 边失败: %w", err)
		}
	}

	// 3. 创建 Tag 节点并建立 HAS_TAG 边
	for _, tag := range entry.Tags {
		tagID := "tag:" + tag
		_, err = tx.ExecContext(ctx,
			`INSERT OR IGNORE INTO nodes (id, label, type) VALUES (?, ?, ?)`,
			tagID, tag, "tag",
		)
		if err != nil {
			return fmt.Errorf("插入 Tag 节点失败: %w", err)
		}

		_, err = tx.ExecContext(ctx,
			`INSERT OR REPLACE INTO edges (source_id, target_id, relation) VALUES (?, ?, ?)`,
			entry.ID, tagID, "HAS_TAG",
		)
		if err != nil {
			return fmt.Errorf("插入 HAS_TAG 边失败: %w", err)
		}
	}

	// 4. 查找相关条目并建立 RELATED_TO 边（通过共同标签关联）
	if len(entry.Tags) > 0 {
		rows, err := tx.QueryContext(ctx,
			`SELECT DISTINCT e.source_id FROM edges e
			 JOIN edges e2 ON e.target_id = e2.target_id
			 WHERE e2.source_id = ? AND e.relation = 'HAS_TAG' AND e.source_id != ?`,
			entry.ID, entry.ID,
		)
		if err == nil {
			for rows.Next() {
				var relatedID string
				if rows.Scan(&relatedID) == nil {
					tx.ExecContext(ctx,
						`INSERT OR IGNORE INTO edges (source_id, target_id, relation) VALUES (?, ?, ?)`,
						entry.ID, relatedID, "RELATED_TO",
					)
				}
			}
			rows.Close()
		}
	}

	return tx.Commit()
}

// QueryRelated 查询与指定条目相关的知识节点
func (g *GraphDB) QueryRelated(ctx context.Context, entryID string) ([]models.KnowledgeNode, []models.KnowledgeEdge, error) {
	// 查询直接关联的节点
	rows, err := g.db.QueryContext(ctx,
		`SELECT n.id, n.label, n.type, n.metadata, e.relation, e.source_id, e.target_id
		 FROM edges e
		 JOIN nodes n ON (n.id = CASE WHEN e.source_id = ? THEN e.target_id ELSE e.source_id END)
		 WHERE e.source_id = ? OR e.target_id = ?
		 LIMIT 50`,
		entryID, entryID, entryID,
	)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	nodeMap := make(map[string]models.KnowledgeNode)
	var edges []models.KnowledgeEdge

	for rows.Next() {
		var nodeID, label, nodeType, relation, sourceID, targetID string
		var metaJSON sql.NullString
		if err := rows.Scan(&nodeID, &label, &nodeType, &metaJSON, &relation, &sourceID, &targetID); err != nil {
			continue
		}

		if _, exists := nodeMap[nodeID]; !exists {
			meta := make(map[string]string)
			if metaJSON.Valid {
				json.Unmarshal([]byte(metaJSON.String), &meta)
			}
			nodeMap[nodeID] = models.KnowledgeNode{
				ID:       nodeID,
				Label:    label,
				Type:     nodeType,
				Metadata: meta,
			}
		}

		edges = append(edges, models.KnowledgeEdge{
			SourceID: sourceID,
			TargetID: targetID,
			Relation: relation,
		})
	}

	// 加入入口节点本身
	entryRow := g.db.QueryRowContext(ctx,
		`SELECT id, label, type, metadata FROM nodes WHERE id = ?`, entryID,
	)
	var entryNode models.KnowledgeNode
	var metaJSON sql.NullString
	if entryRow.Scan(&entryNode.ID, &entryNode.Label, &entryNode.Type, &metaJSON) == nil {
		if metaJSON.Valid {
			json.Unmarshal([]byte(metaJSON.String), &entryNode.Metadata)
		}
		nodeMap[entryID] = entryNode
	}

	nodes := make([]models.KnowledgeNode, 0, len(nodeMap))
	for _, n := range nodeMap {
		nodes = append(nodes, n)
	}

	return nodes, edges, nil
}

// GetGraphData 获取图谱全量数据（用于前端可视化）
func (g *GraphDB) GetGraphData(ctx context.Context, limit int) ([]models.KnowledgeNode, []models.KnowledgeEdge, error) {
	if limit <= 0 {
		limit = 100
	}

	// 查询节点
	nodeRows, err := g.db.QueryContext(ctx,
		`SELECT id, label, type, metadata FROM nodes ORDER BY created_at DESC LIMIT ?`, limit,
	)
	if err != nil {
		return nil, nil, err
	}
	defer nodeRows.Close()

	var nodes []models.KnowledgeNode
	nodeIDs := make(map[string]bool)
	for nodeRows.Next() {
		var node models.KnowledgeNode
		var metaJSON sql.NullString
		if err := nodeRows.Scan(&node.ID, &node.Label, &node.Type, &metaJSON); err != nil {
			continue
		}
		if metaJSON.Valid {
			json.Unmarshal([]byte(metaJSON.String), &node.Metadata)
		}
		nodes = append(nodes, node)
		nodeIDs[node.ID] = true
	}

	// 查询边（仅包含两端节点都在结果中的边）
	edgeRows, err := g.db.QueryContext(ctx,
		`SELECT source_id, target_id, relation FROM edges LIMIT ?`, limit*3,
	)
	if err != nil {
		return nil, nil, err
	}
	defer edgeRows.Close()

	var edges []models.KnowledgeEdge
	for edgeRows.Next() {
		var edge models.KnowledgeEdge
		if err := edgeRows.Scan(&edge.SourceID, &edge.TargetID, &edge.Relation); err != nil {
			continue
		}
		if nodeIDs[edge.SourceID] && nodeIDs[edge.TargetID] {
			edges = append(edges, edge)
		}
	}

	return nodes, edges, nil
}

// UpdateFileStructure 更新项目文件结构到图谱
func (g *GraphDB) UpdateFileStructure(ctx context.Context, dirs, files map[string]string) error {
	tx, err := g.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Insert directory nodes
	for path, desc := range dirs {
		_, err = tx.ExecContext(ctx,
			`INSERT OR REPLACE INTO nodes (id, label, type, metadata) VALUES (?, ?, ?, ?)`,
			"dir:"+path, path, "directory", fmt.Sprintf(`{"purpose":"%s"}`, desc),
		)
		if err != nil {
			return fmt.Errorf("insert dir node: %w", err)
		}
	}

	// Insert file nodes
	for path, desc := range files {
		_, err = tx.ExecContext(ctx,
			`INSERT OR REPLACE INTO nodes (id, label, type, metadata) VALUES (?, ?, ?, ?)`,
			"file:"+path, path, "file", fmt.Sprintf(`{"purpose":"%s"}`, desc),
		)
		if err != nil {
			return fmt.Errorf("insert file node: %w", err)
		}

		// Create CONTAINS edge from parent dir to file
		parent := filepath.Dir(path)
		if parent != "." {
			_, err = tx.ExecContext(ctx,
				`INSERT OR IGNORE INTO edges (source_id, target_id, relation) VALUES (?, ?, ?)`,
				"dir:"+parent, "file:"+path, "CONTAINS",
			)
			if err != nil {
				return fmt.Errorf("insert contains edge: %w", err)
			}
		}
	}

	return tx.Commit()
}

// Close 关闭图数据库连接
func (g *GraphDB) Close() error {
	return g.db.Close()
}
