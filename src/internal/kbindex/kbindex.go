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

	vectorDB, err := NewVectorDB(filepath.Join(dataDir, "vectors.lance"))
	if err != nil {
		return nil, fmt.Errorf("初始化向量数据库失败: %w", err)
	}

	graphDB, err := NewGraphDB(filepath.Join(dataDir, "graph.kuzu"))
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

// SaveSnapshot 保存项目快照
func (k *KBIndex) SaveSnapshot(snapshot *models.ProjectSnapshot) error {
	return k.metaDB.SaveSnapshot(snapshot)
}

// Close 关闭所有数据库连接
func (k *KBIndex) Close() error {
	_ = k.vectorDB.Close()
	_ = k.graphDB.Close()
	_ = k.metaDB.Close()
	return nil
}
