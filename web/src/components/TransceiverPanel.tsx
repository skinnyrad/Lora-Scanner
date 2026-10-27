import { useState } from 'react'
import { Button, Card, CardContent, CardHeader, Chip, Stack, Typography } from '@mui/material'
import { api, errMsg, freqLabel, type Transceiver } from '../api/client'
import { useNotify } from '../hooks/useNotify'
import { portTitle } from '../theme'
import SerialViewer from './SerialViewer'

// Connection management only; transmitting lives on the Transmit page.
export default function TransceiverPanel({ transceiver: t }: { transceiver: Transceiver }) {
  const notify = useNotify()
  const [viewing, setViewing] = useState(false)

  const remove = async () => {
    if (!window.confirm(`Remove ${t.port}?`)) return
    try {
      await api.removeTransceiver(t.id)
      notify('Transceiver removed')
    } catch (e) {
      notify(errMsg(e), 'error')
    }
  }

  return (
    <Card variant="outlined">
      <CardHeader
        title={t.port}
        slotProps={{ title: portTitle }}
        action={<Chip size="small" color={t.freq === null ? 'warning' : 'success'} label={freqLabel(t.freq)} />}
      />
      <CardContent sx={{ pt: 0 }}>
        {t.freq === null && (
          <Typography variant="body2" color="text.secondary" sx={{ mb: 1.5 }}>
            The Feather reports its frequency once at boot. Unplug and replug it (with this page open) to detect it.
          </Typography>
        )}
        <Stack direction="row" spacing={1}>
          <Button variant="contained" onClick={() => setViewing(true)}>View Serial</Button>
          <Button variant="outlined" color="error" onClick={remove}>Remove</Button>
        </Stack>
      </CardContent>
      <SerialViewer transceiver={t} open={viewing} onClose={() => setViewing(false)} />
    </Card>
  )
}
