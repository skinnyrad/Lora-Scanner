import { useCallback, useEffect, useMemo, useState } from 'react'
import { useSearchParams } from 'react-router-dom'
import {
  Box, Button, Chip, FormControlLabel, Paper, Stack, Switch, TextField, ToggleButton, ToggleButtonGroup, Typography,
} from '@mui/material'
import FrameTable from '../components/FrameTable'
import PauseIcon from '@mui/icons-material/PauseRounded'
import PlayIcon from '@mui/icons-material/PlayArrowRounded'
import DownloadIcon from '@mui/icons-material/DownloadRounded'
import ClearIcon from '@mui/icons-material/DeleteSweepRounded'
import { useLive } from '../hooks/useLive'
import { useNotify } from '../hooks/useNotify'
import { dateTime, rssiLabel } from '../lib/format'
import { errMsg, type Frame, type FrameKind } from '../api/client'

const KINDS: { kind: FrameKind; label: string }[] = [
  { kind: 'lora', label: 'LoRa' },
  { kind: 'lorawan', label: 'LoRaWAN' },
  { kind: 'raw', label: 'Raw' },
]

function matches(f: Frame, tokens: string[]) {
  if (tokens.length === 0) return true
  const hay = `${f.id} ${f.kind} ${f.source} ${f.identifier} ${f.freq || ''} ${f.rssi ?? ''} ${f.info}`.toLowerCase()
  return tokens.every((t) => hay.includes(t))
}

export default function Frames() {
  const { frames, clearFrames } = useLive()
  const notify = useNotify()
  const [params, setParams] = useSearchParams()

  const [filter, setFilter] = useState(params.get('device') ?? '')
  const [kinds, setKinds] = useState<FrameKind[]>(['lora', 'lorawan', 'raw'])
  const [follow, setFollow] = useState(true)
  const [frozen, setFrozen] = useState<Frame[] | null>(null) // non-null while paused
  const [selectedId, setSelectedId] = useState<number | null>(null)

  // Arriving via /frames?device=X pre-fills the filter once, then the URL is just the page.
  useEffect(() => {
    if (params.has('device')) setParams({}, { replace: true })
  }, [params, setParams])

  const source = frozen ?? frames
  const rows = useMemo(() => {
    const tokens = filter.toLowerCase().split(/\s+/).filter(Boolean)
    return source.filter((f) => kinds.includes(f.kind) && matches(f, tokens))
  }, [source, filter, kinds])

  const stopFollowing = useCallback(() => setFollow(false), [])

  const selected = useMemo(() => source.find((f) => f.id === selectedId) ?? null, [source, selectedId])

  const clear = async () => {
    try {
      await clearFrames()
      setFrozen(null)
      setSelectedId(null)
    } catch (e) {
      notify(errMsg(e), 'error')
    }
  }

  return (
    <Stack spacing={2}>
      <Stack direction={{ xs: 'column', md: 'row' }} spacing={1} sx={{ alignItems: { md: 'center' } }}>
        <TextField size="small" label="Filter" placeholder="e.g. 27FD3566 lorawan" value={filter}
          onChange={(e) => setFilter(e.target.value)} sx={{ minWidth: 260 }} />
        <ToggleButtonGroup size="small" value={kinds} onChange={(_e, v: FrameKind[]) => setKinds(v)}>
          {KINDS.map((k) => <ToggleButton key={k.kind} value={k.kind}>{k.label}</ToggleButton>)}
        </ToggleButtonGroup>
        <FormControlLabel control={<Switch checked={follow} onChange={(e) => setFollow(e.target.checked)} />} label="Follow" />
        <Box sx={{ flex: 1 }} />
        <Chip size="small" label={`${rows.length} shown / ${frames.length} total`} />
        <Button size="small" startIcon={frozen ? <PlayIcon /> : <PauseIcon />} onClick={() => setFrozen(frozen ? null : frames)}>
          {frozen ? 'Resume' : 'Pause'}
        </Button>
        <Button size="small" component="a" href="/api/frames.csv" download="frames.csv" startIcon={<DownloadIcon />}>CSV</Button>
        <Button size="small" color="error" startIcon={<ClearIcon />}
          onClick={() => window.confirm('Clear all frames and devices?') && clear()}>Clear</Button>
      </Stack>

      <Box sx={{ height: 'calc(100vh - 300px)', minHeight: 280 }}>
        <FrameTable
          rows={rows}
          selectedId={selectedId}
          onSelect={setSelectedId}
          follow={follow}
          onUserScrolledAway={stopFollowing}
          emptyText={frozen ? 'No frames in the paused view' : 'No frames yet'}
        />
      </Box>

      <Paper variant="outlined" sx={{ p: 2, minHeight: 150 }}>
        {selected ? (
          <Stack direction={{ xs: 'column', md: 'row' }} spacing={3}>
            <Box sx={{ display: 'grid', gridTemplateColumns: 'auto 1fr', columnGap: 2, rowGap: 0.5, alignContent: 'start' }}>
              {([
                ['Frame', selected.id],
                ['Received', dateTime(selected.time)],
                ['Kind', selected.kind],
                ['Source', selected.source],
                ['Identifier', selected.identifier || '—'],
                ['Frequency', selected.freq ? `${selected.freq} MHz` : '—'],
                ['Signal', selected.kind === 'raw' ? '—' : rssiLabel(selected.rssi)],
              ] as const).map(([k, v]) => (
                <Box key={k} sx={{ display: 'contents' }}>
                  <Typography variant="body2" color="text.secondary">{k}</Typography>
                  <Typography variant="body2">{v}</Typography>
                </Box>
              ))}
            </Box>
            <Box component="pre" sx={{
              flex: 1, m: 0, p: 1, borderRadius: 1, bgcolor: 'action.hover', overflow: 'auto', maxHeight: 220,
              fontFamily: 'ui-monospace, Menlo, Consolas, monospace', fontSize: 13, whiteSpace: 'pre-wrap', wordBreak: 'break-all',
            }}>
              {selected.raw || selected.info}
            </Box>
          </Stack>
        ) : (
          <Typography color="text.secondary">Select a frame to see its details.</Typography>
        )}
      </Paper>
    </Stack>
  )
}
