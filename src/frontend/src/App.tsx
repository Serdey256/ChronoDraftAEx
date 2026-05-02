import { useState, useEffect } from 'react'
import { Routes, Route, NavLink } from 'react-router-dom'
import { ToastProvider } from './contexts/ToastContext'
import Toast from './components/Toast'
import Dashboard from './components/Dashboard'
import KnowledgeGraph from './components/KnowledgeGraph'
import SnapshotView from './components/SnapshotView'
import ProjectManager from './components/ProjectManager'
import { StartWatcher, StopWatcher, IsWatcherRunning } from '../bindings/ChronoDraftAEx/chronoservice.js'

function App() {
  const [watcherRunning, setWatcherRunning] = useState(false)

  useEffect(() => {
    const poll = () => {
      IsWatcherRunning().then(setWatcherRunning).catch(() => setWatcherRunning(false))
    }
    poll()
    const interval = setInterval(poll, 5000)
    return () => clearInterval(interval)
  }, [])

  const toggleWatcher = async () => {
    try {
      if (watcherRunning) {
        await StopWatcher()
      } else {
        await StartWatcher()
      }
      const running = await IsWatcherRunning()
      setWatcherRunning(running)
    } catch (e) {
      console.error('切换监控状态失败:', e)
    }
  }

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
              <span className={`dot ${watcherRunning ? 'watcher-active' : 'watcher-stopped'}`} />
              {watcherRunning ? '文件监控中' : '监控已停止'}
            </div>
            <button
              className="btn btn-secondary watcher-toggle"
              onClick={toggleWatcher}
            >
              {watcherRunning ? '停止监控' : '启动监控'}
            </button>
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
