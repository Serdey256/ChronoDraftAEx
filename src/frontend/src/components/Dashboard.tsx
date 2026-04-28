import { useState, useEffect } from 'react'
import type { StructuredEntry, SearchResult } from '../types'

// Wails 运行时绑定将在构建时注入 window.go
// 开发环境中使用 mock 数据
declare global {
  interface Window {
    go?: {
      main: {
        App: {
          SearchKnowledge(query: string, topK: number): Promise<SearchResult[]>
          CaptureChanges(sessionID: string): Promise<StructuredEntry>
        }
      }
    }
  }
}

function Dashboard() {
  const [entries, setEntries] = useState<StructuredEntry[]>([])
  const [query, setQuery] = useState('')
  const [loading, setLoading] = useState(false)

  const search = async (q: string) => {
    setLoading(true)
    try {
      if (window.go?.main.App) {
        const results = await window.go.main.App.SearchKnowledge(q, 10)
        setEntries(results.map(r => r.entry))
      } else {
        // Mock data for development
        setEntries([
          {
            id: '1',
            session_id: 'sess-001',
            timestamp: new Date().toISOString(),
            summary: '重构认证模块，引入 OAuth2 PKCE 流程',
            design_decision: '为提升 SPA 应用的安全性，选择 PKCE 替代传统 implicit 流程，防止授权码截获攻击。',
            impact_analysis: '影响前端登录页面、后端 /auth 路由及 token 刷新逻辑。',
            affected_files: [
              { path: 'src/auth/google-oauth.ts', change_type: 'modify' },
              { path: 'src/auth/pkce-helper.ts', change_type: 'add' },
            ],
            tags: ['auth', 'security', 'oauth2'],
          },
        ])
      }
    } catch (e) {
      console.error(e)
    } finally {
      setLoading(false)
    }
  }

  const capture = async () => {
    setLoading(true)
    try {
      if (window.go?.main.App) {
        const entry = await window.go.main.App.CaptureChanges('manual-' + Date.now())
        setEntries(prev => [entry, ...prev])
      }
    } catch (e) {
      console.error(e)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    search('')
  }, [])

  return (
    <div>
      <div className="page-header">
        <h2>📊 仪表盘</h2>
        <p>查看项目知识库的近期变更与全局概览</p>
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
