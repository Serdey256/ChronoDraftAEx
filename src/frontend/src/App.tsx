import { Routes, Route, NavLink } from 'react-router-dom'
import { ToastProvider } from './contexts/ToastContext'
import Toast from './components/Toast'
import Dashboard from './components/Dashboard'
import KnowledgeGraph from './components/KnowledgeGraph'
import SnapshotView from './components/SnapshotView'
import ProjectManager from './components/ProjectManager'

function App() {
  return (
    <ToastProvider>
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
            <NavLink to="/projects" className={({ isActive }) => isActive ? 'nav-item active' : 'nav-item'}>
              📁 项目管理
            </NavLink>
          </nav>
          <div className="sidebar-footer">
            <div className="status-indicator">
              <span className="dot online" />
              Agent 驱动模式
            </div>
          </div>
        </aside>
        <main className="main-content">
          <Routes>
            <Route path="/" element={<Dashboard />} />
            <Route path="/graph" element={<KnowledgeGraph />} />
            <Route path="/snapshot" element={<SnapshotView />} />
            <Route path="/projects" element={<ProjectManager />} />
          </Routes>
        </main>
      </div>
      <Toast />
    </ToastProvider>
  )
}

export default App
