import { useToast, type ToastType } from '../contexts/ToastContext'

const ICONS: Record<ToastType, string> = {
  success: '✅',
  error: '❌',
  warning: '⚠️',
  info: 'ℹ️',
}

const COLORS: Record<ToastType, string> = {
  success: '#238636',
  error: '#da3633',
  warning: '#d29922',
  info: '#58a6ff',
}

export default function Toast() {
  const { toasts, removeToast } = useToast()

  return (
    <div
      style={{
        position: 'fixed',
        top: 16,
        right: 16,
        zIndex: 9999,
        display: 'flex',
        flexDirection: 'column',
        gap: 8,
      }}
    >
      {toasts.map((toast) => (
        <div
          key={toast.id}
          style={{
            display: 'flex',
            alignItems: 'center',
            gap: 8,
            padding: '12px 16px',
            borderRadius: 8,
            backgroundColor: COLORS[toast.type],
            color: '#ffffff',
            boxShadow: '0 4px 12px rgba(0, 0, 0, 0.4)',
            minWidth: 260,
            maxWidth: 400,
            animation: 'toast-fade-in 0.3s ease-out',
          }}
        >
          <span style={{ fontSize: 16 }}>{ICONS[toast.type]}</span>
          <span style={{ flex: 1, fontSize: 14, lineHeight: 1.4 }}>{toast.message}</span>
          <button
            onClick={() => removeToast(toast.id)}
            style={{
              background: 'none',
              border: 'none',
              color: '#ffffff',
              cursor: 'pointer',
              fontSize: 16,
              padding: 0,
              opacity: 0.7,
              lineHeight: 1,
            }}
            aria-label="Close"
          >
            ×
          </button>
        </div>
      ))}
    </div>
  )
}
