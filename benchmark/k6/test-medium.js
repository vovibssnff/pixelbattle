import { randomIntBetween } from 'https://jslib.k6.io/k6-utils/1.2.0/index.js';
import ws from 'k6/ws';
import { check } from 'k6';
import { vu } from 'k6/execution';

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

// Medium load configuration
export const options = {
  stages: [
    { duration: '1m', target: 50 },   // Ramp up to 50 VUs
    { duration: '3m', target: 200 },   // Ramp up to 200 VUs
    { duration: '1m', target: 0 },   // Ramp down
  ],
  thresholds: {
    'ws_connecting': ['p(95)<5000'],
    'ws_session_duration': ['p(95)<5000'],
  },
};

const CANVAS_WIDTH = 500;
const CANVAS_HEIGHT = 250;

export default function () {
  const url = benchmarkWsURL(benchmarkHostPort());
  const params = { tags: { test_type: 'medium_load' } };

  const res = ws.connect(url, params, function (socket) {
    socket.on('open', function open() {
      console.log(`VU ${__VU}: connected`);

      // Place pixels at ~20 messages per minute (every 3 seconds)
      const pixelInterval = randomIntBetween(1250, 2000);
      let pixelCount = 0;
      const maxPixels = randomIntBetween(10, 30); // 10-30 pixels per session

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
        };

        socket.send(JSON.stringify(pixel));
        pixelCount++;
      }, pixelInterval);
    });

    socket.on('ping', function () {
      socket.pong();
    });

    socket.on('pong', function () {
      // Handle pong
    });

    socket.on('message', function (message) {
      try {
        const msg = JSON.parse(message);
        // Validate message structure if needed
      } catch (e) {
        console.error(`VU ${__VU}: Failed to parse message: ${e}`);
      }
    });

    socket.on('close', function () {
      console.log(`VU ${__VU}: disconnected`);
    });

    socket.on('error', function (e) {
      console.error(`VU ${__VU}: WebSocket error: ${e}`);
    });

    // Close connection after session duration (15s to 1m)
    const sessionDuration = randomIntBetween(15000, 60000);
    socket.setTimeout(function () {
      console.log(`VU ${__VU}: Session timeout after ${sessionDuration}ms`);
      socket.close();
    }, sessionDuration);
  });

  check(res, {
    'Connected successfully': (r) => r && r.status === 101,
    'Session duration OK': (r) => r && r.timings && r.timings.duration < 60000,
  });
}
