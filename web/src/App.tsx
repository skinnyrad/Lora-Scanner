import { BrowserRouter, Navigate, Route, Routes } from 'react-router-dom'
import { CssBaseline, ThemeProvider } from '@mui/material'
import { theme } from './theme'
import { LiveProvider } from './hooks/useLive'
import { NotifyProvider } from './hooks/useNotify'
import Layout from './components/Layout'
import Configuration from './pages/Configuration'
import Frames from './pages/Frames'
import Devices from './pages/Devices'
import Transmit from './pages/Transmit'

export default function App() {
  return (
    <ThemeProvider theme={theme}>
      <CssBaseline />
      <NotifyProvider>
        <LiveProvider>
          <BrowserRouter>
            <Routes>
              <Route element={<Layout />}>
                <Route index element={<Configuration />} />
                <Route path="frames" element={<Frames />} />
                <Route path="devices" element={<Devices />} />
                <Route path="transmit" element={<Transmit />} />
                <Route path="*" element={<Navigate to="/" replace />} />
              </Route>
            </Routes>
          </BrowserRouter>
        </LiveProvider>
      </NotifyProvider>
    </ThemeProvider>
  )
}
