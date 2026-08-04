import http from 'http';
import fs from 'fs';
import path from 'path';
import crypto from 'crypto';
import { fileURLToPath } from 'url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

const PORT = process.env.PORT || 8080;
const INDEX_PATH = path.resolve(__dirname, '../web/index.html');

const mockUser = {
  id: 'dev-alice-123',
  email: 'alice@flame.edu.in',
  display_name: 'Alice',
  avatar_url: ''
};

const mockRatings = {
  blitz: { rating: 1200, games_played: 15 },
  bullet: { rating: 1150, games_played: 10 },
  rapid: { rating: 1250, games_played: 5 }
};

const server = http.createServer((req, res) => {
  const url = new URL(req.url, `http://${req.headers.host || 'localhost:8080'}`);

  // CORS headers
  res.setHeader('Access-Control-Allow-Origin', '*');
  res.setHeader('Access-Control-Allow-Credentials', 'true');
  res.setHeader('Access-Control-Allow-Methods', 'GET, POST, PATCH, PUT, DELETE, OPTIONS');
  res.setHeader('Access-Control-Allow-Headers', 'Content-Type, Authorization, Cookie');

  if (req.method === 'OPTIONS') {
    res.writeHead(204);
    res.end();
    return;
  }

  // GET / -> Serve web/index.html
  if (req.method === 'GET' && (url.pathname === '/' || url.pathname === '/index.html')) {
    fs.readFile(INDEX_PATH, 'utf-8', (err, content) => {
      if (err) {
        res.writeHead(500, { 'Content-Type': 'text/plain' });
        res.end('Error loading index.html: ' + err.message);
        return;
      }
      res.writeHead(200, { 'Content-Type': 'text/html; charset=utf-8' });
      res.end(content);
    });
    return;
  }

  // GET /healthz
  if (req.method === 'GET' && url.pathname === '/healthz') {
    res.writeHead(200, { 'Content-Type': 'application/json' });
    res.end(JSON.stringify({ status: 'ok' }));
    return;
  }

  // POST /auth/dev-login
  if (req.method === 'POST' && url.pathname === '/auth/dev-login') {
    let body = '';
    req.on('data', chunk => { body += chunk.toString(); });
    req.on('end', () => {
      let name = 'Alice';
      if (body) {
        const params = new URLSearchParams(body);
        if (params.get('name')) name = params.get('name');
        else {
          try {
            const parsed = JSON.parse(body);
            if (parsed.name) name = parsed.name;
          } catch (_) {}
        }
      }
      mockUser.display_name = name;
      mockUser.id = `dev-${name.toLowerCase()}-123`;
      mockUser.email = `${name.toLowerCase()}@flame.edu.in`;

      res.writeHead(200, {
        'Content-Type': 'application/json',
        'Set-Cookie': `fc_session=mock-token-${name}; Path=/; HttpOnly`
      });
      res.end(JSON.stringify({ identity: mockUser }));
    });
    return;
  }

  // POST /auth/logout
  if (req.method === 'POST' && url.pathname === '/auth/logout') {
    res.writeHead(204, {
      'Set-Cookie': 'fc_session=; Path=/; Max-Age=0'
    });
    res.end();
    return;
  }

  // GET /api/me
  if (req.method === 'GET' && url.pathname === '/api/me') {
    const cookies = req.headers.cookie || '';
    if (cookies.includes('fc_session') || true) {
      res.writeHead(200, { 'Content-Type': 'application/json' });
      res.end(JSON.stringify({ user: mockUser, ratings: mockRatings }));
    } else {
      res.writeHead(401, { 'Content-Type': 'application/json' });
      res.end(JSON.stringify({ error: 'unauthorized' }));
    }
    return;
  }

  // PATCH /api/me
  if (req.method === 'PATCH' && url.pathname === '/api/me') {
    let body = '';
    req.on('data', chunk => { body += chunk.toString(); });
    req.on('end', () => {
      try {
        const parsed = JSON.parse(body);
        if (parsed.display_name) mockUser.display_name = parsed.display_name;
      } catch (_) {}
      res.writeHead(200, { 'Content-Type': 'application/json' });
      res.end(JSON.stringify({ user: mockUser, ratings: mockRatings }));
    });
    return;
  }

  // GET /api/leaderboard
  if (req.method === 'GET' && url.pathname === '/api/leaderboard') {
    res.writeHead(200, { 'Content-Type': 'application/json' });
    res.end(JSON.stringify([
      { display_name: 'GrandmasterFlame', rating: 2450, games_played: 320 },
      { display_name: mockUser.display_name, rating: 1200, games_played: 15 },
      { display_name: 'Bob', rating: 1180, games_played: 12 },
      { display_name: 'Charlie', rating: 1050, games_played: 8 }
    ]));
    return;
  }

  // GET /api/games
  if (req.method === 'GET' && url.pathname === '/api/games') {
    res.writeHead(200, { 'Content-Type': 'application/json' });
    res.end(JSON.stringify([
      {
        id: 'g-101',
        white_id: mockUser.id,
        black_id: 'dev-bob-456',
        category: 'blitz',
        status: 'finished',
        result: '1-0',
        result_reason: 'checkmate',
        white_before: 1180,
        white_after: 1200,
        black_before: 1200,
        black_after: 1180,
        started_at: '2026-08-03T18:00:00Z',
        ended_at: '2026-08-03T18:05:00Z'
      }
    ]));
    return;
  }

  // GET /api/games/live
  if (req.method === 'GET' && url.pathname === '/api/games/live') {
    res.writeHead(200, { 'Content-Type': 'application/json' });
    res.end(JSON.stringify([
      { id: 'g-live-1', white_name: 'Alice', black_name: 'Bob', category: 'blitz', white_rating: 1200, black_rating: 1180 }
    ]));
    return;
  }

  // POST /api/challenges
  if (req.method === 'POST' && url.pathname === '/api/challenges') {
    res.writeHead(200, { 'Content-Type': 'application/json' });
    res.end(JSON.stringify({ id: 'chal-888', url: `http://localhost:${PORT}/?challenge=chal-888` }));
    return;
  }

  // Default static or 404
  res.writeHead(404, { 'Content-Type': 'application/json' });
  res.end(JSON.stringify({ error: 'not_found' }));
});

// Upgrade handler for WebSocket connection mock
server.on('upgrade', (req, socket, head) => {
  if (req.url === '/ws') {
    // Send standard WS handshake response
    const key = req.headers['sec-websocket-key'];
    const digest = crypto.createHash('sha1')
      .update(key + '258EAFA5-E914-47DA-95CA-C5AB0DC85B11')
      .digest('base64');
    
    socket.write(
      'HTTP/1.1 101 Switching Protocols\r\n' +
      'Upgrade: websocket\r\n' +
      'Connection: Upgrade\r\n' +
      `Sec-WebSocket-Accept: ${digest}\r\n\r\n`
    );
  } else {
    socket.destroy();
  }
});

export function startServer(port = PORT) {
  return new Promise((resolve) => {
    server.listen(port, () => {
      console.log(`[DevServer] Listening on http://localhost:${port}`);
      resolve(server);
    });
  });
}

if (process.argv[1] === fileURLToPath(import.meta.url)) {
  startServer();
}
