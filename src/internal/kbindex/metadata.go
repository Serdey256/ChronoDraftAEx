// Package kbindex 负责知识库索引更新
// metadata.go: SQLite 元数据存储封装，用于结构化元数据的主存储
package kbindex

import (
	"ChronoDraftAEx/pkg/models"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

// MetadataStore SQLite 元数据存储
type MetadataStore struct {
	db *sql.DB
}

// NewMetadataStore 创建元数据存储实例
func NewMetadataStore(dbPath string) (*MetadataStore, error) {
	db, err := sql.Open("sqlite", dbPath+"?_journal_mode=WAL")
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
	tags TEXT,
	timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS snapshots (
	id TEXT PRIMARY KEY,
	version TEXT,
	dependencies TEXT,
	metadata TEXT,
	timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS file_changes (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	entry_id TEXT NOT NULL,
	path TEXT NOT NULL,
	change_type TEXT NOT NULL,
	diff TEXT,
	FOREIGN KEY (entry_id) REFERENCES entries(id)
);

CREATE TABLE IF NOT EXISTS commits (
	hash TEXT PRIMARY KEY,
	message TEXT NOT NULL,
	author TEXT,
	timestamp DATETIME NOT NULL,
	files TEXT,
	insertions INTEGER DEFAULT 0,
	deletions INTEGER DEFAULT 0
);

CREATE TABLE IF NOT EXISTS code_entities (
	file_path TEXT NOT NULL,
	entity_type TEXT NOT NULL,
	name TEXT NOT NULL,
	signature TEXT,
	metadata TEXT,
	PRIMARY KEY (file_path, name, entity_type)
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
	tagsJSON, err := json.Marshal(entry.Tags)
	if err != nil {
		return fmt.Errorf("序列化 tags 失败: %w", err)
	}

	tx, err := m.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec(
		`INSERT OR REPLACE INTO entries (id, session_id, summary, design_decision, impact_analysis, tags, timestamp)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		entry.ID,
		entry.SessionID,
		entry.Summary,
		entry.DesignDecision,
		entry.ImpactAnalysis,
		string(tagsJSON),
		entry.Timestamp.Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("插入条目失败: %w", err)
	}

	for _, fc := range entry.AffectedFiles {
		_, err = tx.Exec(
			`INSERT INTO file_changes (entry_id, path, change_type, diff) VALUES (?, ?, ?, ?)`,
			entry.ID, fc.Path, fc.ChangeType, fc.Diff,
		)
		if err != nil {
			return fmt.Errorf("插入文件变更记录失败: %w", err)
		}
	}

	return tx.Commit()
}

// GetEntryByID 根据 ID 查询条目
func (m *MetadataStore) GetEntryByID(id string) (*models.StructuredEntry, error) {
	row := m.db.QueryRow(
		`SELECT id, session_id, summary, design_decision, impact_analysis, tags, timestamp
		 FROM entries WHERE id = ?`, id,
	)

	var entry models.StructuredEntry
	var tagsJSON string
	var ts string
	err := row.Scan(&entry.ID, &entry.SessionID, &entry.Summary, &entry.DesignDecision,
		&entry.ImpactAnalysis, &tagsJSON, &ts)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("条目 %s 不存在", id)
		}
		return nil, err
	}

	if err := json.Unmarshal([]byte(tagsJSON), &entry.Tags); err != nil {
		entry.Tags = []string{}
	}
	entry.Timestamp, _ = time.Parse(time.RFC3339, ts)

	entry.AffectedFiles, err = m.getFileChanges(entry.ID)
	if err != nil {
		return nil, err
	}

	return &entry, nil
}

// ListEntries 分页列出条目（按时间倒序）
func (m *MetadataStore) ListEntries(offset, limit int) ([]models.StructuredEntry, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := m.db.Query(
		`SELECT id, session_id, summary, design_decision, impact_analysis, tags, timestamp
		 FROM entries ORDER BY timestamp DESC LIMIT ? OFFSET ?`, limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []models.StructuredEntry
	for rows.Next() {
		var entry models.StructuredEntry
		var tagsJSON string
		var ts string
		if err := rows.Scan(&entry.ID, &entry.SessionID, &entry.Summary, &entry.DesignDecision,
			&entry.ImpactAnalysis, &tagsJSON, &ts); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(tagsJSON), &entry.Tags); err != nil {
			entry.Tags = []string{}
		}
		entry.Timestamp, _ = time.Parse(time.RFC3339, ts)
		entry.AffectedFiles, _ = m.getFileChanges(entry.ID)
		entries = append(entries, entry)
	}
	return entries, rows.Err()
}

// SaveSnapshot 保存项目快照
func (m *MetadataStore) SaveSnapshot(snapshot *models.ProjectSnapshot) error {
	depsJSON, err := json.Marshal(snapshot.Dependencies)
	if err != nil {
		return fmt.Errorf("序列化 dependencies 失败: %w", err)
	}
	metaJSON, err := json.Marshal(snapshot.Metadata)
	if err != nil {
		return fmt.Errorf("序列化 metadata 失败: %w", err)
	}

	_, err = m.db.Exec(
		`INSERT OR REPLACE INTO snapshots (id, version, dependencies, metadata, timestamp)
		 VALUES (?, ?, ?, ?, ?)`,
		snapshot.ID,
		snapshot.Version,
		string(depsJSON),
		string(metaJSON),
		snapshot.Timestamp.Format(time.RFC3339),
	)
	return err
}

// ListSnapshots 列出所有快照（按时间倒序）
func (m *MetadataStore) ListSnapshots() ([]models.ProjectSnapshot, error) {
	rows, err := m.db.Query(
		`SELECT id, version, dependencies, metadata, timestamp FROM snapshots ORDER BY timestamp DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var snapshots []models.ProjectSnapshot
	for rows.Next() {
		var snap models.ProjectSnapshot
		var depsJSON, metaJSON, ts string
		if err := rows.Scan(&snap.ID, &snap.Version, &depsJSON, &metaJSON, &ts); err != nil {
			return nil, err
		}
		json.Unmarshal([]byte(depsJSON), &snap.Dependencies)
		json.Unmarshal([]byte(metaJSON), &snap.Metadata)
		snap.Timestamp, _ = time.Parse(time.RFC3339, ts)
		snapshots = append(snapshots, snap)
	}
	return snapshots, rows.Err()
}

// getFileChanges 获取某条目关联的文件变更
func (m *MetadataStore) getFileChanges(entryID string) ([]models.FileChange, error) {
	rows, err := m.db.Query(
		`SELECT path, change_type, diff FROM file_changes WHERE entry_id = ?`, entryID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var changes []models.FileChange
	for rows.Next() {
		var fc models.FileChange
		var diff sql.NullString
		if err := rows.Scan(&fc.Path, &fc.ChangeType, &diff); err != nil {
			return nil, err
		}
		if diff.Valid {
			fc.Diff = diff.String
		}
		changes = append(changes, fc)
	}
	return changes, rows.Err()
}

// SaveCommit 保存 git commit 记录（INSERT OR REPLACE 实现幂等）
func (m *MetadataStore) SaveCommit(hash, message, author, timestamp, files string, ins, dels int) error {
	_, err := m.db.Exec(
		`INSERT OR REPLACE INTO commits (hash, message, author, timestamp, files, insertions, deletions)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		hash, message, author, timestamp, files, ins, dels,
	)
	if err != nil {
		return fmt.Errorf("保存 commit 记录失败: %w", err)
	}
	return nil
}

// ListCommits 列出 git commit 记录（按时间倒序）
func (m *MetadataStore) ListCommits(limit int) ([]models.CommitRecord, error) {
	if limit <= 0 {
		limit = 10
	}
	rows, err := m.db.Query(
		`SELECT hash, message, author, timestamp, files, insertions, deletions
		 FROM commits ORDER BY timestamp DESC LIMIT ?`, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var commits []models.CommitRecord
	for rows.Next() {
		var c models.CommitRecord
		if err := rows.Scan(&c.Hash, &c.Message, &c.Author, &c.Timestamp, &c.Files, &c.Insertions, &c.Deletions); err != nil {
			return nil, err
		}
		commits = append(commits, c)
	}
	return commits, rows.Err()
}

// SaveCodeEntities 保存文件的所有代码实体（事务：先删除旧记录，再批量插入）
func (m *MetadataStore) SaveCodeEntities(filePath string, entities []models.CodeEntity) error {
	tx, err := m.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec(`DELETE FROM code_entities WHERE file_path = ?`, filePath)
	if err != nil {
		return fmt.Errorf("删除旧代码实体失败: %w", err)
	}

	stmt, err := tx.Prepare(
		`INSERT OR REPLACE INTO code_entities (file_path, entity_type, name, signature, metadata) VALUES (?, ?, ?, ?, ?)`,
	)
	if err != nil {
		return fmt.Errorf("预编译插入语句失败: %w", err)
	}
	defer stmt.Close()

	for _, e := range entities {
		_, err = stmt.Exec(e.FilePath, e.EntityType, e.Name, e.Signature, e.Metadata)
		if err != nil {
			return fmt.Errorf("插入代码实体失败: %w", err)
		}
	}

	return tx.Commit()
}

// GetCodeEntities 查询指定文件的所有代码实体
func (m *MetadataStore) GetCodeEntities(filePath string) ([]models.CodeEntity, error) {
	rows, err := m.db.Query(
		`SELECT file_path, entity_type, name, signature, metadata FROM code_entities WHERE file_path = ?`,
		filePath,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entities []models.CodeEntity
	for rows.Next() {
		var e models.CodeEntity
		var sig, meta sql.NullString
		if err := rows.Scan(&e.FilePath, &e.EntityType, &e.Name, &sig, &meta); err != nil {
			return nil, err
		}
		if sig.Valid {
			e.Signature = sig.String
		}
		if meta.Valid {
			e.Metadata = meta.String
		}
		entities = append(entities, e)
	}
	if entities == nil {
		entities = []models.CodeEntity{}
	}
	return entities, rows.Err()
}

// UpdateCodeEntityMetadata 更新指定代码实体的 metadata
func (m *MetadataStore) UpdateCodeEntityMetadata(filePath, name, entityType, metadata string) error {
	_, err := m.db.Exec(
		`UPDATE code_entities SET metadata = ? WHERE file_path = ? AND name = ? AND entity_type = ?`,
		metadata, filePath, name, entityType,
	)
	return err
}

// DeleteCodeEntities 删除指定文件的所有代码实体
func (m *MetadataStore) DeleteCodeEntities(filePath string) error {
	_, err := m.db.Exec(`DELETE FROM code_entities WHERE file_path = ?`, filePath)
	if err != nil {
		return fmt.Errorf("删除代码实体失败: %w", err)
	}
	return nil
}

// Close 关闭数据库连接
func (m *MetadataStore) Close() error {
	return m.db.Close()
}

// ListCodeEntityFiles 返回所有已分析的文件路径（去重）
func (m *MetadataStore) ListCodeEntityFiles() ([]string, error) {
	rows, err := m.db.Query(`SELECT DISTINCT file_path FROM code_entities ORDER BY file_path`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []string
	for rows.Next() {
		var path string
		if err := rows.Scan(&path); err != nil {
			return nil, err
		}
		files = append(files, path)
	}
	return files, rows.Err()
}

// GetAllCodeEntities 返回所有代码实体
func (m *MetadataStore) GetAllCodeEntities() ([]models.CodeEntity, error) {
	rows, err := m.db.Query(`SELECT file_path, entity_type, name, signature, metadata FROM code_entities ORDER BY file_path, entity_type`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entities []models.CodeEntity
	for rows.Next() {
		var e models.CodeEntity
		var sig, meta sql.NullString
		if err := rows.Scan(&e.FilePath, &e.EntityType, &e.Name, &sig, &meta); err != nil {
			return nil, err
		}
		if sig.Valid {
			e.Signature = sig.String
		}
		if meta.Valid {
			e.Metadata = meta.String
		}
		entities = append(entities, e)
	}
	if entities == nil {
		entities = []models.CodeEntity{}
	}
	return entities, rows.Err()
}
