<template>
  <div>
    <canvas id="viewport-canvas" oncontextmenu="return false;"></canvas>
    <div id="ui-wrapper" hide="true">
      <div id="color-wrapper">
        <div id="color-swatch"></div>
        <input id="color-field" type="text" placeholder="#000000" value="#000000" />
      </div>
      <div id="zoom-wrapper">
        <button class="zoom-button" id="zoom-out">
          -
        </button>
        <button class="zoom-button" id="zoom-in">
          +
        </button>
      </div>
    </div>
  </div>
</template>

<script>
import GLWindow from '@/webgl/glwindow.js'
import Place from '@/webgl/place.js'
import { matchedRouteKey } from 'vue-router';

export default {
  data() {
    return {
      ws: null,
      connected: false,
      colorField: null,
      colorSwatch: null,
    }
  },
  created() {
    this.setViewport();
    // this.initializeSocket();
  },
  mounted() {
    this.initializeGUI();
    this.connectToWebSocket();
  },
  methods: {
    sendPixel(x, y, color) {
      const pixel = {
        x: Math.ceil(x),
        y: Math.ceil(y),
        color: [color[0], color[1], color[2]],
      };
      console.log(JSON.stringify(pixel));
      this.ws.send(JSON.stringify(pixel));
    },
    setViewport() {
      const meta = document.createElement('meta');
      meta.setAttribute('name', 'viewport');
      meta.setAttribute('content', 'width=device-width, initial-scale=1.0, maximum-scale=1.0, user-scalable=no');
      document.head.appendChild(meta);
    },
    connectToWebSocket() {
      this.ws = new WebSocket("ws://" + window.location.host + "/ws");
      this.ws.addEventListener('open', (event) => {this.onWebSocketOpen(event)});
      this.ws.addEventListener('message', (event) => {this.handleNewPixel(event)});
    },
    onWebSocketOpen() {
      console.log("damnit websocket connected");
    },
    handleNewPixel(event) {
      const pixel = JSON.parse(event.data);
      console.log(pixel);
    },
    initializeGUI() {
      const cvs = document.querySelector("#viewport-canvas");

      const glWindow = new GLWindow(cvs);
      const place = new Place(glWindow);
      
      place.initConnection("/init_canvas");

      let color = new Uint8Array([0, 0, 0]);
      let dragdown = false;
      let touchID = 0;
      let touchScaling = false;
      let lastMovePos = {x: 0, y: 0};
      let lastScalingDist = 0;

      this.colorField = document.querySelector("#color-field");
      this.colorSwatch = document.querySelector("#color-swatch");

      document.addEventListener("keydown", ev => {
        switch (ev.keyCode) {
          case 189:
          case 173:
            ev.preventDefault();
            zoomOut(1.2);
            break;
          case 187:
          case 61:
            ev.preventDefault();
            zoomIn(1.2);
            break;
        }
      });

      window.addEventListener("wheel", ev => {
        let zoom = glWindow.getZoom();
        if (ev.deltaY > 0) {
          zoom /= 1.05;
        } else {
          zoom *= 1.05;
        }
        glWindow.setZoom(zoom);
        glWindow.draw();
      });

      document.querySelector("#zoom-in").addEventListener("click", () => {
        zoomIn(1.2);
      });
      
      document.querySelector("#zoom-out").addEventListener("click", () => {
        zoomOut(1.2);
      });

      window.addEventListener("resize", ev => {
        glWindow.updateViewScale();
        glWindow.draw();
      });

      cvs.addEventListener("mousedown", (ev) => {
        switch (ev.button) {
          case 0:
            dragdown = true;
            lastMovePos = { x: ev.clientX, y: ev.clientY };
            break;
          case 1:
            pickColor({ x: ev.clientX, y: ev.clientY });
            break;
          case 2:
            if (ev.ctrlKey) {
              pickColor({ x: ev.clientX, y: ev.clientY });
            } else {
              ev.stopPropagation();
              drawPixel({ x: ev.clientX, y: ev.clientY }, color);
            }
        }
      });

      document.addEventListener("mouseup", (ev) => {
        dragdown = false;
        document.body.style.cursor = "auto";
      });

      document.addEventListener("mousemove", (ev) => {
        const movePos = { x: ev.clientX, y: ev.clientY };
        if (dragdown) {
          glWindow.move(movePos.x - lastMovePos.x, movePos.y - lastMovePos.y);
          glWindow.draw();
          document.body.style.cursor = "grab";
        }
        lastMovePos = movePos;
      });

      cvs.addEventListener("touchstart", (ev) => {
        let thisTouch = touchID;
        this.touchstartTime = (new Date()).getTime();
        lastMovePos = { x: ev.touches[0].clientX, y: ev.touches[0].clientY };
        if (ev.touches.length === 2) {
          touchScaling = true;
          lastScalingDist = null;
        }

        setTimeout(() => {
          if (thisTouch == touchID) {
            pickColor(lastMovePos);
            navigator.vibrate(200);
          }
        }, 350);
      });

      document.addEventListener("touchend", (ev) => {
        touchID++;
        let elapsed = (new Date()).getTime() - this.touchstartTime;
        if (elapsed < 100) {
          if (drawPixel(lastMovePos, color)) {
            navigator.vibrate(10);
          };
        }
        if (ev.touches.length === 0) {
          touchScaling = false;
        }
      });

      document.addEventListener("touchmove", (ev) => {
        touchID++;
        if (touchScaling) {
          let dist = Math.hypot(
              ev.touches[0].pageX - ev.touches[1].pageX,
              ev.touches[0].pageY - ev.touches[1].pageY);
          if (lastScalingDist != null) {
            let delta = lastScalingDist - dist;
            if (delta < 0) {
              zoomIn(1 + Math.abs(delta) * 0.003);
            } else {
              zoomOut(1 + Math.abs(delta) * 0.003);
            }
          }
          lastScalingDist = dist;
        } else {
          let movePos = { x: ev.touches[0].clientX, y: ev.touches[0].clientY };
          glWindow.move(movePos.x - lastMovePos.x, movePos.y - lastMovePos.y);
          glWindow.draw();
          lastMovePos = movePos;
        }
      });

      cvs.addEventListener("contextmenu", () => { return false; });

      this.colorField.addEventListener("change", ev => {
        let hex = this.colorField.value.replace(/[^A-Fa-f0-9]/g, "").toUpperCase();
        hex = hex.substring(0, 6);
        while (hex.length < 6) {
          hex += "0";
        }
        color[0] = parseInt(hex.substring(0, 2), 16);
        color[1] = parseInt(hex.substring(2, 4), 16);
        color[2] = parseInt(hex.substring(4, 6), 16);
        hex = "#" + hex;
        this.colorField.value = hex;
        this.colorSwatch.style.backgroundColor = hex;
      });

      const pickColor = (pos) => {
        color = glWindow.getColor(glWindow.click(pos));
        let hex = "#";
        for (let i = 0; i < color.length; i++) {
          let d = color[i].toString(16);
          if (d.length == 1) d = "0" + d;
          hex += d;
        }
        this.colorField.value = hex.toUpperCase();
        this.colorSwatch.style.backgroundColor = hex;
      }

      const drawPixel = (pos, color) => {
        pos = glWindow.click(pos);
        if (pos) {
          const oldColor = glWindow.getColor(pos);
          for (let i = 0; i < oldColor.length; i++) {
            if (oldColor[i] != color[i]) {
              //TODO чек таймера
              this.sendPixel(pos.x, pos.y, color);
              place.setPixel(pos.x, pos.y, color);
              return true;
            }
          }
        }
        return false;
      }

      const zoomIn = (factor) => {
        let zoom = glWindow.getZoom();
        glWindow.setZoom(zoom * factor);
        glWindow.draw();
      }

      const zoomOut = (factor) => {
        let zoom = glWindow.getZoom();
        glWindow.setZoom(zoom / factor);
        glWindow.draw();
      }
    }
  }
}
</script>

