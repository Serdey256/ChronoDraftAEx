import { useEffect, useRef, useState, useCallback } from 'react'
import * as d3 from 'd3'
import { Call } from '@wailsio/runtime'
import { useToast } from '../contexts/ToastContext'

interface SimNode extends d3.SimulationNodeDatum {
  id: string
  label: string
  type: string
  depth: number
}

interface SimLink extends d3.SimulationLinkDatum<SimNode> {
  relation: string
}

const colorMap: Record<string, string> = {
  module: '#58a6ff',
  directory: '#3fb950',
  decision: '#d29922',
  concept: '#238636',
  file: '#8b949e',
  entry: '#bc8cff',
  tag: '#f778ba',
}

// Compute depth of each node based on CONTAINS edges (directory nesting level)
function computeNodeDepths(nodes: SimNode[], links: SimLink[]): Map<string, number> {
  const depthMap = new Map<string, number>()
  const childrenOf = new Map<string, string[]>()
  const parentOf = new Map<string, string>()

  for (const l of links) {
    if (l.relation === 'CONTAINS') {
      const s = typeof l.source === 'object' ? (l.source as SimNode).id : String(l.source)
      const t = typeof l.target === 'object' ? (l.target as SimNode).id : String(l.target)
      if (!childrenOf.has(s)) childrenOf.set(s, [])
      childrenOf.get(s)!.push(t)
      parentOf.set(t, s)
    }
  }

  function calcDepth(id: string, visited: Set<string>): number {
    if (visited.has(id)) return 0
    visited.add(id)
    const parent = parentOf.get(id)
    if (parent) return calcDepth(parent, visited) + 1
    return 0
  }

  for (const n of nodes) {
    depthMap.set(n.id, calcDepth(n.id, new Set()))
  }
  return depthMap
}

// Get all descendant node IDs via CONTAINS edges
function getDescendants(nodeId: string, links: SimLink[]): Set<string> {
  const desc = new Set<string>()
  const stack = [nodeId]
  while (stack.length > 0) {
    const current = stack.pop()!
    for (const l of links) {
      if (l.relation !== 'CONTAINS') continue
      const s = typeof l.source === 'object' ? (l.source as SimNode).id : String(l.source)
      const t = typeof l.target === 'object' ? (l.target as SimNode).id : String(l.target)
      if (s === current && !desc.has(t)) {
        desc.add(t)
        stack.push(t)
      }
    }
  }
  return desc
}

