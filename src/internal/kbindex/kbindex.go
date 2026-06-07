// Package kbindex 负责知识库索引更新
// kbindex.go: 将向量数据库、图数据库和元数据存储整合为统一的知识库索引服务
package kbindex

import (
	"ChronoDraftAEx/pkg/models"
	"ChronoDraftAEx/pkg/utils"
	"context"
	"fmt"
	"path/filepath"
)

// KBIndex 知识库索引管理器
type KBIndex struct {
	vectorDB *VectorDB
	graphDB  *GraphDB
	metaDB   *MetadataStore
	dataDir  string
}

// NewKBIndex 创建知识库索引管理器
func NewKBIndex(projectRoot string) (*KBIndex, error) {
	dataDir := utils.GetDataDir(projectRoot)
	if err := utils.EnsureDir(dataDir); err != nil {
		return nil, err
	}

	vectorDB, err := NewVectorDB(filepath.Join(dataDir, "vectors.db"))
	if err != nil {
		return nil, fmt.Errorf("初始化向量数据库失败: %w", err)
	}

	graphDB, err := NewGraphDB(filepath.Join(dataDir, "graph.db"))
	if err != nil {
		return nil, fmt.Errorf("初始化图数据库失败: %w", err)
	}

	metaDB, err := NewMetadataStore(filepath.Join(dataDir, "metadata.db"))
	if err != nil {
		return nil, fmt.Errorf("初始化元数据存储失败: %w", err)
	}

	return &KBIndex{
		vectorDB: vectorDB,
		graphDB:  graphDB,
		metaDB:   metaDB,
		dataDir:  dataDir,
	}, nil
}

// SetEmbeddingConfig 配置嵌入模型 API
func (k *KBIndex) SetEmbeddingConfig(apiKey, apiBase, model string) {
	k.vectorDB.SetEmbeddingConfig(apiKey, apiBase, model)
}

// Init 初始化所有数据库的 Schema
func (k *KBIndex) Init(ctx context.Context) error {
	if err := k.vectorDB.InitSchema(ctx); err != nil {
		return fmt.Errorf("向量库初始化失败: %w", err)
	}
	if err := k.graphDB.InitSchema(ctx); err != nil {
		return fmt.Errorf("图数据库初始化失败: %w", err)
	}
	if err := k.metaDB.InitSchema(); err != nil {
		return fmt.Errorf("元数据存储初始化失败: %w", err)
	}
	return nil
}

// IndexEntry 将结构化知识条目索引到所有存储后端
func (k *KBIndex) IndexEntry(ctx context.Context, entry *models.StructuredEntry) error {
	if err := k.metaDB.SaveEntry(entry); err != nil {
		return fmt.Errorf("保存元数据失败: %w", err)
	}
	if err := k.vectorDB.InsertEntry(ctx, entry); err != nil {
		return fmt.Errorf("插入向量库失败: %w", err)
	}
	if err := k.graphDB.InsertEntry(ctx, entry); err != nil {
		return fmt.Errorf("插入图数据库失败: %w", err)
	}
	return nil
}

// Search 综合搜索：语义搜索 + 图关系查询
func (k *KBIndex) Search(ctx context.Context, query string, topK int) ([]models.SearchResult, error) {
	// 1. 向量语义搜索
	results, err := k.vectorDB.Search(ctx, query, topK)
	if err != nil {
		return nil, fmt.Errorf("向量搜索失败: %w", err)
	}

	// 2. 对每条结果补充图关系信息
	for i := range results {
		nodes, _, err := k.graphDB.QueryRelated(ctx, results[i].Entry.ID)
		if err == nil {
			results[i].NodePath = nodes
		}
	}

	return results, nil
}

// GetGraphData 获取图谱全量数据
func (k *KBIndex) GetGraphData(ctx context.Context, limit int) ([]models.KnowledgeNode, []models.KnowledgeEdge, error) {
	return k.graphDB.GetGraphData(ctx, limit)
}

// QueryRelated 查询与指定条目相关的知识节点
func (k *KBIndex) QueryRelated(ctx context.Context, entryID string) ([]models.KnowledgeNode, []models.KnowledgeEdge, error) {
	return k.graphDB.QueryRelated(ctx, entryID)
}

// ListEntries 列出知识条目
func (k *KBIndex) ListEntries(offset, limit int) ([]models.StructuredEntry, error) {
	return k.metaDB.ListEntries(offset, limit)
}

