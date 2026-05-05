import ws from 'k6/ws';
import { check, Trend } from 'k6';
import { randomIntBetween } from 'https://jslib.k6.io/k6-utils/1.2.0/index.js';
import { vu } from 'k6/execution';

const e2eRenderLatency = new Trend('e2e_render_latency', true);
const FACULTIES = ['KTU', 'TINT', 'FTMF', 'FTMI', 'NOZH'];

function benchmarkHostPort() {
  const h = __ENV.BENCHMARK_HOST;
  return h && String(h).length > 0 ? String(h) : 'localhost:8080';
}

function benchmarkWsURL(hostPort) {
  const uid = vu.idInTest * 100000000 + vu.iterationInTest;
  const faculty = FACULTIES[randomIntBetween(0, FACULTIES.length - 1)];
  return `ws://${hostPort}/ws?uid=${uid}&faculty=${encodeURIComponent(faculty)}`;
}

export const options = {
  stages: [
    { duration: '2m', target: 500 },
    { duration: '2m', target: 5000 },
    { duration: '1m', target: 0 },
  ],
  thresholds: {
    ws_connecting: ['p(95)<5000'],
    e2e_render_latency: ['p(95)<1000'],
  },
};

const CANVAS_WIDTH = 500;
const CANVAS_HEIGHT = 250;

export default function () {
  const host = benchmarkHostPort();
  const url = benchmarkWsURL(host);
  const params = { tags: { test_type: 'burst' } };

  const res = ws.connect(url, params, function (socket) {
    socket.on('open', function () {
      const pixel = {
        x: randomIntBetween(0, CANVAS_WIDTH - 1),
        y: randomIntBetween(0, CANVAS_HEIGHT - 1),
        color: [
          randomIntBetween(0, 255),
          randomIntBetween(0, 255),
          randomIntBetween(0, 255),
        ],
        client_sent_ms: Date.now(),
      };
      socket.send(JSON.stringify(pixel));
    });

    socket.on('message', function (message) {
      try {
        const payload = JSON.parse(message);
        const serverRecvMs = payload.server_recv_ms ?? payload.serverRecvMs;
        if (serverRecvMs) {
          const delta = Date.now() - Number(serverRecvMs);
          if (delta >= 0) e2eRenderLatency.add(delta);
        }
      } catch (_) {
        // ignore non JSON messages
      }
    });

    socket.setTimeout(function () {
      socket.close();
    }, randomIntBetween(5000, 12000));
  });

  check(res, {
    'Connected successfully': (r) => r && r.status === 101,
  });
}

