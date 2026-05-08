// Phase 1 §7.0 stress stage:
//   wall-clock 7 min, sustained 500 VUs (heavy tier).
//   SLO target: P95 e2e_pixel_latency_seconds <= 800 ms, error rate < 1 %.
// Reads into the comparable-KPI report alongside the nominal stage.
import { randomIntBetween } from 'https://jslib.k6.io/k6-utils/1.2.0/index.js';
import ws from 'k6/ws';
import { check } from 'k6';
import { Trend } from 'k6/metrics';
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
    { duration: '60s', target: 250 },   // ramp toward 500 VUs in 1 min
    { duration: '60s', target: 500 },   // reach 500 VUs over the next minute
    { duration: '300s', target: 500 },  // sustain 500 VUs for 5 min
    { duration: '60s', target: 0 },     // ramp down (1 min)
  ],
  // §7.0 thresholds: stress stage SLO. P99 abort prevents runaway runs from blowing the
  // 30-min budget; healthy runs complete cleanly with non-zero variance recorded.
  thresholds: {
    ws_connecting: ['p(95)<5000'],
    e2e_pixel_latency_seconds: [
      'p(95)<0.8',
      { threshold: 'p(99)<5', abortOnFail: true, delayAbortEval: '30s' },
    ],
    'checks{stage:stress}': ['rate>0.99'],
  },
  tags: { stage: 'stress' },
};

const CANVAS_WIDTH = 500;
const CANVAS_HEIGHT = 250;

export default function () {
  const url = benchmarkWsURL(benchmarkHostPort());
  const params = { tags: { test_type: 'heavy_load', stage: 'stress' } };

  const res = ws.connect(url, params, function (socket) {
    socket.on('open', function open() {
      // Place pixels at ~30 messages per minute under stress (every ~2 seconds).
      const pixelInterval = randomIntBetween(750, 1500);
      let pixelCount = 0;
      const maxPixels = randomIntBetween(20, 50);

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

    const sessionDuration = randomIntBetween(30000, 90000);
    socket.setTimeout(function () {
      socket.close();
    }, sessionDuration);
  });

  check(res, {
    'Connected successfully': (r) => r && r.status === 101,
    'Session duration OK': (r) => r && r.timings && r.timings.duration < 120000,
  }, { stage: 'stress' });
}
