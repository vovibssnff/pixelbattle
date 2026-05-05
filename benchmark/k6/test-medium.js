// Phase 1 §7.0 nominal stage:
//   wall-clock 4 min, ramp 50 -> 200 VUs (matches real event peak ~67 msg/s).
//   SLO target: P95 e2e_pixel_latency_seconds <= 200 ms, error rate < 0.1 %.
// Comparable KPIs in plan §2 are read from this stage's summary export.
//
// The custom Trend `e2e_pixel_latency_seconds` records (Date.now() - client_sent_ms) / 1000
// for every broadcast envelope the VU receives (placer-or-observer; the backend echoes
// `client_sent_ms` and `server_recv_ms` on every fan-out so the latency is computable
// without server-side correlation).
import { randomIntBetween } from 'https://jslib.k6.io/k6-utils/1.2.0/index.js';
import ws from 'k6/ws';
import { check, Trend } from 'k6';
import { vu } from 'k6/execution';

const e2ePixelLatencySeconds = new Trend('e2e_pixel_latency_seconds', true);
const FACULTIES = ['KTU', 'TINT', 'FTMF', 'FTMI', 'NOZH'];

function benchmarkHostPort() {
  const h = __ENV.BENCHMARK_HOST;
  return h && String(h).length > 0 ? String(h) : 'localhost:8080';
}

function benchmarkWsURL(hostPort) {
  const hp = hostPort || benchmarkHostPort();
  const uid = vu.idInTest * 100000000 + vu.iterationInTest;
  const faculty = FACULTIES[randomIntBetween(0, FACULTIES.length - 1)];
  return `ws://${hp}/ws?uid=${uid}&faculty=${encodeURIComponent(faculty)}`;
}

export const options = {
  stages: [
    { duration: '60s', target: 50 },    // warm up to 50 VUs
    { duration: '120s', target: 200 },  // ramp to 200 VUs (peak event traffic)
    { duration: '60s', target: 200 },   // hold at 200 for the comparable window
  ],
  // §7.0 thresholds: nominal stage SLO. abortOnFail trips only on a P99 above 5 s, so
  // healthy runs complete cleanly while pathological runs exit with rc=99.
  thresholds: {
    ws_connecting: ['p(95)<5000'],
    e2e_pixel_latency_seconds: [
      'p(95)<0.2',
      { threshold: 'p(99)<5', abortOnFail: true, delayAbortEval: '30s' },
    ],
    'checks{stage:nominal}': ['rate>0.999'],
  },
  tags: { stage: 'nominal' },
};

const CANVAS_WIDTH = 500;
const CANVAS_HEIGHT = 250;

export default function () {
  const url = benchmarkWsURL(benchmarkHostPort());
  const params = { tags: { test_type: 'medium_load', stage: 'nominal' } };

  const res = ws.connect(url, params, function (socket) {
    socket.on('open', function open() {
      // Place pixels at ~20 messages per minute (every 3 seconds) to match nominal traffic.
      const pixelInterval = randomIntBetween(1250, 2000);
      let pixelCount = 0;
      const maxPixels = randomIntBetween(10, 30);

      socket.setInterval(function timeout() {
        if (pixelCount >= maxPixels) {
          socket.close();
          return;
        }

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
        pixelCount++;
      }, pixelInterval);
    });

    socket.on('ping', function () {
      socket.pong();
    });

    socket.on('message', function (message) {
      try {
        const payload = JSON.parse(message);
        const clientSentMs = payload.client_sent_ms ?? payload.clientSentMs;
        if (clientSentMs) {
          const delta = (Date.now() - Number(clientSentMs)) / 1000;
          if (delta >= 0) e2ePixelLatencySeconds.add(delta);
        }
      } catch (_) {
        // non-JSON message; ignored
      }
    });

    socket.on('error', function (e) {
      console.error(`VU ${__VU}: WebSocket error: ${e}`);
    });

    const sessionDuration = randomIntBetween(15000, 60000);
    socket.setTimeout(function () {
      socket.close();
    }, sessionDuration);
  });

  check(res, {
    'Connected successfully': (r) => r && r.status === 101,
    'Session duration OK': (r) => r && r.timings && r.timings.duration < 90000,
  }, { stage: 'nominal' });
}
