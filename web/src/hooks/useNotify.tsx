import { createContext, useCallback, useContext, useState, type ReactNode } from 'react'
import { Alert, Snackbar, type AlertColor } from '@mui/material'

type Notify = (message: string, severity?: AlertColor) => void
const Ctx = createContext<Notify>(() => {})

export function NotifyProvider({ children }: { children: ReactNode }) {
  const [toast, setToast] = useState<{ message: string; severity: AlertColor } | null>(null)
  const notify = useCallback<Notify>((message, severity = 'success') => setToast({ message, severity }), [])
  return (
    <Ctx.Provider value={notify}>
      {children}
      <Snackbar open={!!toast} autoHideDuration={4000} onClose={() => setToast(null)}>
        {toast ? (
          <Alert severity={toast.severity} variant="filled" onClose={() => setToast(null)}>
            {toast.message}
          </Alert>
        ) : undefined}
      </Snackbar>
    </Ctx.Provider>
  )
}

export const useNotify = () => useContext(Ctx)
