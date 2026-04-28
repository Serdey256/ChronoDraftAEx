import { useEffect, useRef } from 'react'
import * as d3 from 'd3'

interface GraphNode {
  id: string
  label: string
  type: string
  x?: number
  y?: number
}

interface GraphLink {
  source: string | GraphNode
  target: string | GraphNode
  relation: string
}

function KnowledgeGraph() {
  const svgRef = useRef<SVGSVGElement>(null)

  useEffect(() => {
    if (!svgRef.current) return

    // Mock data for demonstration
    const nodes: GraphNode[] = [
      { id: 'auth', label: 'auth 模块', type: 'module' },
      { id: 'security', label: 'security 模块', type: 'module' },
      { id: 'database', label: 'database 模块', type: 'module' },
      { id: 'oauth', label: 'OAuth2 决策', type: 'decision' },
      { id: 'pkce', label: 'PKCE 流程', type: 'concept' },
      { id: 'token', label: 'Token 管理', type: 'concept' },
    ]

    const links: GraphLink[] = [
      { source: 'oauth', target: 'auth', relation: 'affects' },
      { source: 'oauth', target: 'security', relation: 'relates_to' },
      { source: 'pkce', target: 'oauth', relation: 'implements' },
      { source: 'token', target: 'auth', relation: 'belongs_to' },
      { source: 'database', target: 'auth', relation: 'depends_on' },
    ]

    const svg = d3.select(svgRef.current)
    svg.selectAll('*').remove()

    const width = svgRef.current.clientWidth
    const height = svgRef.current.clientHeight

    // Color mapping
    const colorMap: Record<string, string> = {
      module: '#58a6ff',
      decision: '#d29922',
      concept: '#238636',
      file: '#8b949e',
    }

    const simulation = d3.forceSimulation(nodes as any)
      .force('link', d3.forceLink(links).id((d: any) => d.id).distance(120))
      .force('charge', d3.forceManyBody().strength(-400))
      .force('center', d3.forceCenter(width / 2, height / 2))
      .force('collide', d3.forceCollide().radius(40))

    // Links
    const link = svg.append('g')
      .attr('stroke', '#30363d')
      .attr('stroke-opacity', 0.8)
      .selectAll('line')
      .data(links)
      .join('line')
      .attr('stroke-width', 2)

    // Link labels
    const linkLabel = svg.append('g')
      .selectAll('text')
      .data(links)
      .join('text')
      .text((d: any) => d.relation)
      .attr('font-size', 10)
      .attr('fill', '#8b949e')
      .attr('text-anchor', 'middle')

    // Nodes
    const node = svg.append('g')
      .selectAll('g')
      .data(nodes)
      .join('g')
      .call(d3.drag<SVGGElement, GraphNode>()
        .on('start', (event, d) => {
          if (!event.active) simulation.alphaTarget(0.3).restart()
          d.x = event.x
          d.y = event.y
        })
        .on('drag', (event, d) => {
          d.x = event.x
          d.y = event.y
        })
        .on('end', (event, d) => {
          if (!event.active) simulation.alphaTarget(0)
          d.x = event.x
          d.y = event.y
        })
      )

    node.append('circle')
      .attr('r', 24)
      .attr('fill', (d: any) => colorMap[d.type] || '#8b949e')
      .attr('stroke', '#0f1117')
      .attr('stroke-width', 2)

    node.append('text')
      .text((d: any) => d.label)
      .attr('dy', 40)
      .attr('text-anchor', 'middle')
      .attr('fill', '#e6edf3')
      .attr('font-size', 12)
      .attr('font-weight', 500)

    simulation.on('tick', () => {
      link
        .attr('x1', (d: any) => (d.source as any).x)
        .attr('y1', (d: any) => (d.source as any).y)
        .attr('x2', (d: any) => (d.target as any).x)
        .attr('y2', (d: any) => (d.target as any).y)

      linkLabel
        .attr('x', (d: any) => ((d.source as any).x + (d.target as any).x) / 2)
        .attr('y', (d: any) => ((d.source as any).y + (d.target as any).y) / 2)

      node.attr('transform', (d: any) => `translate(${d.x},${d.y})`)
    })

    return () => {
      simulation.stop()
    }
  }, [])

  return (
    <div>
      <div className="page-header">
        <h2>🕸️ 知识图谱</h2>
        <p>可视化浏览项目知识实体间的关联关系</p>
      </div>
      <div className="graph-container">
        <svg ref={svgRef} width="100%" height="100%" />
      </div>
    </div>
  )
}

export default KnowledgeGraph
