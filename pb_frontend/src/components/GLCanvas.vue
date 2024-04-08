<template>
  <div oncontextmenu="return false;">
    <canvas @touchstart="onTouchStart" @mousedown="onMouseDown"
    @touchend="onTouchEnd" @mousemove="coordsUpdate" 
    @contextmenu="() => {return false;}" id="viewport-canvas"></canvas>
    <div id="ui-wrapper" hide="true">
      <p id="loading-p"></p>
      <div id="color-wrapper">
        <!-- <div id="color-swatch"></div> -->
        <div v-for="(color, index) in palette" 
          :key="index" 
          :item="color" 
          @click="(ev) => {
            ev.preventDefault();
            this.selectSwatch(index);
          }" 
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
      timerValue: null,
      palette: [],
      activeSwatch: 0,
      loaded: false,
      savedPixels: [],
    }
  },
  created() {
    this.setViewport();
  },
  watch: {
    loaded(newVal) {
      if (newVal) {
        this.renderSavedPIxels();
      }
    },
  },
  mounted() {
    document.title='megapixelbattle'
    this.$data.colorField = document.querySelector("#color-field");
    this.$data.cvs = document.querySelector("#viewport-canvas");
    this.$data.timerValue = document.querySelector("#timer-value");
    this.$data.glWindow = new GLWindow(this.$data.cvs);
    this.$data.place = new Place(this.$data.glWindow, document.querySelector("#ui-wrapper"), document.querySelector("#loading-p"));
    this.$data.color = new Uint8Array([0, 0, 0]);
    this.$data.palette = ["#000000", "#FFFFFF", "#FF0000", "#00FF00"];
    this.initConnection("/init_canvas");
    this.initEventListeners();
    const platform = navigator.platform.toLowerCase();
    if (/(android|webos|iphone|ipad|ipod|blackberry|windows phone)/.test(platform)) {
      console.log("oh my fucking god mobile user");
    }
    this.connectToWebSocket();
    // this.setSwatchesArr(this.$refs.swatches);
    // this.setField(document.querySelector("#color-field"));
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
      // console.log(this.activeSwatch);
      this.activeSwatch = index;
      this.getSwatches()[this.activeSwatch].style.border = '2px solid white';
      let hex = this.palette[this.activeSwatch];
      hex = hex.substring(1, 7);
      while (hex.length < 6) {
        hex += "0";
      }
      // console.log(hex);
      // console.log(this.color);
      this.color[0] = parseInt(hex.substring(0, 2), 16);
      this.color[1] = parseInt(hex.substring(2, 4), 16);
      this.color[2] = parseInt(hex.substring(4, 6), 16);
      // this.color = this.palette[this.activeSwatch];
      // console.log(this.color);
    },
    setViewport() {
      const meta = document.createElement('meta');
      meta.setAttribute('name', 'viewport');
      meta.setAttribute('content', 'width=device-width, initial-scale=1.0, maximum-scale=1.0, user-scalable=no');
      document.head.appendChild(meta);
    },
    initConnection(endpoint) {
      this.$data.place.loadingp.innerHTML = "loading canvas"
      fetch(endpoint)
			.then(async resp => {
				let buf = await this.$data.place.downloadProgress(resp);
				await this.$data.place.setImage(buf);
        this.loaded = true;
        this.$data.place.loadingp.innerHTML = "";
        this.$data.place.uiwrapper.setAttribute("hide", true);
			})
      .catch((error) => {
        this.$router.push('/login');
      });
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
      // this.cvs.addEventListener('mousedown', this.onMouseDown);
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
        // console.log(JSON.stringify(pixel));
        // this.place.setPixel(pixel.x, pixel.y, new Uint8Array([0, 0, 0]));
        this.ws.send(JSON.stringify(pixel));
        
        this.seconds = 2;
        this.timerValue.style.visibility = "visible";
        // console.log(this.timerValue);
        
        this.timer = setInterval(() => {
          if (this.seconds > 0) {
            this.seconds--;
          } else {
            clearInterval(this.timer);
            this.timerRunning = false;
            this.timerValue.style.visibility = "hidden";
          }
        }, 1000);
      } else {
        this.timerValue.style.fontWeight = "bold"
        this.timerValue.style.color = "firebrick"
        this.timerValue.style.fontSize = "22px"
        setInterval(() => {
          this.timerValue.style.fontSize = "16px"
          this.timerValue.style.fontWeight = "normal"
          this.timerValue.style.color = "black"
        }, 200);
      }
    },
    connectToWebSocket() {
      const url = new URL('/ws', location.href);
      url.protocol = 'wss';
      this.ws = new WebSocket(url);
      this.ws.addEventListener('open', (event) => {this.onWebSocketOpen(event)});
      this.ws.addEventListener('message', (event) => {this.handleNewPixel(event)});
    },
    onWebSocketOpen() {
      // console.log("damnit websocket connected");
    },
    handleNewPixel(event) {
      const pixel = JSON.parse(event.data);
      // console.log("Received a pixel from server: ", pixel);
      
      if (!this.loaded) {
        this.savedPixels.push(pixel);
      } else {
        this.place.setPixel(pixel.X, pixel.Y, new Uint8Array([pixel.Color[0], pixel.Color[1], pixel.Color[2]]));
      }
    },
    renderSavedPIxels() {
      for (const pixel of this.savedPixels) {
        this.place.setPixel(pixel.X, pixel.Y, new Uint8Array([pixel.Color[0], pixel.Color[1], pixel.Color[2]]));
        // console.log("rendering...");
      }
      // console.log("values rendered");
      this.savedPixels = []; // Clear saved pixels after replaying
    },
    onMouseDown(ev) {
      let self = this;
      switch (ev.button) {
        case 0:
          this.dragdown = true;
          this.lastMovePos = { x: ev.clientX, y: ev.clientY };
          break;
        case 1:
          ev.preventDefault();
          self.pickColor({ x: ev.clientX, y: ev.clientY });
          break;
        case 2:
          if (ev.ctrlKey) {
            ev.preventDefault();
            self.pickColor({ x: ev.clientX, y: ev.clientY });
            console.log("kek ended");
          } else {
            ev.preventDefault();
            self.drawPixel({ x: ev.clientX, y: ev.clientY }, this.color);
          }
      }
      // console.log(this.colorField.style.backgroundColor);
      // console.log(this.getSwatches()[this.activeSwatch]);
      // console.log(this.currentSwatches[this.activeSwatch].style.backgroundColor);
      // console.log(this.currentColorField.value);
    },
    drawPixel(pos, color) {
      pos = this.glWindow.click(pos);
      if (pos) {
        const oldColor = this.glWindow.getColor(pos);
        for (let i = 0; i < oldColor.length; i++) {
          if (oldColor[i] != color[i]) {
            console.log("sending: ", pos.x, pos.y, color);
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
      var event = new Event('change');

      this.colorField.dispatchEvent(event);
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
      hex = hex.substring(0, 6);
      while (hex.length < 6) {
        hex += "0";
      }
      // console.log(this.color);
      this.color[0] = parseInt(hex.substring(0, 2), 16);
      this.color[1] = parseInt(hex.substring(2, 4), 16);
      this.color[2] = parseInt(hex.substring(4, 6), 16);
      // console.log(this.color);

      this.palette[this.activeSwatch] = "#" + hex;
      this.colorField.value = this.palette[this.activeSwatch];

      // this.getSwatches()[this.activeSwatch].style.backgroundColor = hex;
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
        // console.log("ontouchmove");
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
        this.drawPixel(this.lastMovePos, this.color);
      }
      if (ev.touches.length === 0) {
        this.touchScaling = false;
      }
      // console.log("touchend");
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
  filter: drop-shadow(1px 1px grey);
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

#loading-p {
	position: absolute;
	top: 50%;
	left: 50%;
	font-size: 1.4em;
	transform: translate(-50%, -50%);
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
  transition: all 0.5s ease-in-out 0.5s;
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