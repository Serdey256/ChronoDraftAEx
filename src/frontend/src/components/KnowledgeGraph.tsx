import { useEffect, useRef, useState } from 'react'
import * as d3 from 'd3'
import { Call } from '@wailsio/runtime'
import { useToast } from '../contexts/ToastContext'

interface SimNode extends d3.SimulationNodeDatum {
  id: string
  label: string
  type: string
}

interface SimLink extends d3.SimulationLinkDatum<SimNode> {
  relation: string
}

const colorMap: Record<string, string> = {
  module: '#58a6ff',
  decision: '#d29922',
  concept: '#238636',
  file: '#8b949e',
  entry: '#bc8cff',
  tag: '#f778ba',
}

function KnowledgeGraph() {
  const svgRef = useRef<SVGSVGElement>(null)
  const toast = useToast()
  const [graphQuery, setGraphQuery] = useState('')
  const [graphTopK, setGraphTopK] = useState(5)
  const [graphLoading, setGraphLoading] = useState(false)

  const searchGraph = async () => {
    if (!graphQuery.trim()) return
    setGraphLoading(true)
    try {
      const result = await (Call as any).ByName('main.ChronoService.GetGraphData', graphQuery, graphTopK, 'true')
      const data = typeof result === 'string' ? JSON.parse(result) : result
      if (svgRef.current) d3.select(svgRef.current).selectAll('*').remove()
      if (data?.nodes?.length) {
        const nodes: SimNode[] = data.nodes.map((n: any) => ({ id: n.id, label: n.label, type: n.type }))
        const links: SimLink[] = data.edges.map((e: any) => ({ source: e.source_id, target: e.target_id, relation: e.relation }))
        renderGraph(nodes, links)
      } else {
        toast.info('未找到相关图谱数据')
      }
    } catch (e: any) {
      toast.error('图谱搜索失败: ' + (e.message || String(e)))
    } finally {
      setGraphLoading(false)
    }
  }

  useEffect(() => {
    return () => {
      if (svgRef.current) d3.select(svgRef.current).selectAll('*').remove()
    }
  }, [])

  const renderGraph = (nodes: SimNode[], links: SimLink[]) => {
    if (!svgRef.current) return
    const svg = d3.select(svgRef.current)
    svg.selectAll('*').remove()
    svg.style('cursor', 'grab')

    const width = svgRef.current.clientWidth
    const height = svgRef.current.clientHeight

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
      // @ts-expect-error D3 drag type incompatibility with generic Selection
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

    node.append('circle')
      .attr('r', 24)
      .attr('fill', d => colorMap[d.type] || '#8b949e')
      .attr('stroke', '#0f1117')
      .attr('stroke-width', 2)

    node.append('text')
      .text(d => d.label)
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

    // Zoom behavior — panning and mouse wheel zoom
    const zoom = d3.zoom<SVGSVGElement, unknown>()
      .scaleExtent([0.1, 4])
      .on('zoom', (event) => {
        g.attr('transform', event.transform)
      })
      .on('start', () => svg.style('cursor', 'grabbing'))
      .on('end', () => svg.style('cursor', 'grab'))

    svg.call(zoom)
    // Center the graph initially
    svg.call(zoom.transform, d3.zoomIdentity.translate(width / 2, height / 2))
  }

  return (
    <div>
      <div className="page-header">
        <h2>🕸️ 知识图谱</h2>
        <p>可视化浏览项目知识实体间的关联关系</p>
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
      </div>
      <div className="graph-container">
        <svg ref={svgRef} width="100%" height="100%" />
      </div>
    </div>
  )
}

export default KnowledgeGraph
