export const rssiLabel = (r: number | null | undefined) => (r === null || r === undefined || r === 0 ? 'unknown' : `${r} dBm`)

const pad = (n: number, w = 2) => String(n).padStart(w, '0')

/** HH:MM:SS.mmm, like Wireshark's time column. */
export function clockTime(iso: string) {
  const d = new Date(iso)
  return `${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}.${pad(d.getMilliseconds(), 3)}`
}

export const dateTime = (iso: string) => new Date(iso).toLocaleString()
