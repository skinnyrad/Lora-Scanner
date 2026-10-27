import { createTheme } from '@mui/material'

// Palette taken from the original Flask app's Bootstrap theme.
export const SIDEBAR = '#006666' // bg-primary (rgb 0,102,102)
export const SIDEBAR_WIDTH = 208 // 13rem

export const theme = createTheme({
  palette: {
    mode: 'light',
    primary: { main: '#009999', contrastText: '#fff' },
    secondary: { main: '#6c757d' },
    success: { main: '#198754' },
    info: { main: '#0dcaf0' },
    warning: { main: '#ffc107' },
    error: { main: '#dc3545' },
    background: { default: '#fff', paper: '#fff' },
    text: { primary: '#212529' },
  },
  typography: {
    fontFamily: 'Muli, -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif',
    h1: { fontWeight: 500, lineHeight: 1.1 },
    h4: { fontWeight: 500 },
    h5: { fontWeight: 500 },
    h6: { fontWeight: 500 },
    button: { textTransform: 'none' },
  },
})

// Serial port paths in card headers: compact and allowed to wrap.
export const portTitle = { variant: 'subtitle1', sx: { fontFamily: 'ui-monospace, Menlo, Consolas, monospace', fontWeight: 600, wordBreak: 'break-all' } } as const
