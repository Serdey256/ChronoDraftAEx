import { useState, useEffect } from 'react'
import { ListSnapshots, CreateSnapshot } from '../../bindings/ChronoDraftAEx/chronoservice.js'
import { useToast } from '../contexts/ToastContext'
import type { ProjectSnapshot } from '../types'

function SnapshotView() {
  const toast = useToast()
  const [snapshots, setSnapshots] = useState<ProjectSnapshot[]>([])
  const [selected, setSelected] = useState<ProjectSnapshot | null>(null)
  const [compareTarget, setCompareTarget] = useState<ProjectSnapshot | null>(null)
  const [loading, setLoading] = useState(false)

  useEffect(() => {
    loadSnapshots()
  }, [])

  const loadSnapshots = async () => {
    setLoading(true)
    try {
      const snaps = await ListSnapshots()
      if (snaps && snaps.length > 0) {
        setSnapshots(snaps.map((s: any) => ({
          id: s.id,
          timestamp: s.timestamp ? s.timestamp.toISOString() : new Date().toISOString(),
          version: s.version,
          dependencies: s.dependencies || [],
          metadata: s.metadata || {},
        })))
        setLoading(false)
        return
      }
    } catch (e) {
      toast.warning('获取快照失败')
    }
    setSnapshots([])
    setLoading(false)
  }

  const createSnapshot = async () => {
    setLoading(true)
    try {
      const newSnap = await CreateSnapshot(
        'v' + new Date().toISOString().slice(0, 10),
        ['react@18.3.1', 'wails@v3.0.0-alpha']
      )
      if (newSnap) {
        setSnapshots(prev => [{
          id: newSnap.id,
          timestamp: newSnap.timestamp ? newSnap.timestamp.toISOString() : new Date().toISOString(),
          version: newSnap.version,
          dependencies: newSnap.dependencies || [],
          metadata: newSnap.metadata as Record<string, string> || {},
        }, ...prev])
      }
    } catch (e) {
      toast.error('创建快照失败')
    } finally {
      setLoading(false)
    }
  }

  const compareSnapshots = (a: ProjectSnapshot, b: ProjectSnapshot) => {
    const added = b.dependencies.filter(dep => !a.dependencies.includes(dep))
    const removed = a.dependencies.filter(dep => !b.dependencies.includes(dep))
    return { added, removed }
  }

  const diff = selected && compareTarget
    ? compareSnapshots(selected, compareTarget)
    : null

  return (
    <div>
      <div className="page-header">
        <h2>📸 项目快照</h2>
        <p>对比不同版本间的依赖、接口与宏观变化</p>
      </div>

      <div style={{ marginBottom: 24 }}>
        <button className="btn btn-primary" onClick={createSnapshot} disabled={loading}>
          {loading ? '创建中...' : '➕ 创建当前快照'}
        </button>
      </div>

      <div className="grid">
        <div className="card" style={{ gridColumn: 'span 1' }}>
          <div className="card-title">快照列表</div>
          <div className="entry-list">
            {snapshots.map(snap => (
              <div
                className="entry-item"
                key={snap.id}
                style={{
                  cursor: 'pointer',
                  borderColor: selected?.id === snap.id ? 'var(--accent)' : undefined,
                }}
                onClick={() => {
                  setSelected(snap)
                  setCompareTarget(null)
                }}
              >
                <div className="entry-meta">
                  <span style={{ fontWeight: 600, color: 'var(--text-primary)' }}>{snap.version}</span>
                  <span>·</span>
                  <span>{new Date(snap.timestamp).toLocaleDateString()}</span>
                </div>
                <div style={{ fontSize: 12, color: 'var(--text-secondary)' }}>
                  依赖数: {snap.dependencies.length} · {snap.metadata?.branch || ''}
                </div>
              </div>
            ))}
            {snapshots.length === 0 && !loading && (
              <div style={{ textAlign: 'center', color: 'var(--text-secondary)', padding: 40 }}>
                暂无快照，点击「创建当前快照」开始
              </div>
            )}
          </div>
        </div>

        <div className="card" style={{ gridColumn: 'span 2' }}>
          <div className="card-title">快照详情</div>
          {selected ? (
            <div>
              <div style={{ marginBottom: 16 }}>
                <h4 style={{ fontSize: 16, marginBottom: 8 }}>{selected.version}</h4>
                <div style={{ color: 'var(--text-secondary)', fontSize: 13 }}>
                  <p>时间: {new Date(selected.timestamp).toLocaleString()}</p>
                  <p>分支: {selected.metadata?.branch || 'N/A'}</p>
                  <p>提交: {selected.metadata?.commit || 'N/A'}</p>
                </div>
              </div>

              <div style={{ marginBottom: 16 }}>
                <div style={{ fontWeight: 600, marginBottom: 8 }}>依赖列表</div>
                <div className="entry-tags">
                  {selected.dependencies.map(dep => (
                    <span className="tag" key={dep}>{dep}</span>
                  ))}
                </div>
              </div>

              <div style={{ marginBottom: 16 }}>
                <div style={{ fontWeight: 600, marginBottom: 8 }}>对比版本</div>
                <select
                  className="input"
                  style={{ width: 'auto', minWidth: 200 }}
                  value={compareTarget?.id || ''}
                  onChange={e => {
                    const target = snapshots.find(s => s.id === e.target.value)
                    setCompareTarget(target || null)
                  }}
                >
                  <option value="">选择要对比的快照...</option>
                  {snapshots.filter(s => s.id !== selected.id).map(s => (
                    <option value={s.id} key={s.id}>{s.version}</option>
                  ))}
                </select>
              </div>

              {diff && (
                <div style={{ background: 'var(--bg-tertiary)', borderRadius: 8, padding: 16 }}>
                  <div style={{ fontWeight: 600, marginBottom: 12 }}>变更对比</div>
                  {diff.added.length > 0 && (
                    <div style={{ marginBottom: 12 }}>
                      <div style={{ color: 'var(--success)', fontSize: 12, marginBottom: 4 }}>➕ 新增依赖</div>
                      <div className="entry-tags">
                        {diff.added.map(dep => (
                          <span className="tag" key={dep} style={{ background: 'rgba(35,134,54,0.2)', color: '#3fb950' }}>
                            {dep}
                          </span>
                        ))}
                      </div>
                    </div>
                  )}
                  {diff.removed.length > 0 && (
                    <div>
                      <div style={{ color: 'var(--danger)', fontSize: 12, marginBottom: 4 }}>➖ 移除依赖</div>
                      <div className="entry-tags">
                        {diff.removed.map(dep => (
                          <span className="tag" key={dep} style={{ background: 'rgba(218,54,51,0.2)', color: '#f85149' }}>
                            {dep}
                          </span>
                        ))}
                      </div>
                    </div>
                  )}
                  {diff.added.length === 0 && diff.removed.length === 0 && (
                    <div style={{ color: 'var(--text-secondary)', fontSize: 13 }}>两个快照的依赖完全一致</div>
                  )}
                </div>
              )}
            </div>
          ) : (
            <div style={{ textAlign: 'center', color: 'var(--text-secondary)', padding: 40 }}>
              从左侧选择一个快照查看详情
            </div>
          )}
        </div>
      </div>
    </div>
  )
}

export default SnapshotView
