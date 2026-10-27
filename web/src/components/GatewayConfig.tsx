import { useEffect, useState } from 'react'
import {
  Box, Button, Card, CardContent, CardHeader, IconButton, List, ListItem, ListItemIcon, ListItemText, Stack, TextField, Tooltip, Typography,
} from '@mui/material'
import CircleIcon from '@mui/icons-material/CircleRounded'
import AddIcon from '@mui/icons-material/AddRounded'
import DeleteIcon from '@mui/icons-material/DeleteRounded'
import { api, errMsg, type GatewayStatus } from '../api/client'
import { useLive } from '../hooks/useLive'
import { useNotify } from '../hooks/useNotify'

const MIN_POLL = 5
const MAX_POLL = 3600

const IPV4 = /^(?:(?:25[0-5]|2[0-4]\d|1?\d?\d)\.){3}(?:25[0-5]|2[0-4]\d|1?\d?\d)$/

// Green = online, red = two polls in a row failed (retried every 5s), grey = not polled yet.
function StatusDot({ status }: { status?: GatewayStatus }) {
  const state = status?.state ?? 'unknown'
  const color = state === 'online' ? 'success' : state === 'offline' ? 'error' : 'disabled'
  const title =
    state === 'online' ? `Online. Last poll ${new Date(status!.lastSuccess!).toLocaleTimeString()}`
    : state === 'offline' ? `Offline: ${status!.error ?? 'poll failed'}. Retrying every 5 seconds.`
    : 'Not polled yet'
  return <CircleIcon fontSize="small" color={color} titleAccess={title} />
}

export default function GatewayConfig() {
  const notify = useNotify()
  const { gatewayStatus } = useLive()
  const [ips, setIps] = useState<string[]>([])
  const [input, setInput] = useState('')
  const [saved, setSaved] = useState<number | null>(null) // interval on the server, seconds
  const [poll, setPoll] = useState('')

  useEffect(() => {
    api.gateways().then(setIps).catch((e) => notify(errMsg(e), 'error'))
    api.pollInterval().then((s) => { setSaved(s); setPoll(String(s)) }).catch(() => {})
  }, [notify])

  const ip = input.trim()
  const invalid = ip !== '' && !IPV4.test(ip)
  const duplicate = ips.includes(ip)

  const add = async () => {
    try {
      const added = await api.addGateway(ip)
      setIps((l) => [...l, added.ip])
      setInput('')
      notify(`Added ${added.ip}`)
    } catch (e) {
      notify(errMsg(e), 'error')
    }
  }

  const pollNum = Number(poll)
  const pollValid = poll !== '' && Number.isInteger(pollNum) && pollNum >= MIN_POLL && pollNum <= MAX_POLL

  const savePoll = async () => {
    try {
      const s = await api.setPollInterval(pollNum)
      setSaved(s)
      setPoll(String(s))
      notify(`Polling every ${s} seconds`)
    } catch (e) {
      notify(errMsg(e), 'error')
    }
  }

  const remove = async (target: string) => {
    try {
      await api.removeGateway(target)
      setIps((l) => l.filter((x) => x !== target))
      notify(`Removed ${target}`)
    } catch (e) {
      notify(errMsg(e), 'error')
    }
  }

  return (
    <Card variant="outlined">
      <CardHeader title="LoRaWAN Gateways" slotProps={{ title: { variant: 'h6' } }} subheader="Dragino gateways are polled for new traffic on an interval" />
      <CardContent sx={{ pt: 0 }}>
        <Stack direction="row" spacing={1} sx={{ alignItems: 'flex-start', mb: 3 }}>
          <TextField size="small" type="number" label="Poll interval (seconds)" value={poll}
            onChange={(e) => setPoll(e.target.value)}
            onKeyDown={(e) => e.key === 'Enter' && pollValid && poll !== String(saved) && savePoll()}
            slotProps={{ htmlInput: { min: MIN_POLL, max: MAX_POLL } }}
            error={poll !== '' && !pollValid}
            helperText={poll !== '' && !pollValid ? `Whole seconds, ${MIN_POLL} to ${MAX_POLL}` : undefined}
            sx={{ width: 220 }} />
          <Button variant="outlined" disabled={!pollValid || poll === String(saved)} onClick={savePoll} sx={{ mt: '1px' }}>
            Save
          </Button>
        </Stack>
        <Stack direction="row" spacing={1} sx={{ alignItems: 'flex-start' }}>
          <TextField size="small" label="Gateway IP" placeholder="192.168.1.100" value={input}
            onChange={(e) => setInput(e.target.value)}
            onKeyDown={(e) => e.key === 'Enter' && ip && !invalid && !duplicate && add()}
            error={invalid || duplicate}
            helperText={invalid ? 'Invalid IPv4 address' : duplicate ? 'Already added' : undefined} />
          <Button variant="contained" startIcon={<AddIcon />} disabled={!ip || invalid || duplicate} onClick={add} sx={{ mt: '1px' }}>
            Add Gateway
          </Button>
        </Stack>
        {ips.length === 0 ? (
          <Typography color="text.secondary" sx={{ mt: 2 }}>No gateways added.</Typography>
        ) : (
          <Box sx={{ mt: 1 }}>
            <List dense disablePadding>
              {ips.map((g) => (
                <ListItem key={g} divider
                  secondaryAction={
                    <Tooltip title="Remove">
                      <IconButton edge="end" aria-label={`Remove ${g}`} onClick={() => remove(g)}><DeleteIcon /></IconButton>
                    </Tooltip>
                  }>
                  <ListItemIcon sx={{ minWidth: 32 }}>
                    <StatusDot status={gatewayStatus[g]} />
                  </ListItemIcon>
                  <ListItemText primary={g} slotProps={{ primary: { sx: { fontFamily: 'ui-monospace, Menlo, Consolas, monospace' } } }} />
                </ListItem>
              ))}
            </List>
          </Box>
        )}
      </CardContent>
    </Card>
  )
}
