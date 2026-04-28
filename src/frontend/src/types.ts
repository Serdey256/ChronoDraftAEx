// ChronoDraftAEx 前端与后端共享的核心类型定义

export interface FileChange {
  path: string
  change_type: string // add, modify, delete
  diff?: string
}

export interface StructuredEntry {
  id: string
  timestamp: string
  session_id: string
  summary: string
  design_decision: string
  impact_analysis: string
  affected_files: FileChange[]
  tags: string[]
}

export interface ProjectSnapshot {
  id: string
  timestamp: string
  version: string
  dependencies: string[]
  metadata: Record<string, string>
}

export interface KnowledgeNode {
  id: string
  label: string
  type: string
  metadata: Record<string, string>
}

export interface KnowledgeEdge {
  source_id: string
  target_id: string
  relation: string
}

export interface SearchResult {
  entry: StructuredEntry
  score: number
  node_path?: KnowledgeNode[]
}