function KnowledgeGraph() {
  const svgRef = useRef<SVGSVGElement>(null)
  const toast = useToast()
  const [graphQuery, setGraphQuery] = useState('')
  const [graphTopK, setGraphTopK] = useState(5)
  const [graphLoading, setGraphLoading] = useState(false)

  // Track collapsed directory nodes
  const [, setCollapsedDirs] = useState<Set<string>>(new Set())
  const collapsedRef = useRef<Set<string>>(new Set())

  const searchGraph = async () => {
    if (!graphQuery.trim()) return
    setGraphLoading(true)
    try {
      const result = await (Call as any).ByName('main.ChronoService.SearchGraph', graphQuery, graphTopK)
      const data = result && typeof result === 'object' ? result : (typeof result === 'string' ? JSON.parse(result) : null)
      if (svgRef.current) d3.select(svgRef.current).selectAll('*').remove()
      if (data?.nodes?.length) {
        const rawNodes: SimNode[] = data.nodes.map((n: any) => ({
          id: n.id,
          label: n.label,
          type: n.type,
          depth: 0,
        }))
        const links: SimLink[] = data.edges.map((e: any) => ({
          source: e.source_id,
          target: e.target_id,
          relation: e.relation,
        }))

        // Compute depths
        const depthMap = computeNodeDepths(rawNodes, links)
        for (const n of rawNodes) {
          n.depth = depthMap.get(n.id) ?? 0
        }

        // Auto-collapse directories at depth >= 3
        const initCollapsed = new Set<string>()
        for (const n of rawNodes) {
          if ((n.type === 'directory' || n.type === 'module') && n.depth >= 2) {
            initCollapsed.add(n.id)
          }
        }
        collapsedRef.current = initCollapsed
        setCollapsedDirs(initCollapsed)

        renderGraph(rawNodes, links, initCollapsed)
      } else {
        toast.info('未找到相关图谱数据')
      }
    } catch (e: any) {
      toast.error('图谱搜索失败: ' + (e.message || String(e)))
    } finally {
      setGraphLoading(false)
    }
  }

  // Toggle collapse state via double-click
  const toggleCollapse = useCallback((nodeId: string, nodes: SimNode[], links: SimLink[]) => {
    const newCollapsed = new Set(collapsedRef.current)
    if (newCollapsed.has(nodeId)) {
      newCollapsed.delete(nodeId)
    } else {
      newCollapsed.add(nodeId)
    }
    collapsedRef.current = newCollapsed
    setCollapsedDirs(newCollapsed)
    renderGraph(nodes, links, newCollapsed)
  }, [])

  useEffect(() => {
    return () => {
      if (svgRef.current) d3.select(svgRef.current).selectAll('*').remove()
    }
  }, [])

  const renderGraph = (allNodes: SimNode[], allLinks: SimLink[], collapsed: Set<string>) => {
    if (!svgRef.current) return
    const svg = d3.select(svgRef.current)
    svg.selectAll('*').remove()
    svg.style('cursor', 'grab')

    const width = svgRef.current.clientWidth
    const height = svgRef.current.clientHeight

    // Filter out nodes that are descendants of collapsed directories
    const hiddenNodes = new Set<string>()
    for (const dirId of collapsed) {
      const desc = getDescendants(dirId, allLinks)
      for (const d of desc) hiddenNodes.add(d)
    }

    const nodes = allNodes.filter(n => !hiddenNodes.has(n.id))
    const nodeIdSet = new Set(nodes.map(n => n.id))
    const links = allLinks.filter(l => {
      const s = typeof l.source === 'object' ? (l.source as SimNode).id : String(l.source)
      const t = typeof l.target === 'object' ? (l.target as SimNode).id : String(l.target)
      return nodeIdSet.has(s) && nodeIdSet.has(t)
    })

    // Build child count metadata for collapsed nodes
    const childCounts = new Map<string, number>()
    for (const dirId of collapsed) {
      const desc = getDescendants(dirId, allLinks)
      childCounts.set(dirId, desc.size)
    }

    // Container group for zoom/pan transforms
    const g = svg.append('g')

    const simulation = d3.forceSimulation(nodes)
      .force('link', d3.forceLink<SimNode, SimLink>(links).id(d => d.id).distance(120))
      .force('charge', d3.forceManyBody().strength(-400))
      .force('center', d3.forceCenter(0, 0))
      .force('collide', d3.forceCollide().radius(40))

    const link = g.append('g')
      .attr('stroke', '#30363d')
      .attr('stroke-opacity', 0.8)
      .selectAll('line')
      .data(links)
      .join('line')
      .attr('stroke-width', 2)

    const linkLabel = g.append('g')
      .selectAll('text')
      .data(links)
      .join('text')
      .text(d => d.relation)
      .attr('font-size', 10)
      .attr('fill', '#8b949e')
      .attr('text-anchor', 'middle')

    const node = g.append('g')
      .selectAll('g')
      .data(nodes)
      .join('g')
      // @ts-expect-error D3 drag type incompatibility
      .call(d3.drag<SVGGElement, SimNode>().on('start', (event: any, d: any) => {
        if (!event.active) simulation.alphaTarget(0.3).restart()
        d.fx = d.x
        d.fy = d.y
      }).on('drag', (event: any, d: any) => {
        d.fx = event.x
        d.fy = event.y
      }).on('end', (event: any, d: any) => {
        if (!event.active) simulation.alphaTarget(0)
        d.fx = null
        d.fy = null
      }) as unknown as (selection: d3.Selection<SVGGElement, SimNode, SVGGElement, unknown>) => void)

    // Double-click handler for collapse/expand on directory nodes
    node.on('dblclick', (event: MouseEvent, d: SimNode) => {
      if (d.type === 'directory' || d.type === 'module') {
        event.preventDefault()
        event.stopPropagation()
        toggleCollapse(d.id, allNodes, allLinks)
      }
    })

    // Node circles
    node.append('circle')
      .attr('r', d => {
        if (childCounts.has(d.id)) return 30
        return 24
      })
      .attr('fill', d => colorMap[d.type] || '#8b949e')
      .attr('stroke', d => childCounts.has(d.id) ? '#d29922' : '#0f1117')
      .attr('stroke-width', d => childCounts.has(d.id) ? 3 : 2)
      .style('cursor', d => (d.type === 'directory' || d.type === 'module') ? 'pointer' : 'default')

    // Collapsed count badge
    node.filter(d => childCounts.has(d.id))
      .append('text')
      .attr('dy', 3)
      .attr('text-anchor', 'middle')
      .attr('fill', '#fff')
      .attr('font-size', 10)
      .attr('font-weight', 700)
      .text(d => childCounts.get(d.id)!.toString())

    // Label
    node.append('text')
      .text(d => {
        let label = d.label
        if (label.length > 25) label = label.substring(0, 22) + '...'
        return label
      })
      .attr('dy', 40)
      .attr('text-anchor', 'middle')
      .attr('fill', '#e6edf3')
      .attr('font-size', 12)
      .attr('font-weight', 500)

    simulation.on('tick', () => {
      link
        .attr('x1', d => (d.source as SimNode).x ?? 0)
        .attr('y1', d => (d.source as SimNode).y ?? 0)
        .attr('x2', d => (d.target as SimNode).x ?? 0)
        .attr('y2', d => (d.target as SimNode).y ?? 0)

      linkLabel
        .attr('x', d => {
          const sx = (d.source as SimNode).x ?? 0
          const tx = (d.target as SimNode).x ?? 0
          return (sx + tx) / 2
        })
        .attr('y', d => {
          const sy = (d.source as SimNode).y ?? 0
          const ty = (d.target as SimNode).y ?? 0
          return (sy + ty) / 2
        })

      node.attr('transform', d => `translate(${d.x ?? 0},${d.y ?? 0})`)
    })

    // Zoom behavior
    const zoom = d3.zoom<SVGSVGElement, unknown>()
      .scaleExtent([0.1, 4])
      .on('zoom', (event) => {
        g.attr('transform', event.transform)
      })
      .on('start', () => svg.style('cursor', 'grabbing'))
      .on('end', () => svg.style('cursor', 'grab'))

    svg.call(zoom)
    svg.on('dblclick.zoom', null)
    svg.call(zoom.transform, d3.zoomIdentity.translate(width / 2, height / 2))
  }

  // Legend items
  const legendItems = [
    { type: 'directory', label: '目录/模块', color: colorMap.directory },
    { type: 'file', label: '文件', color: colorMap.file },
    { type: 'entry', label: '知识条目', color: colorMap.entry },
    { type: 'tag', label: '标签', color: colorMap.tag },
    { type: 'decision', label: '设计决策', color: colorMap.decision },
  ]

  return (
    <div>
      <div className="page-header">
        <h2>🕸️ 知识图谱</h2>
        <p>可视化浏览项目知识实体间的关联关系 · 双击目录节点折叠/展开子级</p>
      </div>
      <div className="card" style={{ marginBottom: 16 }}>
        <div style={{ display: 'flex', gap: 8, alignItems: 'center' }}>
          <input
            className="input"
            placeholder="输入关键词搜索关联图谱..."
            value={graphQuery}
            onChange={e => setGraphQuery(e.target.value)}
            onKeyDown={e => e.key === 'Enter' && searchGraph()}
            style={{ flex: 1 }}
          />
          <select
            value={graphTopK}
            onChange={e => setGraphTopK(Number(e.target.value))}
            style={{ width: 80, padding: '6px 8px', background: 'var(--bg-secondary)', border: '1px solid var(--border)', borderRadius: 6, color: 'var(--text-primary)' }}
          >
            {[3,5,10,15,20].map(n => <option key={n} value={n}>{n}</option>)}
          </select>
          <button className="btn btn-primary" onClick={searchGraph} disabled={graphLoading}>
            {graphLoading ? '搜索中...' : '搜索'}
          </button>
        </div>

        {/* Legend */}
        <div style={{ display: 'flex', gap: 16, marginTop: 10, flexWrap: 'wrap' }}>
          {legendItems.map(item => (
            <div key={item.type} style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
              <div style={{
                width: 12, height: 12, borderRadius: '50%',
                backgroundColor: item.color, border: '1px solid #0f1117'
              }} />
              <span style={{ fontSize: 11, color: 'var(--text-secondary)' }}>{item.label}</span>
            </div>
          ))}
          <div style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
            <div style={{
              width: 14, height: 14, borderRadius: '50%',
              border: '2px solid #d29922', backgroundColor: 'transparent'
            }} />
            <span style={{ fontSize: 11, color: 'var(--text-secondary)' }}>已折叠（显示子节点数）</span>
          </div>
        </div>
      </div>
      <div className="graph-container">
        <svg ref={svgRef} width="100%" height="100%" />
      </div>
    </div>
  )
}

export default KnowledgeGraph
