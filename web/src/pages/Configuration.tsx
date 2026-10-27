import { Stack, Typography } from '@mui/material'
import TransceiverPanel from '../components/TransceiverPanel'
import AddTransceiver from '../components/AddTransceiver'
import GatewayConfig from '../components/GatewayConfig'
import { useLive } from '../hooks/useLive'

export default function Configuration() {
  const { transceivers } = useLive()
  return (
    <Stack spacing={3}>
      <Typography>
        Attach a Feather running the Skinny-LoRa firmware to receive raw LoRa packets, and add Dragino LoRaWAN
        gateways below to collect their traffic logs. Received frames show up on the Frames and Devices pages.
      </Typography>
      <Stack direction="row" spacing={2} sx={{ alignItems: 'center' }}>
        <AddTransceiver />
        {transceivers.length === 0 && <Typography color="text.secondary">No transceivers attached.</Typography>}
      </Stack>
      {transceivers.map((t) => <TransceiverPanel key={t.id} transceiver={t} />)}
      <GatewayConfig />
    </Stack>
  )
}
