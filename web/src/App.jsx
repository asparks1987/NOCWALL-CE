import { useEffect, useState } from 'react'

const apiBase = import.meta.env.VITE_API_BASE || '/api'

function App() {
  const [devices, setDevices] = useState([])
  const [incidents, setIncidents] = useState([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState(null)

  const load = async () => {
    setLoading(true)
    setError(null)
    try {
      const [d, i] = await Promise.all([
        fetch(`${apiBase}/devices`).then(r => r.json()),
        fetch(`${apiBase}/incidents`).then(r => r.json()),
      ])
      setDevices(d.devices || [])
      setIncidents(i || [])
    } catch (e) {
      setError(e.message)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    load()
  }, [])

  return (
    <div className="app">
      <header>
        <h1>UISP NOC SPA</h1>
        <p>React + Vite preview hitting the Go API.</p>
        <button onClick={load} disabled={loading}>{loading ? 'Loading...' : 'Reload'}</button>
        {error && <div className="error">Error: {error}</div>}
      </header>
      <section>
        <h2>Devices</h2>
        <ul>
          {devices.map(d => (
            <li key={d.id}>
              <strong>{d.name}</strong> ({d.role}) — {d.online ? 'online' : 'offline'} {d.latency_ms ? `(${d.latency_ms} ms)` : ''}
            </li>
          ))}
        </ul>
      </section>
      <section>
        <h2>Incidents</h2>
        <ul>
          {incidents.map(inc => (
            <li key={inc.id}>
              <strong>{inc.type}</strong> on {inc.device_id} — {inc.severity} (started {inc.started_at})
            </li>
          ))}
        </ul>
      </section>
    </div>
  )
}

export default App
