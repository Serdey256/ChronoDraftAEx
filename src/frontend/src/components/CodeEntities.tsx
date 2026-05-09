import { useState, useEffect } from 'react'
import { Call } from '@wailsio/runtime'
import type { CodeEntity } from '../types'

const c = Call as any

// diagnostic log — visible on page, no browser console needed
let _diag: string[] = []
function diag(msg: string) { _diag.push(`[${new Date().toLocaleTimeString()}] ${msg}`) }

async function callGo<T>(method: string, ...args: any[]): Promise<{ ok: true; data: T } | { ok: false; err: string }> {
  try {
    if (typeof c.ByName === 'function') {
      const r = await c.ByName(method, ...args)
      diag(`${method}(${JSON.stringify(args).slice(0,80)}) OK — ${Array.isArray(r) ? `array[${r.length}]` : typeof r}`)
      return { ok: true, data: r as T }
    }
    diag(`${method}: ByName not available. Call keys: ${Object.keys(c).slice(0,6).join(',')}`)
    return { ok: false, err: `No ByName method. Keys: ${Object.keys(c).slice(0,6)}` }
  } catch (e: any) {
    diag(`${method}: FAILED — ${e?.message || e}`)
    return { ok: false, err: e?.message || String(e) }
  }
}

const typeBadge: Record<string, string> = {
  'function': '🔧', 'struct': '📦', 'class': '📦',
  'interface': '🔌', 'import': '📥', 'const': '📌',
}

