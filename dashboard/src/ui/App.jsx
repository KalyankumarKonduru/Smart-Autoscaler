import React, { useEffect, useState } from 'react'

const endpoint = import.meta.env.VITE_AUTOSCALER_URL || 'http://localhost:8080'

export default function App() {
  const [events, setEvents] = useState([])
  const [rps, setRps] = useState(200)
  const [prediction, setPrediction] = useState(null)

  useEffect(() => {
    const t = setInterval(async () => {
      try {
        const res = await fetch(`${endpoint}/events`)
        const data = await res.json()
        setEvents(data.slice(-50).reverse())
      } catch (e) {}
    }, 3000)
    return () => clearInterval(t)
  }, [])

  const predict = async () => {
    const res = await fetch(`${endpoint}/predict`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ rps: Number(rps) })
    })
    setPrediction(await res.json())
  }

  return (
    <div style={{ fontFamily: 'ui-sans-serif, system-ui', padding: 24 }}>
      <h1 style={{ marginBottom: 8 }}>Smart Autoscaler</h1>
      <p style={{ marginTop: 0, color: '#666' }}>Live scaling events and What‑If simulator</p>

      <section style={{ marginTop: 24, display: 'grid', gap: 16, gridTemplateColumns: '1fr 1fr' }}>
        <div style={{ padding: 16, border: '1px solid #ddd', borderRadius: 12 }}>
          <h3>What‑If Mode</h3>
          <div>
            <input type="range" min="10" max="1000" value={rps} onChange={e => setRps(e.target.value)} />
            <div>RPS: <b>{rps}</b></div>
            <button onClick={predict}>Predict decision</button>
          </div>
          {prediction && (
            <pre style={{ background:'#fafafa', padding:12, borderRadius:8, overflowX:'auto' }}>
{JSON.stringify(prediction,null,2)}
            </pre>
          )}
        </div>
        <div style={{ padding: 16, border: '1px solid #ddd', borderRadius: 12 }}>
          <h3>Recent Scaling Events</h3>
          <table width="100%" cellPadding="6">
            <thead><tr><th>Time</th><th>Action</th><th>From→To</th><th>Reason</th></tr></thead>
            <tbody>
              {events.map((e, i) => (
                <tr key={i}>
                  <td>{new Date(e.time).toLocaleTimeString()}</td>
                  <td>{e.action}</td>
                  <td>{e.from}→{e.to}</td>
                  <td>{e.reason}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </section>
    </div>
  )
}
