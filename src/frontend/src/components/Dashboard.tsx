import { useState, useEffect, useRef } from 'react'
import { SearchKnowledge, CaptureChanges, GetCurrentProject, StartWatcher, StopWatcher, IsWatcherRunning, FullIndex, IsKnowledgeBaseEmpty } from '../../bindings/ChronoDraftAEx/chronoservice.js'
import { useToast } from '../contexts/ToastContext'
import type { StructuredEntry } from '../types'

interface ProjectInfo {
  id: string
  name: string
  path: string
}

function Dashboard() {
  const toast = useToast()
  const [entries, setEntries] = useState<StructuredEntry[]>([])
  const [query, setQuery] = useState('')
  const [loading, setLoading] = useState(false)
  const [currentProject, setCurrentProject] = useState<ProjectInfo | null>(null)
  const [watcherRunning, setWatcherRunning] = useState(false)
  const [kbEmpty, setKbEmpty] = useState(false)
  const autoRefreshRef = useRef<ReturnType<typeof setInterval> | null>(null)

  const search = async (q: string) => {
    setLoading(true)
    try {
      const results = await SearchKnowledge(q, 10)
      setEntries(results.map((r: any) => r.entry))
    } catch (e) {
      toast.error('搜索失败: ' + (e instanceof Error ? e.message : String(e)))
    } finally {
      setLoading(false)
    }
  }

  const capture = async () => {
    setLoading(true)
    try {
      const entry = await CaptureChanges('manual-' + Date.now())
      if (entry) {
        setEntries(prev => [{
          id: entry.id,
          session_id: entry.session_id,
          timestamp: entry.timestamp ? new Date(entry.timestamp).toISOString() : new Date().toISOString(),
          summary: entry.summary,
          design_decision: entry.design_decision,
          impact_analysis: entry.impact_analysis,
          affected_files: entry.affected_files || [],
          tags: entry.tags || [],
        }, ...prev])
      }
    } catch (e) {
      toast.error('捕获变更失败: ' + (e instanceof Error ? e.message : String(e)))
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    search('')
    loadCurrentProject()
    checkEmpty()
  }, [])

  const checkEmpty = async () => {
    try {
      const empty = await IsKnowledgeBaseEmpty()
      setKbEmpty(empty)
    } catch (e) {
      // ignore
    }
  }

  const fullIndex = async () => {
    setLoading(true)
    try {
      await FullIndex()
      toast.success('全量索引完成')
      setKbEmpty(false)
      search('')
    } catch (e: any) {
      toast.error('全量索引失败: ' + (e.message || String(e)))
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    const pollStatus = () => {
      IsWatcherRunning().then(setWatcherRunning).catch(() => setWatcherRunning(false))
    }
    pollStatus()
    const statusInterval = setInterval(pollStatus, 5000)
    return () => clearInterval(statusInterval)
  }, [])

  useEffect(() => {
    if (watcherRunning) {
      autoRefreshRef.current = setInterval(() => {
        search('')
      }, 5000)
    } else if (autoRefreshRef.current) {
      clearInterval(autoRefreshRef.current)
      autoRefreshRef.current = null
    }
    return () => {
      if (autoRefreshRef.current) {
        clearInterval(autoRefreshRef.current)
        autoRefreshRef.current = null
      }
    }
  }, [watcherRunning])

  const toggleWatcher = async () => {
    try {
      if (watcherRunning) {
        await StopWatcher()
      } else {
        await StartWatcher()
      }
      const running = await IsWatcherRunning()
      setWatcherRunning(running)
    } catch (e) {
      toast.error('切换监控状态失败: ' + (e instanceof Error ? e.message : String(e)))
    }
  }

  const loadCurrentProject = async () => {
    try {
      const project = await GetCurrentProject()
      if (project) {
        setCurrentProject({
          id: project.id,
          name: project.name,
          path: project.path,
        })
      }
    } catch (e) {
      toast.error('加载当前项目失败: ' + (e instanceof Error ? e.message : String(e)))
    }
  }

  return (
    <div>
      <div className="page-header">
        <h2>📊 仪表盘</h2>
        <p>查看项目知识库的近期变更与全局概览</p>
        {currentProject && (
          <div style={{ 
            marginTop: 8, 
            padding: '8px 12px', 
            background: 'rgba(88, 166, 255, 0.1)', 
            borderRadius: 6,
            border: '1px solid var(--accent)',
            display: 'inline-block'
          }}>
            <span style={{ fontSize: 12, color: 'var(--text-secondary)' }}>当前监控: </span>
            <span style={{ fontSize: 13, fontWeight: 600, color: 'var(--accent)' }}>{currentProject.name}</span>
            <span style={{ fontSize: 11, color: 'var(--text-secondary)', marginLeft: 8 }}>{currentProject.path}</span>
          </div>
        )}
      </div>

      <div className="grid" style={{ marginBottom: 24 }}>
        <div className="card">
          <div className="card-title">知识条目总数</div>
          <div style={{ fontSize: 28, fontWeight: 700, color: 'var(--accent)' }}>{entries.length}</div>
        </div>
        <div className="card">
          <div className="card-title">今日新增</div>
          <div style={{ fontSize: 28, fontWeight: 700, color: 'var(--success)' }}>
            {entries.filter(e => new Date(e.timestamp).toDateString() === new Date().toDateString()).length}
          </div>
        </div>
        <div className="card">
          <div className="card-title">MCP 服务状态</div>
          <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginTop: 8 }}>
            <span className="dot online" />
            <span style={{ fontSize: 14, color: 'var(--text-secondary)' }}>运行中 :8787</span>
          </div>
        </div>
      </div>

      {kbEmpty && (
        <div style={{
          background: 'rgba(210, 153, 34, 0.1)',
          border: '1px solid #d29922',
          borderRadius: 8,
          padding: 16,
          marginBottom: 16,
          textAlign: 'center'
        }}>
          <div style={{ fontSize: 14, color: '#d29922', marginBottom: 8 }}>
            知识库为空，点击下方按钮建立初始索引
          </div>
          <button className="btn btn-primary" onClick={fullIndex} disabled={loading}>
            {loading ? '索引中，请稍候...' : '📦 全量索引'}
          </button>
        </div>
      )}

      <div className="card" style={{ marginBottom: 24 }}>
        <div style={{ display: 'flex', gap: 12, marginBottom: 16 }}>
          <input
            className="input"
            placeholder="输入关键词搜索知识库..."
            value={query}
            onChange={e => setQuery(e.target.value)}
            onKeyDown={e => e.key === 'Enter' && search(query)}
            style={{ flex: 1 }}
          />
          <button className="btn btn-primary" onClick={() => search(query)} disabled={loading}>
            {loading ? '搜索中...' : '搜索'}
          </button>
          <button className="btn btn-secondary" onClick={capture} disabled={loading}>
            {loading ? '捕获中...' : '📥 捕获变更'}
          </button>
          <button
            className={`btn ${watcherRunning ? 'btn-danger' : 'btn-secondary'} watcher-btn`}
            onClick={toggleWatcher}
          >
            <span className={`watcher-dot ${watcherRunning ? 'watcher-active' : 'watcher-stopped'}`} />
            {watcherRunning ? '停止监控' : '启动监控'}
          </button>
        </div>

        <div className="entry-list">
          {entries.map(entry => (
            <div className="entry-item" key={entry.id}>
              <div className="entry-meta">
                <span>{new Date(entry.timestamp).toLocaleString()}</span>
                <span>·</span>
                <span>Session: {entry.session_id}</span>
              </div>
              <h4 style={{ fontSize: 15, fontWeight: 600, marginBottom: 8 }}>{entry.summary}</h4>
              <p style={{ color: 'var(--text-secondary)', fontSize: 13, marginBottom: 6 }}>
                <strong>设计决策：</strong>{entry.design_decision}
              </p>
              <p style={{ color: 'var(--text-secondary)', fontSize: 13 }}>
                <strong>影响面：</strong>{entry.impact_analysis}
              </p>
              <div className="entry-tags">
                {entry.tags.map(tag => (
                  <span className="tag" key={tag}>{tag}</span>
                ))}
              </div>
            </div>
          ))}
          {entries.length === 0 && !loading && (
            <div style={{ textAlign: 'center', color: 'var(--text-secondary)', padding: 40 }}>
              暂无知识条目，点击「捕获变更」开始记录
            </div>
          )}
        </div>
      </div>
    </div>
  )
}

export default Dashboard
