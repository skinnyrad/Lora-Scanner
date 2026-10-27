import { createContext, useCallback, useContext, useEffect, useMemo, useRef, useState, type ReactNode } from 'react'
import { api, type Device, type Frame, type GatewayStatus, type Transceiver } from '../api/client'

const MAX_FRAMES = 10000
const FLUSH_MS = 250

interface Live {
  connected: boolean // websocket is up
  transceivers: Transceiver[]
  gatewayStatus: Record<string, GatewayStatus> // keyed by gateway IP; missing means not polled yet
  frames: Frame[]
  devices: Record<string, Device>
  clearFrames: () => Promise<void>
}

const Ctx = createContext<Live | null>(null)

export function LiveProvider({ children }: { children: ReactNode }) {
  const [connected, setConnected] = useState(false)
  const [transceivers, setTransceivers] = useState<Transceiver[]>([])
  const [gatewayStatus, setGatewayStatus] = useState<Record<string, GatewayStatus>>({})
  const [frames, setFrames] = useState<Frame[]>([])
  const [devices, setDevices] = useState<Record<string, Device>>({})

  // Frames and device updates arrive in bursts; buffer them and render on a timer.
  const pendingFrames = useRef<Frame[]>([])
  const pendingDevices = useRef<Record<string, Device>>({})
  const lastId = useRef(0)
  const retry = useRef(1000)

  useEffect(() => {
    const flush = () => {
      if (pendingFrames.current.length) {
        const batch = pendingFrames.current.sort((a, b) => a.id - b.id)
        pendingFrames.current = []
        // ws and the catch-up fetch can both deliver a frame; keep each id once.
        const fresh = batch.filter((f, i) => f.id > lastId.current && (i === 0 || f.id !== batch[i - 1].id))
        if (fresh.length) {
          lastId.current = fresh[fresh.length - 1].id
          setFrames((prev) => {
            const next = prev.concat(fresh)
            return next.length > MAX_FRAMES ? next.slice(next.length - MAX_FRAMES) : next
          })
        }
      }
      const pd = pendingDevices.current
      if (Object.keys(pd).length) {
        pendingDevices.current = {}
        setDevices((prev) => ({ ...prev, ...pd }))
      }
    }
    const timer = window.setInterval(flush, FLUSH_MS)

    let ws: WebSocket | undefined
    let reconnect: number | undefined
    let stopped = false

    const sync = async () => {
      try {
        const [fs, ds] = await Promise.all([api.frames(lastId.current), api.devices()])
        pendingFrames.current.push(...fs)
        setDevices(Object.fromEntries(ds.map((d) => [d.identifier, d])))
      } catch {
        /* the next reconnect retries */
      }
    }

    const open = () => {
      const proto = location.protocol === 'https:' ? 'wss' : 'ws'
      ws = new WebSocket(`${proto}://${location.host}/api/ws`)
      ws.onopen = () => {
        retry.current = 1000
        setConnected(true)
        sync()
      }
      ws.onmessage = (ev) => {
        const m = JSON.parse(ev.data)
        if (m.type === 'transceivers') {
          setTransceivers(m.transceivers)
        } else if (m.type === 'gateways') {
          setGatewayStatus(m.gateways)
        } else if (m.type === 'frame') {
          pendingFrames.current.push(m.frame)
        } else if (m.type === 'device') {
          pendingDevices.current[m.device.identifier] = m.device
        }
      }
      ws.onclose = () => {
        setConnected(false)
        if (stopped) return
        reconnect = window.setTimeout(open, retry.current)
        retry.current = Math.min(retry.current * 2, 10000)
      }
    }
    open()
    return () => {
      stopped = true
      window.clearInterval(timer)
      window.clearTimeout(reconnect)
      ws?.close()
    }
  }, [])

  const clearFrames = useCallback(async () => {
    await api.clearFrames()
    pendingFrames.current = []
    pendingDevices.current = {}
    setFrames([])
    setDevices({})
  }, [])

  const value = useMemo(() => ({ connected, transceivers, gatewayStatus, frames, devices, clearFrames }), [connected, transceivers, gatewayStatus, frames, devices, clearFrames])
  return <Ctx.Provider value={value}>{children}</Ctx.Provider>
}

export function useLive() {
  const v = useContext(Ctx)
  if (!v) throw new Error('useLive outside LiveProvider')
  return v
}
