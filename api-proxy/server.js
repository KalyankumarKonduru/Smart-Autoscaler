import { request } from "undici";
import http from "http";

const TARGET = process.env.TARGET; // e.g. http://34.173.97.171
if (!TARGET) throw new Error("TARGET env var missing");

const server = http.createServer(async (req, res) => {
  // --- Basic CORS ---
  res.setHeader("Access-Control-Allow-Origin", "*");
  res.setHeader("Access-Control-Allow-Methods", "GET,POST,OPTIONS");
  res.setHeader("Access-Control-Allow-Headers", "Content-Type,Origin");
  if (req.method === "OPTIONS") {
    res.writeHead(204);
    return res.end();
  }

  // --- Health check handler ---
  if (req.url === "/healthz") {
    res.writeHead(200, { "content-type": "text/plain" });
    return res.end("ok");
  }

  try {
    const targetUrl = new URL(req.url, TARGET); // build correct target
    const chunks = [];
    for await (const chunk of req) chunks.push(chunk);
    const body = chunks.length ? Buffer.concat(chunks) : null;

    // Forward the request to backend
    const upstream = await request(targetUrl.toString(), {
      method: req.method,
      headers: { "content-type": req.headers["content-type"] || "" },
      body,
    });

    // Copy upstream headers safely (avoid duplicate 'transfer-encoding')
    const headers = Object.fromEntries(
      Object.entries(upstream.headers).filter(([k]) => k.toLowerCase() !== "transfer-encoding")
    );
    res.writeHead(upstream.statusCode, headers);

    // Pipe upstream response body directly
    if (upstream.body && Symbol.asyncIterator in upstream.body) {
        for await (const c of upstream.body) res.write(c);
      }
      res.end();
      
  } catch (err) {
    console.error("Proxy error:", err.message);
    if (!res.headersSent) {
      res.writeHead(502, { "content-type": "application/json" });
    }
    res.end(JSON.stringify({ error: err.message }));
  }
});

const port = process.env.PORT || 8080;
server.listen(port, () => console.log(`Proxy listening on ${port} → ${TARGET}`));
