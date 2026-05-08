import { createContext, useContext, useState, useCallback, type ReactNode } from 'react'

interface BackgroundContextValue {
  backgroundImage: string | null
  bgOpacity: number
  setBgOpacity: (opacity: number) => void
  setBackground: (file: File) => Promise<string | null>
  clearBackground: () => void
}

const BackgroundContext = createContext<BackgroundContextValue>({
  backgroundImage: null,
  bgOpacity: 0.7,
  setBgOpacity: () => {},
  setBackground: async () => null,
  clearBackground: () => {},
})

const STORAGE_KEY = 'chronodraft-bg'
const OPACITY_KEY = 'chronodraft-bg-opacity'
const MAX_SIZE = 3 * 1024 * 1024 // 3MB

export function BackgroundProvider({ children }: { children: ReactNode }) {
  const [backgroundImage, setBackgroundImage] = useState<string | null>(() => {
    try {
      return localStorage.getItem(STORAGE_KEY)
    } catch {
      return null
    }
  })

  const [bgOpacity, setBgOpacityState] = useState<number>(() => {
    try {
      const v = localStorage.getItem(OPACITY_KEY)
      return v ? parseFloat(v) : 0.7
    } catch {
      return 0.7
    }
  })

  const setBackground = useCallback((file: File): Promise<string | null> => {
    return new Promise((resolve) => {
      if (file.size > MAX_SIZE) {
        resolve(null)
        return
      }
      if (!file.type.startsWith('image/')) {
        resolve(null)
        return
      }
      const reader = new FileReader()
      reader.onload = () => {
        const dataUrl = reader.result as string
        setBackgroundImage(dataUrl)
        try {
          localStorage.setItem(STORAGE_KEY, dataUrl)
        } catch {
          // storage full - still set in memory
        }
        resolve(dataUrl)
      }
      reader.onerror = () => resolve(null)
      reader.readAsDataURL(file)
    })
  }, [])

  const clearBackground = useCallback(() => {
    setBackgroundImage(null)
    try {
      localStorage.removeItem(STORAGE_KEY)
    } catch {
      // ignore
    }
  }, [])

  const setBgOpacity = useCallback((v: number) => {
    setBgOpacityState(v)
    try {
      localStorage.setItem(OPACITY_KEY, String(v))
    } catch {
      // ignore
    }
  }, [])

  return (
    <BackgroundContext.Provider value={{ backgroundImage, bgOpacity, setBgOpacity, setBackground, clearBackground }}>
      {children}
    </BackgroundContext.Provider>
  )
}

export function useBackground() {
  return useContext(BackgroundContext)
}
