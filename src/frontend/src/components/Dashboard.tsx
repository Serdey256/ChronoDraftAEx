import { useState, useEffect } from 'react'
import { Call } from '@wailsio/runtime'
import { SearchKnowledge, GetCurrentProject, IndexProject, IsKnowledgeBaseEmpty, GenerateAgentsMD } from '../../bindings/ChronoDraftAEx/chronoservice.js'
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
  const [kbEmpty, setKbEmpty] = useState(false)
  const [indexPhase, setIndexPhase] = useState('')
  const [indexing, setIndexing] = useState(false)
  const [agentsMDEnabled, setAgentsMDEnabled] = useState(true)

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

  useEffect(() => {
    search('')
    loadCurrentProject()
    checkEmpty()
    ;(async () => {
      try {
        const v = await (Call as any).ByName('main.ChronoService.IsAgentsMDEnabled')
        setAgentsMDEnabled(!!v)
      } catch { setAgentsMDEnabled(true) }
    })()
  }, [])

  const checkEmpty = async () => {
    try {
      const empty = await IsKnowledgeBaseEmpty()
      setKbEmpty(empty)
    } catch (e) {
      // ignore
    }
  }

  const refresh = async () => {
    setLoading(true)
    try {
      const [results] = await Promise.all([SearchKnowledge('', 10), GenerateAgentsMD()])
      setEntries(results.map((r: any) => r.entry))
      const empty = await IsKnowledgeBaseEmpty()
      setKbEmpty(empty)
      toast.success('刷新完成')
    } catch (e: any) {
      toast.error('刷新失败: ' + (e.message || String(e)))
    } finally {
      setLoading(false)
    }
  }

  const indexProject = async () => {
    setIndexing(true)
    setIndexPhase('扫描文件')
    let poll: ReturnType<typeof setInterval> | undefined
    try {
      poll = setInterval(async () => {
        try {
          const p = await (Call as any).ByName('main.ChronoService.GetIndexPhase')
          if (p && typeof p === 'string' && p !== '') setIndexPhase(p)
        } catch {}
      }, 500)

      await IndexProject()
      clearInterval(poll)
      setIndexPhase('')
      toast.success('项目结构扫描完成')
      setKbEmpty(false)
      search('')
    } catch (e: any) {
      if (poll !== undefined) clearInterval(poll)
      toast.error('扫描项目结构失败: ' + (e.message || String(e)))
    } finally {
      setIndexing(false)
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
            知识库为空。Agent 完成代码修改后，变更会自动显示在这里。如需初始化项目结构，请点击下方按钮。
          </div>
          {indexing ? (
            <div style={{ textAlign: 'center' }}>
              <div style={{
                height: 4, background: 'var(--border)', borderRadius: 2,
                marginBottom: 8, overflow: 'hidden'
              }}>
                <div style={{
                  height: '100%', width: indexPhase === '生成AGENTS.md' ? '90%' : indexPhase === 'AI摘要' ? '70%' : indexPhase === 'AST代码分析' ? '50%' : indexPhase === '安装GitHook' ? '30%' : '10%',
                  background: 'var(--accent)', borderRadius: 2,
                  transition: 'width 0.5s ease'
                }} />
              </div>
              <span style={{ fontSize: 13, color: 'var(--text-secondary)' }}>
                {indexPhase || '扫描中...'}
              </span>
            </div>
          ) : (
            <button className="btn btn-primary" onClick={indexProject} disabled={loading}>
              {loading ? '扫描中，请稍候...' : '扫描项目结构'}
            </button>
          )}
          <div style={{ fontSize: 12, color: 'var(--text-secondary)', marginTop: 8 }}>Agent 可通过 record_change 工具上报代码变更</div>
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
          <button className="btn btn-secondary" onClick={refresh} disabled={loading}>
            {loading ? '刷新中...' : '🔄 刷新'}
          </button>
        </div>

        <div style={{ marginTop: 8, display: 'flex', alignItems: 'center', gap: 8 }}>
          <button
            className={agentsMDEnabled ? 'btn btn-primary' : 'btn btn-secondary'}
            onClick={async () => {
              const next = !agentsMDEnabled
              setAgentsMDEnabled(next)
              try { await (Call as any).ByName('main.ChronoService.SetAgentsMDEnabled', next) } catch {}
            }}
            style={{ fontSize: 12, padding: '4px 12px' }}
          >
            {agentsMDEnabled ? '📝 AGENTS.md: 开' : '📝 AGENTS.md: 关'}
          </button>
          <span style={{ fontSize: 11, color: 'var(--text-secondary)' }}>
            {agentsMDEnabled ? '每次变更后自动更新' : '已禁用自动生成，节省 token'}
          </span>
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
              Agent 完成代码修改后，变更会自动显示在这里
            </div>
          )}
        </div>
      </div>
    </div>
  )
}

export default Dashboard
