import { useMemo } from 'react'
import { useNavigate } from 'react-router-dom'
import { Stack, Typography } from '@mui/material'
import { DataGrid, type GridColDef } from '@mui/x-data-grid'
import { useLive } from '../hooks/useLive'
import { dateTime, rssiLabel } from '../lib/format'
import type { Device } from '../api/client'

const columns: GridColDef<Device>[] = [
  { field: 'identifier', headerName: 'Identifier', flex: 1, minWidth: 200 },
  { field: 'freq', headerName: 'Frequency', width: 120, type: 'number', align: 'left', headerAlign: 'left', valueFormatter: (v: number) => `${v} MHz` },
  {
    field: 'rssi', headerName: 'Signal Strength', width: 150,
    valueGetter: (_v, row) => row.rssi ?? 0, valueFormatter: (_v, row) => rssiLabel(row.rssi),
  },
  { field: 'firstSeen', headerName: 'First Seen', width: 190, valueGetter: (v: string) => new Date(v), valueFormatter: (_v, row) => dateTime(row.firstSeen), type: 'dateTime' },
  { field: 'lastSeen', headerName: 'Last Seen', width: 190, valueGetter: (v: string) => new Date(v), valueFormatter: (_v, row) => dateTime(row.lastSeen), type: 'dateTime' },
  { field: 'frames', headerName: 'Frames Seen', width: 130, type: 'number', align: 'left', headerAlign: 'left' },
]

export default function Devices() {
  const { devices } = useLive()
  const navigate = useNavigate()
  const rows = useMemo(() => Object.values(devices), [devices])

  return (
    <Stack spacing={2}>
      <Typography color="text.secondary">Every device heard so far, updated live. Select a row to see its frames.</Typography>
      <DataGrid
        rows={rows}
        columns={columns}
        getRowId={(r) => r.identifier}
        density="compact"
        autoHeight
        disableColumnMenu
        hideFooter={rows.length <= 100}
        initialState={{ sorting: { sortModel: [{ field: 'lastSeen', sort: 'desc' }] } }}
        onRowClick={(p) => navigate(`/frames?device=${encodeURIComponent(p.row.identifier)}`)}
        sx={{ '& .MuiDataGrid-row': { cursor: 'pointer' }, bgcolor: 'background.paper' }}
        localeText={{ noRowsLabel: 'No devices yet' }}
      />
    </Stack>
  )
}
