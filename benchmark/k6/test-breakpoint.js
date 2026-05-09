// Phase 1 §7.0 breakpoint stage:
//   wall-clock 10 min, linear ramp 100 -> 3000 VUs.
//   Success criterion: at least one of (error rate > 5 %, P99 > 5 s, WS handshake fail
//   > 1 %). Records `saturation_vus` Trend with the active-VU count at each detected
//   SLO breach so the reporter can take min(saturation_vus) as the saturation point.
//
// Aborts cleanly via abortOnFail on e2e_pixel_latency_seconds p(99) > 5s; rc=99 is
// expected and is NOT treated as failure by the playbook.
import { randomIntBetween } from 'https://jslib.k6.io/k6-utils/1.2.0/index.js';
import ws from 'k6/ws';
import { check } from 'k6';
import { Trend } from 'k6/metrics';
import exec, { vu } from 'k6/execution';

const e2ePixelLatencySeconds = new Trend('e2e_pixel_latency_seconds', true);
const saturationVus = new Trend('saturation_vus', false);
const FACULTIES = ['KTU', 'TINT', 'FTMF', 'FTMI', 'NOZH'];

const SLO_BREACH_LATENCY_MS = 5000;

function benchmarkUseTls() {
  const t = __ENV.BENCHMARK_TLS;
  return t === '1' || String(t).toLowerCase() === 'true';
}

function benchmarkHostPort() {
  const h = __ENV.BENCHMARK_HOST;
  return h && String(h).length > 0 ? String(h) : 'localhost:8080';
}

function benchmarkWsURL(hostPort) {
  const hp = hostPort || benchmarkHostPort();
  const scheme = benchmarkUseTls() ? 'wss' : 'ws';
  const uid = vu.idInTest * 100000000 + vu.iterationInTest;
  const faculty = FACULTIES[randomIntBetween(0, FACULTIES.length - 1)];
  return `${scheme}://${hp}/ws?uid=${uid}&faculty=${encodeURIComponent(faculty)}`;
}

function activeVus() {
  // exec.instance.vusActive landed in k6 0.34; older versions fall back to __VU.
  try {
    return exec.instance.vusActive ?? Number(__VU);
  } catch (_) {
    return Number(__VU);
  }
}

export const options = {
  // Linear ramp 100 -> 3000 over 10 min using ramping-vus.
  scenarios: {
    breakpoint: {
      executor: 'ramping-vus',
      startVUs: 100,
      stages: [
        { duration: '600s', target: 3000 },
      ],
      gracefulRampDown: '30s',
      gracefulStop: '30s',
      tags: { stage: 'breakpoint' },
    },
  },
  thresholds: {
    ws_connecting: ['p(95)<5000'],
    e2e_pixel_latency_seconds: [
      { threshold: 'p(99)<5000', abortOnFail: true, delayAbortEval: '30s' },
    ],
    'checks{stage:breakpoint}': ['rate>0.5'],
  },
  tags: { stage: 'breakpoint' },
};

const CANVAS_WIDTH = 500;
const CANVAS_HEIGHT = 250;

export default function () {
  const url = benchmarkWsURL(benchmarkHostPort());
  const params = { tags: { test_type: 'breakpoint', stage: 'breakpoint' } };

  const res = ws.connect(url, params, function (socket) {
    socket.on('open', function open() {
      // Aggressive cadence under breakpoint - every 0.5..1.5s.
      const pixelInterval = randomIntBetween(500, 1500);
      let pixelCount = 0;
      const maxPixels = randomIntBetween(15, 40);

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


    socket.on('message', function (message) {
      try {
        const payload = JSON.parse(message);
        const clientSentMs = payload.client_sent_ms ?? payload.clientSentMs;
        if (clientSentMs) {
          const delta = Date.now() - Number(clientSentMs);
          if (delta >= 0) {
            e2ePixelLatencySeconds.add(delta);
            if (delta > SLO_BREACH_LATENCY_MS) {
              // Record the active-VU count at the moment of each breach. The reporter
              // consumes min(saturation_vus) as "VU count at first SLO break".
              saturationVus.add(activeVus());
            }
          }
        }
      } catch (_) {
        // non-JSON message; ignored
      }
    });

    socket.on('error', function (e) {
      console.error(`VU ${__VU}: WebSocket error: ${e}`);
    });

    const sessionDuration = randomIntBetween(20000, 60000);
    socket.setTimeout(function () {
      socket.close();
    }, sessionDuration);
  });

  const handshakeOk = check(res, {
    'WS handshake': (r) => r && r.status === 101,
  }, { stage: 'breakpoint' });

  if (!handshakeOk) {
    saturationVus.add(activeVus());
  }
}
