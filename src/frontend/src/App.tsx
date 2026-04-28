import { Routes, Route, NavLink } from 'react-router-dom'
import Dashboard from './components/Dashboard'
import KnowledgeGraph from './components/KnowledgeGraph'
import SnapshotView from './components/SnapshotView'

function App() {
  return (
    <div className="app-container">
      <aside className="sidebar">
        <div className="brand">
          <h1>ChronoDraftAEx</h1>
          <p>AI 编码的第二个大脑</p>
        </div>
        <nav className="nav-menu">
          <NavLink to="/" className={({ isActive }) => isActive ? 'nav-item active' : 'nav-item'} end>
            📊 仪表盘
          </NavLink>
          <NavLink to="/graph" className={({ isActive }) => isActive ? 'nav-item active' : 'nav-item'}>
            🕸️ 知识图谱
          </NavLink>
          <NavLink to="/snapshot" className={({ isActive }) => isActive ? 'nav-item active' : 'nav-item'}>
            📸 项目快照
          </NavLink>
        </nav>
        <div className="sidebar-footer">
          <div className="status-indicator">
            <span className="dot online" />
            MCP 服务运行中
          </div>
        </div>
      </aside>
      <main className="main-content">
        <Routes>
          <Route path="/" element={<Dashboard />} />
          <Route path="/graph" element={<KnowledgeGraph />} />
          <Route path="/snapshot" element={<SnapshotView />} />
        </Routes>
      </main>
    </div>
  )
}

export default App
