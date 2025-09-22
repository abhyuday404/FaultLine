import express from 'express';
import cors from 'cors';
import net from 'node:net';

const app = express();
const PORT = process.env.SHOWCASE_TCP_SERVER_PORT || 5175;

app.use(cors());
app.use(express.json());

app.get('/health', (_req, res) => {
  res.json({ ok: true });
});

app.post('/tcp/test', async (req, res) => {
  const { host = '127.0.0.1', port = 15432, timeoutMs = 3000, payload = '' } = req.body || {};

  const start = Date.now();
  let connected = false;
  let timedOut = false;

  const result = await new Promise((resolve) => {
    const socket = new net.Socket();

    const onDone = (out) => {
      try { socket.destroy(); } catch {}
      resolve(out);
    };

    socket.setTimeout(Number(timeoutMs) || 3000);

    socket.on('connect', () => {
      connected = true;
      if (payload) {
        try { socket.write(payload); } catch {}
      }
      // End quickly; we're only validating the connect path
      try { socket.end(); } catch {}
    });

    socket.on('timeout', () => {
      timedOut = true;
      onDone({
        ok: false,
        host,
        port,
        timeoutMs,
        error: 'timeout',
        elapsedMs: Date.now() - start,
      });
    });

    socket.on('error', (err) => {
      onDone({
        ok: false,
        host,
        port,
        timeoutMs,
        error: err.code || err.message || String(err),
        elapsedMs: Date.now() - start,
      });
    });

    socket.on('close', (hadError) => {
      if (timedOut) return; // already handled
      if (hadError && !connected) return; // already handled in error
      onDone({
        ok: connected,
        host,
        port,
        timeoutMs,
        status: connected ? 'closed-after-connect' : 'closed',
        elapsedMs: Date.now() - start,
      });
    });

    try {
      socket.connect({ host, port: Number(port) });
    } catch (e) {
      onDone({
        ok: false,
        host,
        port,
        timeoutMs,
        error: e.message || String(e),
        elapsedMs: Date.now() - start,
      });
    }
  });

  res.json(result);
});

app.listen(PORT, () => {
  console.log(`[showcase-tcp] listening on http://localhost:${PORT}`);
});
