import { useState } from 'react'
import { NavLink, Outlet } from 'react-router-dom'
import { Avatar, Box, Chip, Collapse, IconButton, List, ListItemButton, ListItemIcon, ListItemText, Typography } from '@mui/material'
import MenuIcon from '@mui/icons-material/MenuRounded'
import SettingsIcon from '@mui/icons-material/SettingsRounded'
import ViewListIcon from '@mui/icons-material/ViewListRounded'
import SensorsIcon from '@mui/icons-material/SensorsRounded'
import SendIcon from '@mui/icons-material/SendRounded'
import { useLive } from '../hooks/useLive'
import { SIDEBAR, SIDEBAR_WIDTH } from '../theme'
import brandIcon from '../assets/brand-icon.svg'

const NAV = [
  { to: '/', label: 'Configuration', icon: <SettingsIcon />, end: true },
  { to: '/frames', label: 'Frames', icon: <ViewListIcon />, end: false },
  { to: '/devices', label: 'Devices', icon: <SensorsIcon />, end: false },
  { to: '/transmit', label: 'Transmit', icon: <SendIcon />, end: false },
]

// Same look as the Flask "sideNav": dark teal bar, uppercase links that
// brighten on hover/active, circular logo on desktop.
const linkSx = {
  justifyContent: 'flex-start',
  py: 1,
  px: 2,
  color: 'rgba(255,255,255,.55)',
  '&:hover': { color: 'rgba(255,255,255,.75)', bgcolor: 'transparent' },
  '&.active': { color: '#fff' },
  '& .MuiListItemIcon-root': { color: 'inherit', minWidth: 36 },
  '& .MuiListItemText-primary': { fontWeight: 800, letterSpacing: '0.05rem', textTransform: 'uppercase', fontSize: '0.9rem' },
} as const

export default function Layout() {
  const [open, setOpen] = useState(false)
  const { connected } = useLive()
  const close = () => setOpen(false)

  const links = (
    <List sx={{ width: '100%' }}>
      {NAV.map((n) => (
        <ListItemButton
          key={n.label}
          component={NavLink}
          end={n.end}
          to={n.to}
          onClick={close}
          sx={linkSx}
        >
          <ListItemIcon>{n.icon}</ListItemIcon>
          <ListItemText primary={n.label} />
        </ListItemButton>
      ))}
    </List>
  )

  const live = (
    <Chip size="small" label={connected ? 'Live' : 'Reconnecting…'}
      sx={{ bgcolor: connected ? 'rgba(255,255,255,.2)' : 'warning.main', color: connected ? '#fff' : '#212529' }} />
  )

  return (
    <Box sx={{ display: { md: 'flex' } }}>
      {/* Desktop sidebar */}
      <Box component="nav" sx={{
        display: { xs: 'none', md: 'flex' }, flexDirection: 'column', alignItems: 'center',
        position: 'fixed', top: 0, left: 0, width: SIDEBAR_WIDTH, height: '100vh', bgcolor: SIDEBAR,
      }}>
        <Avatar src={brandIcon} alt="Skinny R&D" sx={{
          width: 112, height: 112, mt: 'auto', mb: 1, bgcolor: '#fff', border: '0.4rem solid rgba(255,255,255,.2)', boxSizing: 'content-box',
          // The artwork reaches the corners of its square, so inset it to stay inside the circle.
          '& img': { objectFit: 'contain', p: '16px', boxSizing: 'border-box' },
        }} />
        <Box sx={{ width: '100%', mb: 'auto' }}>{links}</Box>
        <Box sx={{ pb: 2 }}>{live}</Box>
      </Box>

      {/* Mobile top bar */}
      <Box component="nav" sx={{ display: { xs: 'block', md: 'none' }, position: 'fixed', top: 0, left: 0, right: 0, zIndex: 1100, bgcolor: SIDEBAR }}>
        <Box sx={{ display: 'flex', alignItems: 'center', px: 2, py: 1 }}>
          <Typography sx={{ flex: 1, color: '#fff', fontSize: '1.25rem' }}>LoRa Scanner</Typography>
          <Box sx={{ mr: 1 }}>{live}</Box>
          <IconButton aria-label="Toggle navigation" onClick={() => setOpen(!open)}
            sx={{ color: 'rgba(255,255,255,.55)', border: '1px solid rgba(255,255,255,.1)', borderRadius: 1 }}>
            <MenuIcon />
          </IconButton>
        </Box>
        <Collapse in={open}>{links}</Collapse>
      </Box>

      <Box component="main" sx={{
        flex: 1, minWidth: 0, ml: { md: `${SIDEBAR_WIDTH}px` },
        px: { xs: 2, md: 3 }, pt: { xs: 9, md: 3 }, pb: 4,
      }}>
        <Outlet />
      </Box>
    </Box>
  )
}
