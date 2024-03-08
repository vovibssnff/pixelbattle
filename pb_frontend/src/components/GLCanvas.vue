<template>
  <div oncontextmenu="return false;">
    <canvas @mousedown="onMouseDown" @touchstart="onTouchStart" 
    @touchend="onTouchEnd" @mousemove="coordsUpdate" 
    @contextmenu="() => {return false;}" id="viewport-canvas"></canvas>
    <div id="ui-wrapper" hide="true">
      <div id="color-wrapper">
        <!-- <div id="color-swatch"></div> -->
        <div v-for="(color, index) in palette" 
          :key="index" 
          :item="color" 
          @click="() => {this.selectSwatch(index);}" 
          class="color-swatch" 
          :style="{ backgroundColor: color }" 
          ref="swatches" 
        ></div>

        <input @change="onChange" id="color-field" type="text" :placeholder="palette[activeSwatch]" :value="palette[activeSwatch]" />
        <!-- <input @change="onChange" id="color-field" type="text" placeholder="#000000" value="#000000" /> -->
      </div>
      <div id="zoom-wrapper">
        <button @click="() => {this.zoomOut(1.2);}" class="zoom-button" id="zoom-out">-</button>
        <button @click="() => {this.zoomIn(1.2);}" class="zoom-button" id="zoom-in">+</button>
      </div>
      <div id="cursor-info">
        <span id="x-coordinate">{{ Math.floor(this.val_x) }}</span>, <span id="y-coordinate">{{ Math.floor(this.val_y) }}</span>
      </div>
      <div id="timer">
        <span id="timer-value">{{ this.seconds }}</span>
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
      val_x: null,
      val_y: null,
      pos: null,
      ws: null,
      connected: false,
      colorField: null,
      colorSwatches: [],
      cvs: null,
      glWindow: null,
      place: null,
      color: null,
      dragdown: null,
      touchID: null,
      touchScaling: null,
      lastMovePos: null,
      lastScalingDist: null,
      timerRunning: false,
      seconds: 0,
      timer: null,
      palette: [],
      activeSwatch: null
    }
  },
  created() {
    this.setViewport();
  },
  mounted() {
    this.$data.colorField = document.querySelector("#color-field");
    this.$data.cvs = document.querySelector("#viewport-canvas");
    this.$data.glWindow = new GLWindow(this.$data.cvs);
    this.$data.place = new Place(this.$data.glWindow);
    this.$data.color = new Uint8Array([0, 0, 0]);
    this.$data.palette = ["#000000", "#FFFFFF", "#FF0000", "#00FF00"]
    this.place.initConnection("/init_canvas");
    this.initEventListeners();
    const platform = navigator.platform.toLowerCase();
    if (/(android|webos|iphone|ipad|ipod|blackberry|windows phone)/.test(platform)) {
      this.initMobileEventListeners();
      console.log("oh my fucking god android user");
    }
    this.connectToWebSocket();
    window.alert("ПКМ - рисование, ЛКМ - навигация, CTRL+ПКМ - копирование цвета в палитру, https://www.color-hex.com/ - в помощь для подбора цветов");
  },
  methods: {
    getSwatches() {
      return this.$refs.swatches;
    },
    selectSwatch(index) {
      try {
        this.getSwatches()[this.activeSwatch].style.border = '2px solid black';
      } catch{}
      this.activeSwatch = index;
      this.getSwatches()[this.activeSwatch].style.border = '2px solid white';
      let hex = this.palette[this.activeSwatch];
      hex = hex.substring(1, 7);
      while (hex.length < 7) {
        hex += "0";
      }
      this.color[0] = parseInt(hex.substring(1, 3), 16);
      this.color[1] = parseInt(hex.substring(3, 5), 16);
      this.color[2] = parseInt(hex.substring(5, 7), 16);
      // this.color = this.palette[this.activeSwatch];
    },
    setViewport() {
      const meta = document.createElement('meta');
      meta.setAttribute('name', 'viewport');
      meta.setAttribute('content', 'width=device-width, initial-scale=1.0, maximum-scale=1.0, user-scalable=no');
      document.head.appendChild(meta);
    },
    initMobileEventListeners() {

    },
    initEventListeners() {
      document.addEventListener('keydown', this.onKeyDown);
      document.addEventListener('touchmove', this.onTouchMove);
      document.addEventListener('mousemove', this.onMouseMove);
      document.addEventListener("mouseup", () => {
        this.dragdown = false;
        document.body.style.cursor = "auto";
      });
      window.addEventListener('wheel', this.onWheel);
      window.addEventListener("resize", () => {
        this.glWindow.updateViewScale();
        this.glWindow.draw();
      });
    },
    coordsUpdate(ev) {
      try {
        this.pos = this.glWindow.click({ x: ev.clientX, y: ev.clientY });
        this.val_x = this.pos.x;
        this.val_y = this.pos.y;
      } catch {

      }
    },
    sendPixel(x, y, color) {
      if (!this.timerRunning) {
        this.timerRunning = true;
        const pixel = {
          x: Math.floor(x),
          y: Math.floor(y),
          color: [color[0], color[1], color[2]],
        };
        console.log(JSON.stringify(pixel));
        this.place.setPixel(pixel.x, pixel.y, new Uint8Array([0, 0, 0]));
        this.ws.send(JSON.stringify(pixel));
        
        this.seconds = 2;

        this.timer = setInterval(() => {
          if (this.seconds > 0) {
            this.seconds--;
          } else {
            clearInterval(this.timer);
            this.timerRunning = false;
          }
        }, 1000);
      }
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
      this.place.setPixel(pixel.X, pixel.Y, new Uint8Array([pixel.Color[0], pixel.Color[1], pixel.Color[2]]));
    },
    onMouseDown(ev) {
      switch (ev.button) {
        case 0:
          this.dragdown = true;
          this.lastMovePos = { x: ev.clientX, y: ev.clientY };
          break;
        case 1:
          this.pickColor({ x: ev.clientX, y: ev.clientY });
          break;
        case 2:
          if (ev.ctrlKey) {
            this.pickColor({ x: ev.clientX, y: ev.clientY });
          } else {
            ev.preventDefault();
            this.drawPixel({ x: ev.clientX, y: ev.clientY }, this.color);
          }
      }
    },
    drawPixel(pos, color) {
      pos = this.glWindow.click(pos);
      if (pos) {
        const oldColor = this.glWindow.getColor(pos);
        for (let i = 0; i < oldColor.length; i++) {
          if (oldColor[i] != color[i]) {
            console.log("sending: ", pos.x, pos.y);
            this.sendPixel(pos.x, pos.y, color);
            return true;
          }
        }
      }
      return false;
    },
    pickColor(pos) {
      this.color = this.glWindow.getColor(this.glWindow.click(pos));
      let hex = "#";
      for (let i = 0; i < this.color.length; i++) {
        let d = this.color[i].toString(16);
        if (d.length == 1) d = "0" + d;
        hex += d;
      }
      this.colorField.value = hex.toUpperCase();
      this.getSwatches()[this.activeSwatch].style.backgroundColor = hex;
    },
    zoomIn(factor) {
      let zoom = this.glWindow.getZoom();
      this.glWindow.setZoom(zoom * factor);
      this.glWindow.draw();
    },
    zoomOut(factor) {
      let zoom = this.glWindow.getZoom();
      this.glWindow.setZoom(zoom / factor);
      this.glWindow.draw();
    },
    onKeyDown(ev) {
      switch (ev.keyCode) {
        case 189:
        case 173:
          ev.preventDefault();
          this.zoomOut(1.2);
          break;
        case 187:
        case 61:
          ev.preventDefault();
          this.zoomIn(1.2);
          break;
      }
    },
    onChange() {
      let hex = this.colorField.value.replace(/[^A-Fa-f0-9]/g, "").toUpperCase();
      console.log(this.colorField.value.replace(/[^A-Fa-f0-9]/g, "").toUpperCase());
      hex = hex.substring(0, 6);
      while (hex.length < 6) {
        hex += "0";
      }
      this.color[0] = parseInt(hex.substring(0, 2), 16);
      this.color[1] = parseInt(hex.substring(2, 4), 16);
      this.color[2] = parseInt(hex.substring(4, 6), 16);

      this.palette[this.activeSwatch] = "#" + hex;
      this.colorField.value = this.palette[this.activeSwatch];
      console.log(this.getSwatches);

      this.getSwatches()[this.activeSwatch].style.backgroundColor = hex;
    },
    onTouchMove(ev) {
      this.touchID++;
      if (this.touchScaling && ev.touches.length!=1) {
        let dist = Math.hypot(
            ev.touches[0].pageX - ev.touches[1].pageX,
            ev.touches[0].pageY - ev.touches[1].pageY);
        if (this.lastScalingDist != null) {
          let delta = this.lastScalingDist - dist;
          if (delta < 0) {
            this.zoomIn(1 + Math.abs(delta) * 0.003);
          } else {
            this.zoomOut(1 + Math.abs(delta) * 0.003);
          }
        }
        this.lastScalingDist = dist;
      } else {
        try {
          this.pos = this.glWindow.click({ x: ev.touches[0].clientX, y: ev.touches[0].clientY });
          this.val_x = this.pos.x;
          this.val_y = this.pos.y;
        } catch {}
        let movePos = { x: ev.touches[0].clientX, y: ev.touches[0].clientY };
        this.glWindow.move(movePos.x - this.lastMovePos.x, movePos.y - this.lastMovePos.y);
        this.glWindow.draw();
        this.lastMovePos = movePos;
      }
    },
    onWheel(ev) {
      let zoom = this.glWindow.getZoom();
      if (ev.deltaY > 0) {
        zoom /= 1.05;
      } else {
        zoom *= 1.05;
      }
      this.glWindow.setZoom(zoom);
      this.glWindow.draw();
    },
    onMouseMove(ev) {
      const movePos = { x: ev.clientX, y: ev.clientY };
      if (this.dragdown) {
        this.glWindow.move(movePos.x - this.lastMovePos.x, movePos.y - this.lastMovePos.y);
        this.glWindow.draw();
        document.body.style.cursor = "grab";
      }
      this.lastMovePos = movePos;
      // console.log(movePos);
    },
    onTouchStart(ev) {
      ev.preventDefault();
      let thisTouch = this.touchID;
      this.touchstartTime = (new Date()).getTime();
      this.lastMovePos = { x: ev.touches[0].clientX, y: ev.touches[0].clientY };
      if (ev.touches.length === 2) {
        this.touchScaling = true;
        this.lastScalingDist = null;
      }
      setTimeout(() => {
        if (thisTouch == this.$data.touchID) {
          this.pickColor(this.lastMovePos);
          // navigator.vibrate(200);
        }
      }, 350);
    },
    onTouchEnd(ev) {
      this.$data.touchID++;
      let elapsed = (new Date()).getTime() - this.touchstartTime;
      if (elapsed < 100) {
        console.log(this.lastMovePos);
        this.drawPixel(this.lastMovePos, this.color);
      }
      if (ev.touches.length === 0) {
        this.touchScaling = false;
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

.color-swatch {
  width: 30px;
  height: 30px;
  background-color: #000000;
  border: 2px solid black;
  cursor: pointer;
  pointer-events: all;
}

#color-field { 
  font-size: 16px;
  height: 30px;
  padding: 1px;
  border: none;
  outline: none;
  pointer-events: all;
  background-color: transparent; /* Remove white background */
  width: 40%; /* Decrease width */
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
  border: 2px solid black;
  cursor: pointer;
  pointer-events: all;
  user-select: none;
}

#cursor-info {
  position: absolute;
  top: 16px;
  left: 16px;
  font-size: 16px;
}

#timer {
  position: absolute;
  top: 16px;
  right: 16px;
  font-size: 16px;
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