// DeleteEntry 从所有索引后端删除知识条目，并返回删除前的完整条目用于撤销
func (k *KBIndex) DeleteEntry(ctx context.Context, entryID string) (*models.StructuredEntry, error) {
	entry, err := k.metaDB.GetEntryByID(entryID)
	if err != nil {
		return nil, err
	}
	rollback := func(cause error) (*models.StructuredEntry, error) {
		if restoreErr := k.IndexEntry(ctx, entry); restoreErr != nil {
			return nil, fmt.Errorf("%w; 回滚恢复失败: %v", cause, restoreErr)
		}
		return nil, cause
	}

	if err := k.vectorDB.DeleteEntry(ctx, entryID); err != nil {
		return nil, err
	}
	if err := k.graphDB.DeleteEntry(ctx, entryID); err != nil {
		return rollback(err)
	}
	if err := k.metaDB.DeleteEntry(entryID); err != nil {
		return rollback(err)
	}
	return entry, nil
}

// RestoreEntry 将删除前的知识条目重新索引到所有后端
func (k *KBIndex) RestoreEntry(ctx context.Context, entry *models.StructuredEntry) error {
	return k.IndexEntry(ctx, entry)
}

// SaveSnapshot 保存项目快照
func (k *KBIndex) SaveSnapshot(snapshot *models.ProjectSnapshot) error {
	return k.metaDB.SaveSnapshot(snapshot)
}

// ListSnapshots 列出所有快照
func (k *KBIndex) ListSnapshots() ([]models.ProjectSnapshot, error) {
	return k.metaDB.ListSnapshots()
}

// SaveCommit 保存 git commit 记录
func (k *KBIndex) SaveCommit(hash, message, author, timestamp, files string, ins, dels int) error {
	return k.metaDB.SaveCommit(hash, message, author, timestamp, files, ins, dels)
}

// ListCommits 列出 git commit 记录
func (k *KBIndex) ListCommits(limit int) ([]models.CommitRecord, error) {
	return k.metaDB.ListCommits(limit)
}

// SaveCodeEntities 保存文件的所有代码实体
func (k *KBIndex) SaveCodeEntities(filePath string, entities []models.CodeEntity) error {
	return k.metaDB.SaveCodeEntities(filePath, entities)
}

// GetCodeEntities 查询指定文件的所有代码实体
func (k *KBIndex) GetCodeEntities(filePath string) ([]models.CodeEntity, error) {
	return k.metaDB.GetCodeEntities(filePath)
}

// DeleteCodeEntities 删除指定文件的所有代码实体
func (k *KBIndex) DeleteCodeEntities(filePath string) error {
	return k.metaDB.DeleteCodeEntities(filePath)
}

// ListCodeEntityFiles 返回所有已分析的文件路径
func (k *KBIndex) ListCodeEntityFiles() ([]string, error) {
	return k.metaDB.ListCodeEntityFiles()
}

// UpdateCodeEntityMetadata 更新指定代码实体的 metadata
func (k *KBIndex) UpdateCodeEntityMetadata(filePath, name, entityType, metadata string) error {
	return k.metaDB.UpdateCodeEntityMetadata(filePath, name, entityType, metadata)
}

// GetAllCodeEntities 返回所有代码实体
func (k *KBIndex) GetAllCodeEntities() ([]models.CodeEntity, error) {
	return k.metaDB.GetAllCodeEntities()
}

// Close 关闭所有数据库连接
func (k *KBIndex) Close() error {
	_ = k.vectorDB.Close()
	_ = k.graphDB.Close()
	_ = k.metaDB.Close()
	return nil
}

// UpdateDirectoryHierarchy 从文件路径列表构建目录层级图
func (k *KBIndex) UpdateDirectoryHierarchy(ctx context.Context, filePaths []string) error {
	return k.graphDB.UpdateDirectoryHierarchy(ctx, filePaths)
}

// UpdateImportEdges 批量创建文件间的 IMPORTS 边
func (k *KBIndex) UpdateImportEdges(ctx context.Context, imports map[string][]string) error {
	return k.graphDB.UpdateImportEdges(ctx, imports)
}

// GetModuleGraph 查询指定目录下的子图
func (k *KBIndex) GetModuleGraph(ctx context.Context, dirPath string, limit int) ([]models.KnowledgeNode, []models.KnowledgeEdge, error) {
	return k.graphDB.GetModuleGraph(ctx, dirPath, limit)
}

// SearchNodesByLabel 按标签模糊搜索图节点
func (k *KBIndex) SearchNodesByLabel(ctx context.Context, query string, limit int) ([]models.KnowledgeNode, error) {
	return k.graphDB.SearchNodesByLabel(ctx, query, limit)
}
