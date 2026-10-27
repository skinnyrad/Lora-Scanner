export interface Transceiver {
  id: number
  port: string
  freq: number | null // MHz; null until the firmware's boot banner has been seen
}

export interface GatewayStatus {
  state: 'online' | 'offline' | 'unknown'
  error?: string
  lastSuccess: string | null
}

export interface BeaconState {
  running: boolean
  text: string
  intervalMs: number
}

export type FrameKind = 'lora' | 'lorawan' | 'raw'

export interface Frame {
  id: number
  transceiver: number // 0 for gateway frames
  time: string // ISO
  kind: FrameKind
  source: string
  identifier: string
  freq: number
  rssi: number | null
  info: string
  raw: string
}

export interface Device {
  identifier: string
  freq: number
  rssi: number | null
  firstSeen: string
  lastSeen: string
  frames: number
}


async function request<T>(method: string, url: string, body?: unknown): Promise<T> {
  const res = await fetch(url, {
    method,
    headers: body === undefined ? undefined : { 'Content-Type': 'application/json' },
    body: body === undefined ? undefined : JSON.stringify(body),
  })
  if (res.status === 204) return undefined as T
  const data = await res.json().catch(() => ({}))
  if (!res.ok) throw new Error(data.error ?? `${res.status} ${res.statusText}`)
  return data as T
}

export const api = {
  ports: () => request<{ ports: string[] }>('GET', '/api/ports').then((r) => r.ports),
  transceivers: () => request<Transceiver[]>('GET', '/api/transceivers'),
  addTransceiver: (port: string) => request<Transceiver>('POST', '/api/transceivers', { port }),
  removeTransceiver: (id: number) => request<void>('DELETE', `/api/transceivers/${id}`),
  transmit: (id: number, text: string) => request<unknown>('POST', `/api/transceivers/${id}/transmit`, { text }),
  beacon: (id: number) => request<BeaconState>('GET', `/api/transceivers/${id}/beacon`),
  startBeacon: (id: number, text: string, intervalMs: number) =>
    request<BeaconState>('POST', `/api/transceivers/${id}/beacon`, { text, intervalMs }),
  stopBeacon: (id: number) => request<BeaconState>('DELETE', `/api/transceivers/${id}/beacon`),
  gateways: () => request<string[]>('GET', '/api/gateways'),
  addGateway: (ip: string) => request<{ ip: string }>('POST', '/api/gateways', { ip }),
  removeGateway: (ip: string) => request<void>('DELETE', `/api/gateways/${encodeURIComponent(ip)}`),
  pollInterval: () => request<{ seconds: number }>('GET', '/api/gateways/poll').then((r) => r.seconds),
  setPollInterval: (seconds: number) =>
    request<{ seconds: number }>('PUT', '/api/gateways/poll', { seconds }).then((r) => r.seconds),
  frames: (after = 0) => request<Frame[]>('GET', `/api/frames?after=${after}`),
  clearFrames: () => request<void>('DELETE', '/api/frames'),
  devices: () => request<Device[]>('GET', '/api/devices'),
}

export const freqLabel = (f: number | null) => (f === null ? 'Detecting…' : `${Number(f.toFixed(3))} MHz`)

export const errMsg = (e: unknown) => (e instanceof Error ? e.message : String(e))
