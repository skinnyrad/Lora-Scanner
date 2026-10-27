import { useCallback, useEffect, useState } from 'react'
import { Button, Chip, Stack, TextField, Typography } from '@mui/material'
import { api, errMsg, type BeaconState } from '../api/client'
import { useNotify } from '../hooks/useNotify'

export default function BeaconControl({ id }: { id: number }) {
  const notify = useNotify()
  const [state, setState] = useState<BeaconState>({ running: false, text: '', intervalMs: 0 })
  const [text, setText] = useState('')
  const [interval, setIntervalMs] = useState('5000')

  const refresh = useCallback(() => api.beacon(id).then(setState).catch(() => {}), [id])
  // The server owns the timer, so re-sync after reloads and on connection changes.
  useEffect(() => {
    refresh()
    const t = window.setInterval(refresh, 5000)
    return () => window.clearInterval(t)
  }, [refresh])

  const ms = Number(interval)
  const valid = text.length > 0 && Number.isFinite(ms) && ms >= 1000

  const act = async (fn: () => Promise<BeaconState>) => {
    try {
      setState(await fn())
    } catch (e) {
      notify(errMsg(e), 'error')
    }
  }

  return (
    <Stack spacing={1}>
      <Stack direction="row" spacing={1} sx={{ alignItems: 'center' }}>
        <Typography variant="subtitle2">Beacon</Typography>
        <Chip size="small" color={state.running ? 'success' : 'default'}
          label={state.running ? `Every ${state.intervalMs} ms: ${state.text}` : 'Stopped'} />
      </Stack>
      <Stack direction={{ xs: 'column', sm: 'row' }} spacing={1}>
        <TextField size="small" label="Beacon data" value={text} onChange={(e) => setText(e.target.value)} sx={{ flex: 1 }} />
        <TextField size="small" label="Interval (ms)" value={interval} onChange={(e) => setIntervalMs(e.target.value)}
          error={interval !== '' && !(ms >= 1000)} helperText={ms >= 1000 ? undefined : 'Minimum 1000'} />
        <Button variant="contained" disabled={!valid} onClick={() => act(() => api.startBeacon(id, text, ms))}>
          Start
        </Button>
        <Button variant="outlined" color="error" disabled={!state.running} onClick={() => act(() => api.stopBeacon(id))}>
          Stop
        </Button>
      </Stack>
    </Stack>
  )
}
