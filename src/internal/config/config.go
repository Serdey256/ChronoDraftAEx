// Package config 提供项目配置的持久化管理
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// ProjectConfig 表示一个被监控的项目
type ProjectConfig struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Path        string `json:"path"`
	Description string `json:"description,omitempty"`
	IsActive    bool   `json:"is_active"`
	CreatedAt   string `json:"created_at"`
}

// AppConfig 应用全局配置
type AppConfig struct {
	Projects   []ProjectConfig `json:"projects"`
	ActiveID   string          `json:"active_id"`
	LastAPIKey string          `json:"last_api_key,omitempty"`
}

// Manager 配置管理器
type Manager struct {
	configPath string
	config     *AppConfig
}

// NewManager 创建配置管理器
func NewManager(configDir string) *Manager {
	return &Manager{
		configPath: filepath.Join(configDir, "config.json"),
		config:     &AppConfig{Projects: []ProjectConfig{}},
	}
}

// Load 从文件加载配置
func (m *Manager) Load() error {
	data, err := os.ReadFile(m.configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // 文件不存在，使用默认空配置
		}
		return fmt.Errorf("读取配置文件失败: %w", err)
	}

	if err := json.Unmarshal(data, m.config); err != nil {
		return fmt.Errorf("解析配置文件失败: %w", err)
	}
	return nil
}

// Save 保存配置到文件
func (m *Manager) Save() error {
	if err := os.MkdirAll(filepath.Dir(m.configPath), 0755); err != nil {
		return fmt.Errorf("创建配置目录失败: %w", err)
	}

	data, err := json.MarshalIndent(m.config, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化配置失败: %w", err)
	}

	if err := os.WriteFile(m.configPath, data, 0644); err != nil {
		return fmt.Errorf("写入配置文件失败: %w", err)
	}
	return nil
}

// GetConfig 获取当前配置
func (m *Manager) GetConfig() *AppConfig {
	return m.config
}

// AddProject 添加新项目
func (m *Manager) AddProject(project ProjectConfig) error {
	// 检查路径是否已存在
	for _, p := range m.config.Projects {
		if p.Path == project.Path {
			return fmt.Errorf("项目路径已存在: %s", project.Path)
		}
	}

	m.config.Projects = append(m.config.Projects, project)
	
	// 如果是第一个项目，自动设为活跃
	if len(m.config.Projects) == 1 {
		m.config.ActiveID = project.ID
	}

	return m.Save()
}

// RemoveProject 删除项目
func (m *Manager) RemoveProject(id string) error {
	for i, p := range m.config.Projects {
		if p.ID == id {
			m.config.Projects = append(m.config.Projects[:i], m.config.Projects[i+1:]...)
			
			// 如果删除的是当前活跃项目，清空活跃ID
			if m.config.ActiveID == id {
				m.config.ActiveID = ""
				if len(m.config.Projects) > 0 {
					m.config.ActiveID = m.config.Projects[0].ID
				}
			}
			
			return m.Save()
		}
	}
	return fmt.Errorf("项目不存在: %s", id)
}

// SetActiveProject 设置活跃项目
func (m *Manager) SetActiveProject(id string) error {
	found := false
	for _, p := range m.config.Projects {
		if p.ID == id {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("项目不存在: %s", id)
	}

	m.config.ActiveID = id
	return m.Save()
}

// GetActiveProject 获取当前活跃项目
func (m *Manager) GetActiveProject() *ProjectConfig {
	if m.config.ActiveID == "" {
		return nil
	}
	for i := range m.config.Projects {
		if m.config.Projects[i].ID == m.config.ActiveID {
			return &m.config.Projects[i]
		}
	}
	return nil
}

// GetProjectByID 根据 ID 获取项目
func (m *Manager) GetProjectByID(id string) *ProjectConfig {
	for i := range m.config.Projects {
		if m.config.Projects[i].ID == id {
			return &m.config.Projects[i]
		}
	}
	return nil
}

// ListProjects 列出所有项目
func (m *Manager) ListProjects() []ProjectConfig {
	return m.config.Projects
}