<style scoped>
* {
  padding: 0;
  margin: 0;
  font-family: monospace;
}

body {
  overflow: hidden;
  position: fixed;
  -webkit-overflow-scrolling: touch;
}

#viewport-canvas {
  position: absolute;
  top: 0;
  left: 0;
  image-rendering: pixelated;
  width: 100vw;
  height: 100vh;
  background-color: #e0e0e0;
}

#ui-wrapper {
  position: fixed;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  background-color: #ffffff;
  transition: background 1s;
}

#ui-wrapper>#color-wrapper,
#zoom-wrapper {
  visibility: hidden;
}

#ui-wrapper[hide=true] {
  pointer-events: none;
  background: none;
}

#ui-wrapper[hide=true]>#color-wrapper,
#zoom-wrapper {
  visibility: visible;
}

#color-wrapper {
  position: absolute;
  bottom: 16px;
  left: 16px;
  display: flex;
  flex-direction: row;
}

#color-swatch {
  width: 30px;
  height: 30px;
  background-color: #000000;
}

#color-field {
  font-size: 16px;
  height: 30px;
  padding: 1px;
  border: none;
  outline: none;
  pointer-events: all;
}

#zoom-wrapper {
  position: absolute;
  bottom: 16px;
  right: 16px;
}

.zoom-button {
  width: 36px;
  height: 36px;
  border: none;
  background: none;
  outline: none;
  background-color: red;
  font-size: 24px;
  background-color: #ffffff;
  border: 1px solid black;
  cursor: pointer;
  pointer-events: all;
  user-select: none;
}

@media (hover: none) {
  #help-text-desktop {
    display: none;
  }

  #help-text-mobile {
    display: inline;
  }
}
</style>