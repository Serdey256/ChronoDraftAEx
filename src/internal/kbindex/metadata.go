// Package kbindex 负责知识库索引更新
// metadata.go: SQLite 元数据存储封装，用于结构化元数据的主存储
package kbindex

import (
	"ChronoDraftAEx/pkg/models"
	"database/sql"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
)

// MetadataStore SQLite 元数据存储
type MetadataStore struct {
	db *sql.DB
}

// NewMetadataStore 创建元数据存储实例
func NewMetadataStore(dbPath string) (*MetadataStore, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}
	return &MetadataStore{db: db}, nil
}

// InitSchema 初始化 SQLite 表结构
func (m *MetadataStore) InitSchema() error {
	schema := `
CREATE TABLE IF NOT EXISTS entries (
	id TEXT PRIMARY KEY,
	session_id TEXT NOT NULL,
	summary TEXT,
	design_decision TEXT,
	impact_analysis TEXT,
	tags TEXT, -- JSON array
	timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS snapshots (
	id TEXT PRIMARY KEY,
	version TEXT,
	dependencies TEXT, -- JSON array
	metadata TEXT,     -- JSON object
	timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS file_changes (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	entry_id TEXT NOT NULL,
	path TEXT NOT NULL,
	change_type TEXT NOT NULL,
	FOREIGN KEY (entry_id) REFERENCES entries(id)
);

CREATE INDEX IF NOT EXISTS idx_entries_session ON entries(session_id);
CREATE INDEX IF NOT EXISTS idx_entries_time ON entries(timestamp);
CREATE INDEX IF NOT EXISTS idx_file_changes_entry ON file_changes(entry_id);
`
	_, err := m.db.Exec(schema)
	return err
}

// SaveEntry 保存结构化条目到 SQLite
func (m *MetadataStore) SaveEntry(entry *models.StructuredEntry) error {
	// TODO: 实现插入逻辑，包括 tags JSON 序列化
	_ = entry
	return nil
}

// GetEntryByID 根据 ID 查询条目
func (m *MetadataStore) GetEntryByID(id string) (*models.StructuredEntry, error) {
	// TODO: 实现查询逻辑
	_ = id
	return nil, fmt.Errorf("not implemented")
}

// ListEntries 分页列出条目
func (m *MetadataStore) ListEntries(offset, limit int) ([]models.StructuredEntry, error) {
	// TODO: 实现分页查询
	_ = offset
	_ = limit
	return nil, fmt.Errorf("not implemented")
}

// SaveSnapshot 保存项目快照
func (m *MetadataStore) SaveSnapshot(snapshot *models.ProjectSnapshot) error {
	// TODO: 实现快照保存
	_ = snapshot
	return nil
}

// Close 关闭数据库连接
func (m *MetadataStore) Close() error {
	return m.db.Close()
}
