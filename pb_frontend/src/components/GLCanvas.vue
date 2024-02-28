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

export default {
  data() {
    return {
      ws: null,
      connected: false,
      colorField: null,
      colorSwatch: null,
      cvs: null,
      glWindow: null,
      place: null,
      color: null,
      dragdown: null,
      touchID: null,
      touchScaling: null,
      lastMovePos: null,
      lastScalingDist: null,
    }
  },
  created() {
    this.setViewport();
  },
  mounted() {
    let self = this;
    this.initRendering();
    this.initEventListeners();
    this.connectToWebSocket();
    
  },
  methods: {
    setViewport() {
      const meta = document.createElement('meta');
      meta.setAttribute('name', 'viewport');
      meta.setAttribute('content', 'width=device-width, initial-scale=1.0, maximum-scale=1.0, user-scalable=no');
      document.head.appendChild(meta);
    },
    initRendering() {
      self.colorField = document.querySelector("#color-field");
      self.colorSwatch = document.querySelector("#color-swatch");
      self.cvs = document.querySelector("#viewport-canvas");
      self.glWindow = new GLWindow(self.cvs);
      self.place = new Place(self.glWindow);
      self.place.initConnection("/init_canvas");
      self.color = new Uint8Array([0, 0, 0])
    },
    initEventListeners() {
      self.cvs.addEventListener('mousedown', this.onMouseDown);
      self.cvs.addEventListener('contextmenu', () => {return false;});
      self.cvs.addEventListener('touchstart', this.onTouchStart);
      self.cvs.addEventListener('touchend', this.onTouchEnd);
      document.addEventListener('keydown', this.onKeyDown);
      self.colorField.addEventListener('change', this.onChange);
      
      document.addEventListener('touchmove', this.onTouchMove);
      window.addEventListener('wheel', this.onWheel);
      document.querySelector("#zoom-in").addEventListener("click", () => {zoomIn(1.2);});
      document.querySelector("#zoom-out").addEventListener("click", () => {zoomOut(1.2);});
      document.addEventListener('mousemove', this.onMouseMove);
      window.addEventListener("resize", ev => {
        self.glWindow.updateViewScale();
        self.glWindow.draw();
      });
      document.addEventListener("mouseup", (ev) => {
        self.dragdown = false;
        document.body.style.cursor = "auto";
      });
    },
    sendPixel(x, y, color) {
      const pixel = {
        x: Math.ceil(x),
        y: Math.ceil(y),
        color: [color[0], color[1], color[2]],
      };
      console.log(JSON.stringify(pixel));
      this.ws.send(JSON.stringify(pixel));
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
      console.log("Received a pixel from server: ", pixel);
      self.place.setPixel(pixel.X, pixel.Y, new Uint8Array([pixel.Color[0], pixel.Color[1], pixel.Color[2]]));
    },
    onMouseDown(ev) {
      switch (ev.button) {
        case 0:
          self.dragdown = true;
          self.lastMovePos = { x: ev.clientX, y: ev.clientY };
          break;
        case 1:
          pickColor({ x: ev.clientX, y: ev.clientY });
          break;
        case 2:
          if (ev.ctrlKey) {
            pickColor({ x: ev.clientX, y: ev.clientY });
          } else {
            ev.stopPropagation();
            this.drawPixel({ x: ev.clientX, y: ev.clientY }, self.color);
          }
      }
    },
    drawPixel(pos, color) {
      pos = self.glWindow.click(pos);
      if (pos) {
        const oldColor = self.glWindow.getColor(pos);
        for (let i = 0; i < oldColor.length; i++) {
          if (oldColor[i] != color[i]) {
            //TODO чек таймера
            this.sendPixel(pos.x, pos.y, color);
            return true;
          }
        }
      }
      return false;
    },
    pickColor(pos) {
      self.color = self.glWindow.getColor(self.glWindow.click(pos));
      let hex = "#";
      for (let i = 0; i < self.color.length; i++) {
        let d       = self.color[i].toString(16);
        if (d.length == 1) d = "0" + d;
        hex += d;
      }
      self.colorField.value = hex.toUpperCase();
      self.colorSwatch.style.backgroundColor = hex;
    },
    zoomIn(factor) {
      let zoom = self.glWindow.getZoom();
      self.glWindow.setZoom(zoom * factor);
      self.glWindow.draw();
    },
    zoomOut(factor) {
      let zoom = self.glWindow.getZoom();
      self.glWindow.setZoom(zoom / factor);
      self.glWindow.draw();
    },
    onKeyDown(ev) {
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
    },
    onChange() {
      let hex = self.colorField.value.replace(/[^A-Fa-f0-9]/g, "").toUpperCase();
      hex = hex.substring(0, 6);
      while (hex.length < 6) {
        hex += "0";
      }
      self.color[0] = parseInt(hex.substring(0, 2), 16);
      self.color[1] = parseInt(hex.substring(2, 4), 16);
      self.color[2] = parseInt(hex.substring(4, 6), 16);
      hex = "#" + hex;
      self.colorField.value = hex;
      self.colorSwatch.style.backgroundColor = hex;
    },
    onTouchMove(ev) {
      self.touchID++;
      if (self.touchScaling) {
        let dist = Math.hypot(
            ev.touches[0].pageX - ev.touches[1].pageX,
            ev.touches[0].pageY - ev.touches[1].pageY);
        if (self.lastScalingDist != null) {
          let delta = self.lastScalingDist - dist;
          if (delta < 0) {
            zoomIn(1 + Math.abs(delta) * 0.003);
          } else {
            zoomOut(1 + Math.abs(delta) * 0.003);
          }
        }
        self.lastScalingDist = dist;
      } else {
        let movePos = { x: ev.touches[0].clientX, y: ev.touches[0].clientY };
        self.glWindow.move(movePos.x - self.lastMovePos.x, movePos.y - self.lastMovePos.y);
        self.glWindow.draw();
        self.lastMovePos = movePos;
      }
    },
    onWheel(ev) {
      let zoom = self.glWindow.getZoom();
        if (ev.deltaY > 0) {
          zoom /= 1.05;
        } else {
          zoom *= 1.05;
        }
      self.glWindow.setZoom(zoom);
      self.glWindow.draw();
    },
    onMouseMove(ev) {
      const movePos = { x: ev.clientX, y: ev.clientY };
      if (self.dragdown) {
        self.glWindow.move(movePos.x - self.lastMovePos.x, movePos.y - self.lastMovePos.y);
        self.glWindow.draw();
        document.body.style.cursor = "grab";
      }
      self.lastMovePos = movePos;
    },
    onTouchStart(ev) {
      let thisTouch = self.touchID;
      this.touchstartTime = (new Date()).getTime();
      self.lastMovePos = { x: ev.touches[0].clientX, y: ev.touches[0].clientY };
      if (ev.touches.length === 2) {
        self.touchScaling = true;
        self.lastScalingDist = null;
      }
      setTimeout(() => {
        if (thisTouch == self.touchID) {
          pickColor(self.lastMovePos);
          navigator.vibrate(200);
        }
      }, 350);
    },
    onTouchEnd() {
      self.touchID++;
      let elapsed = (new Date()).getTime() - self.touchstartTime;
      if (elapsed < 100) {
        if (drawPixel(self.lastMovePos, self.color)) {
          navigator.vibrate(10);
        };
      }
      if (ev.touches.length === 0) {
        self.touchScaling = false;
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