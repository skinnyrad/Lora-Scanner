import { useEffect, useMemo, useRef, useState } from 'react'
import { Box, Button, Dialog, DialogActions, DialogContent, DialogTitle, FormControlLabel, Switch } from '@mui/material'
import { freqLabel, type Transceiver } from '../api/client'
import { useLive } from '../hooks/useLive'

// Read-only view of everything the device has printed, taken from the live frame store.
export default function SerialViewer({ transceiver, open, onClose }: { transceiver: Transceiver; open: boolean; onClose: () => void }) {
  const { frames } = useLive()
  const [clearedAt, setClearedAt] = useState(0) // hide frames up to this id
  const [autoscroll, setAutoscroll] = useState(true)
  const ref = useRef<HTMLDivElement>(null)

  const lines = useMemo(
    () => frames.filter((f) => f.kind === 'raw' && f.transceiver === transceiver.id && f.id > clearedAt),
    [frames, transceiver.id, clearedAt],
  )

  useEffect(() => {
    if (open && autoscroll && ref.current) ref.current.scrollTop = ref.current.scrollHeight
  }, [lines, autoscroll, open])

  return (
    <Dialog open={open} onClose={onClose} fullWidth maxWidth="md">
      <DialogTitle sx={{ fontSize: '1.1rem', wordBreak: 'break-all' }}>Serial output: {transceiver.port} ({freqLabel(transceiver.freq)})</DialogTitle>
      <DialogContent>
        <Box ref={ref} sx={{
          height: 420, overflowY: 'auto', p: 1, borderRadius: 1, bgcolor: 'action.hover',
          fontFamily: 'ui-monospace, Menlo, Consolas, monospace', fontSize: 13, whiteSpace: 'pre-wrap', wordBreak: 'break-all',
        }}>
          {lines.length === 0 && <Box sx={{ color: 'text.secondary' }}>Nothing received yet.</Box>}
          {lines.map((f) => <div key={f.id}>{f.info}</div>)}
        </Box>
      </DialogContent>
      <DialogActions>
        <FormControlLabel control={<Switch checked={autoscroll} onChange={(e) => setAutoscroll(e.target.checked)} />} label="Autoscroll" />
        <Box sx={{ flex: 1 }} />
        <Button onClick={() => setClearedAt(lines.length ? lines[lines.length - 1].id : clearedAt)}>Clear View</Button>
        <Button variant="contained" onClick={onClose}>Close</Button>
      </DialogActions>
    </Dialog>
  )
}
