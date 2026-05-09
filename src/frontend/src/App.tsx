import { useRef } from 'react'
import { Routes, Route, NavLink } from 'react-router-dom'
import { ToastProvider, useToast } from './contexts/ToastContext'
import { BackgroundProvider, useBackground } from './contexts/BackgroundContext'
import Toast from './components/Toast'
import Dashboard from './components/Dashboard'
import KnowledgeGraph from './components/KnowledgeGraph'
import SnapshotView from './components/SnapshotView'
import ProjectManager from './components/ProjectManager'
import CodeEntities from './components/CodeEntities'

function App() {
  return (
    <BackgroundProvider>
      <ToastProvider>
        <AppContent />
        <Toast />
      </ToastProvider>
    </BackgroundProvider>
  )
}

function AppContent() {
  const toast = useToast()
  const { backgroundImage, bgOpacity, setBgOpacity, setBackground, clearBackground } = useBackground()
  const fileInputRef = useRef<HTMLInputElement>(null)

  const handleFileChange = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (!file) return
    const result = await setBackground(file)
    if (result) {
      toast.success('背景已更新')
    } else if (file.size > 3 * 1024 * 1024) {
      toast.error('图片过大，请选择小于 3MB 的图片')
    } else {
      toast.error('无法加载该图片')
    }
    // Reset input so same file can be re-selected
    if (fileInputRef.current) fileInputRef.current.value = ''
  }

  const handleReset = () => {
    clearBackground()
    toast.info('背景已重置')
  }

  return (
    <div
      className={backgroundImage ? 'has-background' : ''}
      style={backgroundImage ? { '--bg-card-opacity': bgOpacity } as React.CSSProperties : undefined}
    >
      {backgroundImage && (
        <div className="app-bg-layer" style={{ backgroundImage: `url(${backgroundImage})` }} />
      )}
      {backgroundImage && <div className="app-bg-overlay" />}
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
            <NavLink to="/code" className={({ isActive }) => isActive ? 'nav-item active' : 'nav-item'}>
              🧬 代码结构
            </NavLink>
          </nav>
          <div className="sidebar-footer">
            <div className="status-indicator">
              <span className="dot online" />
              Agent 驱动模式
            </div>
            <div className="bg-control">
              <input
                ref={fileInputRef}
                type="file"
                accept="image/*"
                style={{ display: 'none' }}
                onChange={handleFileChange}
              />
              <button className="bg-btn" onClick={() => fileInputRef.current?.click()}>
                🖼️ 自定义背景
              </button>
              {backgroundImage && (
                <div className="bg-reset" onClick={handleReset}>重置背景</div>
              )}
              {backgroundImage && (
                <div className="bg-opacity-slider">
                  <label>透明度</label>
                  <input
                    type="range"
                    min="0.3"
                    max="0.95"
                    step="0.01"
                    value={bgOpacity}
                    onChange={e => setBgOpacity(parseFloat(e.target.value))}
                  />
                </div>
              )}
            </div>
          </div>
        </aside>
        <main className="main-content">
          <Routes>
            <Route path="/" element={<Dashboard />} />
            <Route path="/graph" element={<KnowledgeGraph />} />
            <Route path="/snapshot" element={<SnapshotView />} />
            <Route path="/projects" element={<ProjectManager />} />
            <Route path="/code" element={<CodeEntities />} />
          </Routes>
        </main>
      </div>
    </div>
  )
}

export default App
