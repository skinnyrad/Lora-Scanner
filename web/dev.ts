// Dev server: Bun's HTML bundler with HMR, proxying /api (HTTP + WebSocket) to the Go server.
import index from './index.html'

const API = process.env.LORA_API ?? 'localhost:5000'

type Payload = Parameters<WebSocket['send']>[0]
type WsData = { path: string; upstream?: WebSocket; queue: Payload[] }

Bun.serve<WsData>({
  port: 5173,
  routes: {
    '/api/*': (req: Request, server: Bun.Server<WsData>) => {
      const url = new URL(req.url)
      if (req.headers.get('upgrade') === 'websocket') {
        return server.upgrade(req, { data: { path: url.pathname + url.search, queue: [] } })
          ? undefined
          : new Response('upgrade failed', { status: 400 })
      }
      return fetch(`http://${API}${url.pathname}${url.search}`, {
        method: req.method,
        headers: req.headers,
        body: req.body,
      })
    },
    '/*': index,
  },
  development: true,
  websocket: {
    open(ws) {
      const up = new WebSocket(`ws://${API}${ws.data.path}`)
      ws.data.upstream = up
      up.onopen = () => ws.data.queue.splice(0).forEach((m) => up.send(m))
      up.onmessage = (e) => ws.send(e.data)
      up.onclose = () => ws.close()
    },
    message(ws, msg) {
      const up = ws.data.upstream
      const payload = msg as Payload
      if (up?.readyState === WebSocket.OPEN) up.send(payload)
      else ws.data.queue.push(payload)
    },
    close(ws) {
      ws.data.upstream?.close()
    },
  },
})

console.log('dev server on http://localhost:5173 (proxying /api to ' + API + ')')
