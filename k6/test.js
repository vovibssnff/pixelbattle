import { randomString, randomIntBetween } from 'https://jslib.k6.io/k6-utils/1.2.0/index.js';
import ws from 'k6/ws';
import { check, sleep } from 'k6';

const sessionDuration = randomIntBetween(10000, 1800000); // user session between 10s and 1m

export const options = {
  vus: 100,
  iterations: 100,
  duration: '30m'
};

export default function () {
  const url = "ws://localhost:8080/ws";
//   const params = { tags: { my_tag: 'my ws session' } };
    const params = {};

  const res = ws.connect(url, params, function (socket) {
    socket.on('open', function open() {
        console.log(`VU ${__VU}: connected`);

      socket.setInterval(function timeout() {
        socket.send(JSON.stringify(
            {
                x: Math.ceil(5),
                y: Math.ceil(255),
                color: [0, 0, 0],
            }
          ));
      }, randomIntBetween(2000, 4000));
    });

    socket.on('ping', function () {
      console.log('PING!');
    });

    socket.on('pong', function () {
      console.log('PONG!');
    });

    socket.on('close', function () {
      console.log(`ws disconnected`);
    });

    socket.on('message', function (message) {
      const msg = JSON.parse(message);
      console.log("got msg: ", msg);
    });

    socket.setTimeout(function () {
      console.log(`VU ${__VU}: ${sessionDuration}ms passed, leaving the chat`);
      socket.send(JSON.stringify({ event: 'LEAVE' }));
    }, sessionDuration);

    socket.setTimeout(function () {
      console.log(`Closing the socket forcefully 3s after graceful LEAVE`);
      socket.close();
    }, sessionDuration + 3000);
  });

  check(res, { 'Connected successfully': (r) => r && r.status === 101 });
}