function CodeEntities() {
  const [entities, setEntities] = useState<CodeEntity[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [aiEnabled, setAiEnabled] = useState(false)
  const [diagLog, setDiagLog] = useState('')
  const [showDebug, setShowDebug] = useState(false)
  const [annotationCurrent, setAnnotationCurrent] = useState(0)
  const [annotationTotal, setAnnotationTotal] = useState(0)
  const [annotating, setAnnotating] = useState(false)

  useEffect(() => {
    callGo<any>('main.ChronoService.IsAIAnnotationEnabled').then(
      r => { if (r.ok) setAiEnabled(!!r.data) }, () => {}
    )
  }, [])

  const loadEntities = async () => {
    _diag = []
    diag('=== loadEntities() ===')
    setLoading(true); setError(null); setDiagLog('')

    try {
      // Primary: get all entities in one call (no path matching needed)
      diag('Calling GetAllCodeEntities')
      const r = await callGo<CodeEntity[]>('main.ChronoService.GetAllCodeEntities')
      let all: CodeEntity[] = []
      if (r.ok) {
        const d: any = r.data
        if (Array.isArray(d)) {
          all = d
          diag(`GetAllCodeEntities OK — array[${d.length}]`)
        } else if (typeof d === 'object' && d !== null) {
          // Wails object wrapper — try unwrap
          const keys = Object.keys(d)
          diag(`GetAllCodeEntities returned object with keys: ${keys.slice(0,10).join(', ')}`)
          const unwrapped = d.list || d.data || d.result || d.value || d[keys[0]]
          if (Array.isArray(unwrapped)) { all = unwrapped; diag(`Unwrapped via property, got ${unwrapped.length} entities`) }
          else if (typeof unwrapped === 'object' && unwrapped !== null) {
            // Entities might be keyed by something
            const vals = Object.values(unwrapped)
            if (vals.length > 0 && Array.isArray(vals[0])) { all = vals[0]; diag(`Double-unwrapped, got ${vals[0].length} entities`) }
            else diag(`Could not unwrap: values are ${typeof vals[0]}`)
          }
        }
      } else {
        diag(`GetAllCodeEntities FAILED: ${r.err}`)
      }

      setEntities(all)
      diag(`Total entities: ${all.length}`)
      if (all.length === 0) {
        setError('未找到代码实体。请确认:\n1. 已点击仪表盘"扫描项目"按钮\n2. 项目路径下存在 .go 或 .ts 源文件')
      }
    } catch (e: any) {
      diag(`FATAL: ${e?.message || e}`)
      setError('加载异常: ' + (e.message || String(e)))
    } finally {
      setLoading(false)
      setDiagLog(_diag.join('\n'))
    }
  }

  useEffect(() => { loadEntities() }, [])

  const grouped = entities.reduce<Record<string, CodeEntity[]>>((acc, e) => {
    const key = e.file_path || '(unknown)'
    if (!acc[key]) acc[key] = []
    acc[key].push(e)
    return acc
  }, {})

  function parseDescription(metadata?: string): string | null {
    if (!metadata) return null
    try { const m = JSON.parse(metadata); return m.description || null } catch { return null }
  }

  const annotateAll = async () => {
    setAnnotating(true)
    setAnnotationCurrent(0)
    setAnnotationTotal(0)
    diag('Starting batch AI annotation...')

    // Poll progress every 800ms
    let poll: ReturnType<typeof setInterval> | undefined
    try {
      poll = setInterval(async () => {
        try {
          const p = await callGo<any>('main.ChronoService.GetAnnotationProgress')
          if (p?.ok && p?.data) {
            const d = p.data
            setAnnotationCurrent(Number(d.current) || 0)
            setAnnotationTotal(Number(d.total) || 0)
          }
        } catch {}
      }, 800)

      await callGo('main.ChronoService.AnnotateAllCodeEntities')
      if (poll !== undefined) clearInterval(poll)
      setAnnotationCurrent(0)
      setAnnotationTotal(0)
      diag('Batch annotation complete — reloading entities')
      await loadEntities()
    } catch (e: any) {
      if (poll !== undefined) clearInterval(poll)
      diag(`Batch annotation FAILED: ${e?.message || e}`)
    } finally {
      setAnnotating(false)
      setShowDebug(true)
    }
  }

  const toggleAI = async () => {
    const next = !aiEnabled; setAiEnabled(next)
    try { await callGo('main.ChronoService.SetAIAnnotation', next) } catch { setAiEnabled(!next) }
  }

  return (
    <div>
      <div className="page-header">
        <h2>🧬 代码结构</h2>
        <p>AST 自动提取的函数、类型与导入关系</p>
        <div style={{ marginTop: 8, display: 'flex', alignItems: 'center', gap: 8, flexWrap: 'wrap' }}>
          <button
            className={aiEnabled ? 'btn btn-primary' : 'btn btn-secondary'}
            onClick={toggleAI}
            style={{ fontSize: 12, padding: '4px 12px' }}
          >
            {aiEnabled ? '🤖 AI 标注: 开' : '🧠 AI 标注: 关'}
          </button>
          {aiEnabled && (
            annotating ? (
              <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                <div style={{
                  height: 4, width: 120, background: 'var(--border)', borderRadius: 2,
                  overflow: 'hidden', flexShrink: 0
                }}>
                  <div style={{
                    height: '100%',
                    width: annotationTotal > 0 ? `${Math.round(annotationCurrent / annotationTotal * 100)}%` : '0%',
                    background: 'var(--accent)', borderRadius: 2,
                    transition: 'width 0.3s ease'
                  }} />
                </div>
                <span style={{ fontSize: 11, color: 'var(--text-secondary)', whiteSpace: 'nowrap' }}>
                  {annotationCurrent}/{annotationTotal}
                </span>
              </div>
            ) : (
              <button className="btn btn-primary" onClick={annotateAll} disabled={annotating}
                style={{ fontSize: 12, padding: '4px 12px' }}>
                ✏️ 批量AI标注
              </button>
            )
          )}
          <button
            className="btn btn-secondary"
            onClick={() => setShowDebug(!showDebug)}
            style={{ fontSize: 12, padding: '4px 12px' }}
          >
            {showDebug ? '🔍 诊断: 开' : '🔍 诊断'}
          </button>
        </div>
      </div>

      {/* DIAGNOSTIC PANEL */}
      {showDebug && diagLog && (
        <div style={{
          background: '#0d1117', border: '1px solid #30363d', borderRadius: 8,
          padding: 12, marginBottom: 16, fontFamily: 'monospace', fontSize: 11,
          color: '#8b949e', maxHeight: 400, overflow: 'auto', whiteSpace: 'pre-wrap'
        }}>
          <div style={{ color: '#58a6ff', marginBottom: 6, fontWeight: 600 }}>🔍 诊断日志</div>
          {diagLog}
        </div>
      )}

      {loading && <div style={{ textAlign: 'center', padding: 40, color: 'var(--text-secondary)' }}>正在加载代码结构...</div>}

      {error && !loading && (
        <div style={{ background: 'rgba(210,153,34,0.1)', border: '1px solid #d29922', borderRadius: 8, padding: 16, marginBottom: 16, textAlign: 'center' }}>
          <div style={{ fontSize: 14, color: '#d29922', marginBottom: 8, whiteSpace: 'pre-line' }}>{error}</div>
          <button className="btn btn-primary" onClick={loadEntities}>🔄 重试</button>
        </div>
      )}

      {!loading && !error && Object.keys(grouped).length === 0 && (
        <div style={{ textAlign: 'center', color: 'var(--text-secondary)', padding: 40 }}>
          暂无代码实体数据。请运行 <strong>index_project</strong> 触发 AST 分析。
        </div>
      )}

      {Object.entries(grouped).map(([file, fileEntities]) => (
        <div className="card" key={file} style={{ marginBottom: 16 }}>
          <div className="card-title" style={{ fontFamily: 'monospace', fontSize: 13 }}>📄 {file}</div>
          <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
            {fileEntities.map((e, i) => (
              <div key={i} style={{ display: 'flex', alignItems: 'flex-start', gap: 10, padding: '8px 10px', background: 'rgba(255,255,255,0.03)', borderRadius: 6, border: '1px solid rgba(255,255,255,0.06)' }}>
                <span style={{ fontSize: 16, flexShrink: 0, marginTop: 1 }}>{typeBadge[e.entity_type] || '📎'}</span>
                <div style={{ flex: 1, minWidth: 0 }}>
                  <div style={{ display: 'flex', alignItems: 'center', gap: 8, flexWrap: 'wrap' }}>
                    <span style={{ fontWeight: 600, color: 'var(--accent)', fontSize: 14 }}>{e.name}</span>
                    <span style={{ fontSize: 10, padding: '1px 6px', borderRadius: 4, background: 'rgba(88,166,255,0.15)', color: 'var(--accent)', textTransform: 'uppercase' }}>{e.entity_type}</span>
                  </div>
                  {e.signature && <div style={{ fontFamily: 'monospace', fontSize: 12, color: 'var(--text-secondary)', marginTop: 4, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{e.signature}</div>}
                  {parseDescription(e.metadata) && <div style={{ fontSize: 12, color: '#8b949e', marginTop: 3, fontStyle: 'italic' }}>💬 {parseDescription(e.metadata)}</div>}
                </div>
              </div>
            ))}
          </div>
        </div>
      ))}
    </div>
  )
}

export default CodeEntities
