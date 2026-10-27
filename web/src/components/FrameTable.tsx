import { useEffect, useLayoutEffect, useRef, useState } from 'react'
import { Box } from '@mui/material'
import { clockTime, rssiLabel } from '../lib/format'
import type { Frame } from '../api/client'

const ROW_H = 28
const OVERSCAN = 12
const COLS = '70px 110px 190px 220px 80px 90px 80px minmax(120px, 1fr)'
const MONO = 'ui-monospace, SFMono-Regular, Menlo, Consolas, monospace'

// Wireshark-style row tints.
const tint = { raw: '#f1f3f4', lora: '#dff3f3', lorawan: '#e4eefb' } as const

const cell = { px: 1, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' } as const

interface Props {
  rows: Frame[]
  selectedId: number | null
  onSelect: (id: number) => void
  follow: boolean
  onUserScrolledAway: () => void // the user scrolled off the newest row
  emptyText: string
}

// A windowed list: only the visible rows exist in the DOM, so the whole
// 10,000-frame buffer scrolls smoothly. (MUI's community DataGrid caps a page at 100 rows.)
export default function FrameTable({ rows, selectedId, onSelect, follow, onUserScrolledAway, emptyText }: Props) {
  const ref = useRef<HTMLDivElement>(null)
  const [scrollTop, setScrollTop] = useState(0)
  const [height, setHeight] = useState(400)
  const [gutter, setGutter] = useState(0) // scrollbar width, so the header lines up with the rows

  useLayoutEffect(() => {
    const el = ref.current!
    const measure = () => {
      setHeight(el.clientHeight)
      setGutter(el.offsetWidth - el.clientWidth)
    }
    measure()
    const ro = new ResizeObserver(measure)
    ro.observe(el)
    return () => ro.disconnect()
  }, [])

  // Follow: pin to the newest row after every change.
  useLayoutEffect(() => {
    const el = ref.current
    if (follow && el) {
      el.scrollTop = el.scrollHeight
      setScrollTop(el.scrollTop)
    }
  }, [rows, follow])

  useEffect(() => {
    const el = ref.current!
    const onScroll = () => {
      setScrollTop(el.scrollTop)
      const atBottom = el.scrollTop + el.clientHeight >= el.scrollHeight - ROW_H
      if (!atBottom) onUserScrolledAway()
    }
    el.addEventListener('scroll', onScroll, { passive: true })
    return () => el.removeEventListener('scroll', onScroll)
  }, [onUserScrolledAway])

  const first = Math.max(0, Math.floor(scrollTop / ROW_H) - OVERSCAN)
  const last = Math.min(rows.length, Math.ceil((scrollTop + height) / ROW_H) + OVERSCAN)
  const visible = rows.slice(first, last)

  return (
    <Box role="table" aria-rowcount={rows.length} sx={{
      height: '100%', display: 'flex', flexDirection: 'column', overflow: 'hidden',
      border: 1, borderColor: 'divider', borderRadius: 1, bgcolor: 'background.paper', fontFamily: MONO, fontSize: 13,
    }}>
      {/* The header is a plain sibling above the scroller (not sticky), so nothing can scroll under or around it. */}
      <Box role="row" sx={{
        display: 'grid', gridTemplateColumns: COLS, flex: 'none', height: ROW_H + 4, alignItems: 'center', pr: `${gutter}px`,
        borderBottom: 1, borderColor: 'divider', fontFamily: 'inherit', fontWeight: 600,
      }}>
        {['No.', 'Time', 'Source', 'Identifier', 'Freq', 'RSSI', 'Kind', 'Info'].map((h) => <Box key={h} role="columnheader" sx={cell}>{h}</Box>)}
      </Box>

      <Box ref={ref} sx={{ flex: 1, minHeight: 0, overflowY: 'scroll', overflowX: 'hidden' }}>
        {rows.length === 0 ? (
          <Box sx={{ p: 2, color: 'text.secondary', fontFamily: 'inherit' }}>{emptyText}</Box>
        ) : (
          <Box sx={{ height: rows.length * ROW_H, position: 'relative' }}>
            <Box sx={{ position: 'absolute', top: first * ROW_H, left: 0, right: 0 }}>
              {visible.map((f) => {
                const sel = f.id === selectedId
                return (
                  <Box key={f.id} role="row" data-id={f.id} data-kind={f.kind} aria-selected={sel} onClick={() => onSelect(f.id)} sx={{
                    display: 'grid', gridTemplateColumns: COLS, height: ROW_H, alignItems: 'center', cursor: 'pointer',
                    bgcolor: sel ? 'primary.main' : tint[f.kind], color: sel ? 'primary.contrastText' : f.kind === 'raw' ? 'text.secondary' : 'text.primary',
                    '&:hover': { filter: 'brightness(0.96)' },
                  }}>
                    <Box sx={cell}>{f.id}</Box>
                    <Box sx={cell}>{clockTime(f.time)}</Box>
                    <Box sx={cell} title={f.source}>{f.source}</Box>
                    <Box sx={cell} title={f.identifier}>{f.identifier}</Box>
                    <Box sx={cell}>{f.freq ? f.freq : ''}</Box>
                    <Box sx={cell}>{f.kind === 'raw' ? '' : rssiLabel(f.rssi)}</Box>
                    <Box sx={cell}>{f.kind}</Box>
                    <Box sx={cell} title={f.info}>{f.info}</Box>
                  </Box>
                )
              })}
            </Box>
          </Box>
        )}
      </Box>
    </Box>
  )
}
