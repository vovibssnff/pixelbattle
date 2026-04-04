import { randomString, randomIntBetween } from 'https://jslib.k6.io/k6-utils/1.2.0/index.js';
import ws from 'k6/ws';
import { check, sleep } from 'k6';

// Heavy load configuration
export const options = {
  stages: [
    { duration: '3m', target: 100 },  // Ramp up to 100 VUs
    { duration: '9m', target: 500 },  // Ramp up to 500 VUs
    { duration: '3m', target: 0 },   // Ramp down
  ],
  thresholds: {
    'ws_connecting': ['rate<0.05'],      // Less than 5% connection failures
    'ws_session_duration': ['p(95)<15000'], // 95% of sessions complete in under 15s
  },
};

const CANVAS_WIDTH = 500;
const CANVAS_HEIGHT = 250;

export default function () {
  const url = "ws://localhost:8080/ws";
  const params = { tags: { test_type: 'heavy_load' } };

  const res = ws.connect(url, params, function (socket) {
    socket.on('open', function open() {
      console.log(`VU ${__VU}: connected`);

      // Place pixels at ~30 messages per minute (every 2 seconds)
      const pixelInterval = randomIntBetween(1500, 3000);
      let pixelCount = 0;
      const maxPixels = randomIntBetween(20, 50); // 20-50 pixels per session

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

    // Close connection after session duration (1m to 3m)
    const sessionDuration = randomIntBetween(60000, 180000);
    socket.setTimeout(function () {
      console.log(`VU ${__VU}: Session timeout after ${sessionDuration}ms`);
      socket.close();
    }, sessionDuration);
  });

  check(res, { 
    'Connected successfully': (r) => r && r.status === 101,
    'Session duration OK': (r) => r && r.timings.duration < 180000,
  });
}
