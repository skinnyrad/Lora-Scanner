import { useState } from 'react'
import { Button, Card, CardContent, CardHeader, Chip, Divider, Stack, TextField, Typography } from '@mui/material'
import { api, errMsg, freqLabel, type Transceiver } from '../api/client'
import { useLive } from '../hooks/useLive'
import { useNotify } from '../hooks/useNotify'
import BeaconControl from '../components/BeaconControl'
import { portTitle } from '../theme'

function TransmitCard({ t }: { t: Transceiver }) {
  const notify = useNotify()
  const [text, setText] = useState('')

  const send = async () => {
    try {
      await api.transmit(t.id, text)
      notify(`Transmitted on ${t.port}`)
    } catch (e) {
      notify(errMsg(e), 'error')
    }
  }

  return (
    <Card variant="outlined">
      <CardHeader title={t.port} slotProps={{ title: portTitle }} action={<Chip size="small" color={t.freq === null ? 'warning' : 'success'} label={freqLabel(t.freq)} />} />
      <CardContent sx={{ pt: 0 }}>
        <Stack spacing={2}>
          <Stack direction={{ xs: 'column', sm: 'row' }} spacing={1}>
            <TextField size="small" label="Data" value={text} onChange={(e) => setText(e.target.value)} sx={{ flex: 1 }}
              onKeyDown={(e) => e.key === 'Enter' && text && send()} />
            <Button variant="contained" disabled={!text} onClick={send}>Send</Button>
          </Stack>
          <Divider />
          <BeaconControl id={t.id} />
        </Stack>
      </CardContent>
    </Card>
  )
}

export default function Transmit() {
  const { transceivers } = useLive()
  return (
    <Stack spacing={3}>
      <Typography>Send a one-off message, or start a beacon that repeats it on an interval. Beacons run on the server and keep going if you close this page.</Typography>
      {transceivers.length === 0 && <Typography color="text.secondary">No transceivers attached. Add one on the Configuration page.</Typography>}
      {transceivers.map((t) => <TransmitCard key={t.id} t={t} />)}
    </Stack>
  )
}
