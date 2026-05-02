import { createContext, useContext, useState, useCallback, useRef, type ReactNode } from 'react'

export type ToastType = 'success' | 'error' | 'warning' | 'info'

export interface ToastItem {
  id: string
  type: ToastType
  message: string
}

interface ToastContextValue {
  success: (message: string) => void
  error: (message: string) => void
  warning: (message: string) => void
  info: (message: string) => void
  toasts: ToastItem[]
  removeToast: (id: string) => void
}

const DURATION: Record<ToastType, number> = {
  success: 3000,
  error: 5000,
  warning: 4000,
  info: 3000,
}

const MAX_TOASTS = 5

const ToastContext = createContext<ToastContextValue | null>(null)

export function useToast(): ToastContextValue {
  const ctx = useContext(ToastContext)
  if (!ctx) throw new Error('useToast must be used within ToastProvider')
  return ctx
}

export function ToastProvider({ children }: { children: ReactNode }) {
  const [toasts, setToasts] = useState<ToastItem[]>([])
  const counterRef = useRef(0)

  const removeToast = useCallback((id: string) => {
    setToasts((prev) => prev.filter((t) => t.id !== id))
  }, [])

  const addToast = useCallback(
    (type: ToastType, message: string) => {
      const id = `toast-${++counterRef.current}`
      setToasts((prev) => {
        const next = [...prev, { id, type, message }]
        if (next.length > MAX_TOASTS) return next.slice(next.length - MAX_TOASTS)
        return next
      })
      setTimeout(() => removeToast(id), DURATION[type])
    },
    [removeToast],
  )

  const success = useCallback((msg: string) => addToast('success', msg), [addToast])
  const error = useCallback((msg: string) => addToast('error', msg), [addToast])
  const warning = useCallback((msg: string) => addToast('warning', msg), [addToast])
  const info = useCallback((msg: string) => addToast('info', msg), [addToast])

  return (
    <ToastContext.Provider value={{ success, error, warning, info, toasts, removeToast }}>
      {children}
    </ToastContext.Provider>
  )
}
