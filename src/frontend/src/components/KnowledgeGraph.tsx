import { useEffect, useRef, useState } from 'react'
import * as d3 from 'd3'
import { GetGraphData } from '../../bindings/ChronoDraftAEx/chronoservice.js'

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

const mockNodes: SimNode[] = [
  { id: 'auth', label: 'auth 模块', type: 'module' },
  { id: 'security', label: 'security 模块', type: 'module' },
  { id: 'database', label: 'database 模块', type: 'module' },
  { id: 'oauth', label: 'OAuth2 决策', type: 'decision' },
  { id: 'pkce', label: 'PKCE 流程', type: 'concept' },
  { id: 'token', label: 'Token 管理', type: 'concept' },
]

const mockLinks: SimLink[] = [
  { source: 'oauth', target: 'auth', relation: 'affects' },
  { source: 'oauth', target: 'security', relation: 'relates_to' },
  { source: 'pkce', target: 'oauth', relation: 'implements' },
  { source: 'token', target: 'auth', relation: 'belongs_to' },
  { source: 'database', target: 'auth', relation: 'depends_on' },
]

function KnowledgeGraph() {
  const svgRef = useRef<SVGSVGElement>(null)
  const [loading, setLoading] = useState(false)

  useEffect(() => {
    if (!svgRef.current) return

    const loadAndRender = async () => {
      setLoading(true)
      let nodes: SimNode[] = []
      let links: SimLink[] = []

      try {
        const result = await GetGraphData(100)
        if (result?.nodes?.length) {
          nodes = result.nodes.map((n: any) => ({ id: n.id, label: n.label, type: n.type }))
          links = result.edges.map((e: any) => ({ source: e.source_id, target: e.target_id, relation: e.relation }))
        }
      } catch (e) {
        console.warn('获取图谱数据失败', e)
      } finally {
        setLoading(false)
      }

      if (nodes.length === 0) {
        nodes = mockNodes
        links = mockLinks
      }

      renderGraph(nodes, links)
    }

    loadAndRender()

    return () => {
      if (svgRef.current) d3.select(svgRef.current).selectAll('*').remove()
    }
  }, [])

  const renderGraph = (nodes: SimNode[], links: SimLink[]) => {
    if (!svgRef.current) return
    const svg = d3.select(svgRef.current)
    svg.selectAll('*').remove()

    const width = svgRef.current.clientWidth
    const height = svgRef.current.clientHeight

    const simulation = d3.forceSimulation(nodes)
      .force('link', d3.forceLink<SimNode, SimLink>(links).id(d => d.id).distance(120))
      .force('charge', d3.forceManyBody().strength(-400))
      .force('center', d3.forceCenter(width / 2, height / 2))
      .force('collide', d3.forceCollide().radius(40))

    const link = svg.append('g')
      .attr('stroke', '#30363d')
      .attr('stroke-opacity', 0.8)
      .selectAll('line')
      .data(links)
      .join('line')
      .attr('stroke-width', 2)

    const linkLabel = svg.append('g')
      .selectAll('text')
      .data(links)
      .join('text')
      .text(d => d.relation)
      .attr('font-size', 10)
      .attr('fill', '#8b949e')
      .attr('text-anchor', 'middle')

    const node = svg.append('g')
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
  }

  return (
    <div>
      <div className="page-header">
        <h2>🕸️ 知识图谱</h2>
        <p>{loading ? '加载中...' : '可视化浏览项目知识实体间的关联关系'}</p>
      </div>
      <div className="graph-container">
        <svg ref={svgRef} width="100%" height="100%" />
      </div>
    </div>
  )
}

export default KnowledgeGraph
