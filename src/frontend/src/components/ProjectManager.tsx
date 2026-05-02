import { useState, useEffect } from 'react'
import { ListProjects, AddProject, SwitchProject, RemoveProject, GetCurrentProject } from '../../bindings/ChronoDraftAEx/chronoservice.js'
import { useToast } from '../contexts/ToastContext'

interface Project {
  id: string
  name: string
  path: string
  description?: string
  is_active: boolean
  created_at: string
}

function ProjectManager() {
  const toast = useToast()
  const [projects, setProjects] = useState<Project[]>([])
  const [currentProject, setCurrentProject] = useState<Project | null>(null)
  const [showAddForm, setShowAddForm] = useState(false)
  const [newProjectName, setNewProjectName] = useState('')
  const [newProjectPath, setNewProjectPath] = useState('')
  const [newProjectDesc, setNewProjectDesc] = useState('')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')

  useEffect(() => {
    loadProjects()
  }, [])

  const loadProjects = async () => {
    try {
      const list = await ListProjects()
      setProjects(list || [])
      
      const current = await GetCurrentProject()
      if (current) {
        setCurrentProject({
          id: current.id,
          name: current.name,
          path: current.path,
          description: current.description || '',
          is_active: true,
          created_at: current.created_at || '',
        })
      }
    } catch (e) {
      toast.error('加载项目列表失败')
    }
  }

  const handleAddProject = async () => {
    if (!newProjectName.trim() || !newProjectPath.trim()) {
      setError('项目名称和路径不能为空')
      return
    }

    setLoading(true)
    setError('')
    try {
      await AddProject(newProjectName.trim(), newProjectPath.trim(), newProjectDesc.trim())
      setNewProjectName('')
      setNewProjectPath('')
      setNewProjectDesc('')
      setShowAddForm(false)
      await loadProjects()
    } catch (e: any) {
      setError(e.message || '添加项目失败')
    } finally {
      setLoading(false)
    }
  }

  const handleSwitchProject = async (projectId: string) => {
    setLoading(true)
    try {
      await SwitchProject(projectId)
      await loadProjects()
    } catch (e: any) {
      setError(e.message || '切换项目失败')
    } finally {
      setLoading(false)
    }
  }

  const handleRemoveProject = async (projectId: string) => {
    if (!confirm('确定要删除这个项目吗？')) return
    
    setLoading(true)
    try {
      await RemoveProject(projectId)
      await loadProjects()
    } catch (e: any) {
      setError(e.message || '删除项目失败')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="card">
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
        <div className="card-title">项目管理</div>
        <button 
          className="btn btn-primary" 
          onClick={() => setShowAddForm(!showAddForm)}
          disabled={loading}
        >
          {showAddForm ? '取消' : '➕ 添加项目'}
        </button>
      </div>

      {currentProject && (
        <div style={{ 
          background: 'rgba(88, 166, 255, 0.1)', 
          border: '1px solid var(--accent)', 
          borderRadius: 8, 
          padding: 12,
          marginBottom: 16 
        }}>
          <div style={{ fontWeight: 600, color: 'var(--accent)' }}>
            当前监控项目: {currentProject.name}
          </div>
          <div style={{ fontSize: 12, color: 'var(--text-secondary)', marginTop: 4 }}>
            {currentProject.path}
          </div>
        </div>
      )}

      {showAddForm && (
        <div style={{ 
          background: 'var(--bg-tertiary)', 
          borderRadius: 8, 
          padding: 16,
          marginBottom: 16 
        }}>
          <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
            <div>
              <label style={{ display: 'block', fontSize: 12, color: 'var(--text-secondary)', marginBottom: 4 }}>
                项目名称 *
              </label>
              <input
                className="input"
                placeholder="例如: MyWebApp"
                value={newProjectName}
                onChange={e => setNewProjectName(e.target.value)}
              />
            </div>
            <div>
              <label style={{ display: 'block', fontSize: 12, color: 'var(--text-secondary)', marginBottom: 4 }}>
                项目路径 *
              </label>
              <input
                className="input"
                placeholder="例如: C:\\Users\\name\\Projects\\MyWebApp"
                value={newProjectPath}
                onChange={e => setNewProjectPath(e.target.value)}
              />
            </div>
            <div>
              <label style={{ display: 'block', fontSize: 12, color: 'var(--text-secondary)', marginBottom: 4 }}>
                描述
              </label>
              <input
                className="input"
                placeholder="可选的项目描述"
                value={newProjectDesc}
                onChange={e => setNewProjectDesc(e.target.value)}
              />
            </div>
            {error && (
              <div style={{ color: 'var(--danger)', fontSize: 13 }}>{error}</div>
            )}
            <button 
              className="btn btn-primary" 
              onClick={handleAddProject}
              disabled={loading}
            >
              {loading ? '添加中...' : '确认添加'}
            </button>
          </div>
        </div>
      )}

      <div className="entry-list">
        {projects.length === 0 ? (
          <div style={{ textAlign: 'center', color: 'var(--text-secondary)', padding: 24 }}>
            暂无项目，点击上方按钮添加
          </div>
        ) : (
          projects.map(project => (
            <div 
              key={project.id} 
              className="entry-item"
              style={{
                borderColor: currentProject?.id === project.id ? 'var(--accent)' : undefined,
                background: currentProject?.id === project.id ? 'rgba(88, 166, 255, 0.05)' : undefined,
              }}
            >
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
                <div style={{ flex: 1 }}>
                  <div style={{ fontWeight: 600, fontSize: 14 }}>{project.name}</div>
                  <div style={{ fontSize: 12, color: 'var(--text-secondary)', marginTop: 2 }}>
                    {project.path}
                  </div>
                  {project.description && (
                    <div style={{ fontSize: 12, color: 'var(--text-secondary)', marginTop: 4 }}>
                      {project.description}
                    </div>
                  )}
                </div>
                <div style={{ display: 'flex', gap: 8 }}>
                  {currentProject?.id !== project.id && (
                    <button
                      className="btn btn-secondary"
                      style={{ padding: '4px 12px', fontSize: 12 }}
                      onClick={() => handleSwitchProject(project.id)}
                      disabled={loading}
                    >
                      切换
                    </button>
                  )}
                  <button
                    className="btn btn-secondary"
                    style={{ padding: '4px 12px', fontSize: 12, color: 'var(--danger)' }}
                    onClick={() => handleRemoveProject(project.id)}
                    disabled={loading}
                  >
                    删除
                  </button>
                </div>
              </div>
            </div>
          ))
        )}
      </div>
    </div>
  )
}

export default ProjectManager
