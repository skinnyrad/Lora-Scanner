import { useState } from 'react'
import { Button, Dialog, DialogActions, DialogContent, DialogTitle, MenuItem, TextField } from '@mui/material'
import AddIcon from '@mui/icons-material/AddRounded'
import { api, errMsg } from '../api/client'
import { useLive } from '../hooks/useLive'
import { useNotify } from '../hooks/useNotify'

export default function AddTransceiver() {
  const { transceivers } = useLive()
  const notify = useNotify()
  const [open, setOpen] = useState(false)
  const [ports, setPorts] = useState<string[]>([])
  const [port, setPort] = useState('')

  const show = async () => {
    try {
      const inUse = new Set(transceivers.map((t) => t.port))
      const free = (await api.ports()).filter((p) => !inUse.has(p))
      setPorts(free)
      setPort(free[0] ?? '')
      setOpen(true)
    } catch (e) {
      notify(errMsg(e), 'error')
    }
  }

  const add = async () => {
    setOpen(false)
    try {
      await api.addTransceiver(port)
      notify(`Added ${port}`)
    } catch (e) {
      notify(errMsg(e), 'error')
    }
  }

  return (
    <>
      <Button variant="contained" startIcon={<AddIcon />} onClick={show}>Add Transceiver</Button>
      <Dialog open={open} onClose={() => setOpen(false)} fullWidth maxWidth="xs">
        <DialogTitle>Add transceiver</DialogTitle>
        <DialogContent>
          <TextField select fullWidth margin="dense" label="Serial port" value={port} onChange={(e) => setPort(e.target.value)}
            helperText={ports.length ? 'The frequency is detected automatically.' : 'No unused serial ports found'}>
            {ports.map((p) => <MenuItem key={p} value={p}>{p}</MenuItem>)}
          </TextField>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setOpen(false)}>Cancel</Button>
          <Button variant="contained" disabled={!port} onClick={add}>Add</Button>
        </DialogActions>
      </Dialog>
    </>
  )
}
