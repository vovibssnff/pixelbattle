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

      <label for="checkbox" id="howto">?</label>
      <input hidden type="checkbox" id="checkbox">
      <label for="checkbox" class="modal-overlay">
        <div class="modal">
          <!-- <h1>Добро пожаловать на Pixelbattle!</h1> -->
<h1>О мудром владении цветами и пергаментом на веб-сайте</h1><br/><br/>

Слушай же, о путник, и ведай, как управляться с волшебной палитрой:
<br/><br/>
ПКМ — держи перо крепко, дабы узоры чертить.<br/>
ЛКМ — ведай путь свой верно, дабы по холсту странствовать.<br/>
CTRL + ПКМ — познай цвет истинный, да занеси его в палитру свою.<br/><br/>
<span>В помощь тебе — <a href="https://color-hex.com">цветная книга мудрецов</a></span><br/>
<br/>
Время течет, как река, и ставить пиксели дозволено лишь раз в три удара сердца.
<br/>
Пусть же рука твоя будет тверда, а дух — неколебим. Веди перо свое, дабы создать творение, коему позавидуют сами небеса!
          <label for="checkbox">X</label>
        </div>
      </label>

      <div id="zoom-wrapper">
        <button @click="() => {this.zoomOut(1.2);}" class="zoom-button" id="zoom-out">-</button>
        <button @click="() => {this.zoomIn(1.2);}" class="zoom-button" id="zoom-in">+</button>
      </div>
      <div id="cursor-info">
        <span id="x-coordinate">{{ Math.floor(this.val_x) }}</span>, <span id="y-coordinate">{{ Math.floor(this.val_y) }}</span>
      </div>
      <div id="timer">{{ this.seconds }}
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
      secondTimer: null,
      timerValue: null,
      palette: [],
      activeSwatch: 0,
      loaded: false,
      savedPixels: [],
      isGod: null
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
    document.title='Pixelbattle'
    this.$data.howToOpen = 0

    this.$data.colorField = document.querySelector("#color-field");
    this.$data.cvs = document.querySelector("#viewport-canvas");
    this.$data.timerValue = document.querySelector("#timer");
    this.$data.cursorInfo = document.querySelector("#cursor-info");
    this.$data.swatches = document.getElementsByClassName("color-swatch");
    this.$data.glWindow = new GLWindow(this.$data.cvs);
    this.$data.place = new Place(this.$data.glWindow, document.querySelector("#ui-wrapper"), document.querySelector("#loading-p"));
    this.$data.color = new Uint8Array([0, 0, 0]);
    this.$data.palette = ["#000000", "#FFFFFF", "#FF0000", "#00FF00"];
    if (process.env.NODE_ENV === 'production') {
      console.log("production")
      this.initConnection("/init_canvas");
      this.connectToWebSocket("/ws");
      this.initEventListeners();
    } else {
      console.log("dev")
      this.loaded = true;
      this.ws = {}
      this.ws.send = function() { return; };

      // Create canvas element
      const canvas = document.createElement('canvas');
      canvas.width = 100;
      canvas.height = 200;
      const ctx = canvas.getContext('2d');
      const grad=ctx.createLinearGradient(0,0, 280,0);
      grad.addColorStop(0, "lightblue");
      grad.addColorStop(1, "darkblue");

      // Fill rectangle with gradient
      ctx.fillStyle = grad;
      ctx.fillRect(0, 0, canvas.width, canvas.height);

      canvas.toBlob(blob => {
        this.$data.place.setImage(blob);
      })
      this.$data.place.loadingp.innerHTML = "";
      this.$data.place.uiwrapper.setAttribute("hide", true);
      this.$data.isGod = true;
      this.initEventListeners();
      
    }
    const platform = navigator.platform.toLowerCase();
    if (/(android|webos|iphone|ipad|ipod|blackberry|windows phone)/.test(platform)) {
      console.log("oh my ... god mobile user");
    }
    // this.setSwatchesArr(this.$refs.swatches);
    // this.setField(document.querySelector("#color-field"));
    // window.alert("ПКМ - рисование, ЛКМ - навигация, CTRL+ПКМ - копирование цвета в палитру, https://www.color-hex.com/ - в помощь для подбора цветов");
  },
  methods: {
    selectSwatch(index) {
      if (this.activeSwatch == index) {
        return;
      }
      try {
        this.swatches[this.activeSwatch].style.borderWidth = '0';
        this.swatches[this.activeSwatch].style.width = '30px';
      } catch{}
      // console.log(this.activeSwatch);
      this.activeSwatch = index;
      this.swatches[this.activeSwatch].style.borderWidth = '2px 3px';
      this.swatches[this.activeSwatch].style.width = '31px';
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
        this.$data.isGod = resp.headers.get("Is-God");
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
    send(x, y, color) {
      const pixel = {
          x: Math.floor(x),
          y: Math.floor(y),
          color: [color[0], color[1], color[2]],
        };
        this.ws.send(JSON.stringify(pixel));
    },
    sendPixel(x, y, color) {
      // console.log(this.isGod);
      if (this.isGod=="true") {
        this.send(x, y, color);
        return;
      }
      if (!this.timerRunning) {
        this.timerRunning = true;
        this.send(x, y, color);
        
        this.seconds = 3;
        this.timerValue.style.opacity = 1;
        
        this.timer = setInterval(() => {
          if (this.seconds > 1) {
            this.seconds--;
          } else {
            clearInterval(this.timer);
            this.timerRunning = false;
            this.timerValue.style.opacity = 0;
          }
        }, 1000);
      } else {
        this.timerValue.style.fontWeight = "bold";
        this.timerValue.style.color = "firebrick";
        this.timerValue.style.fontSize = "28px";
        this.timerValue.style.width = "30px";
        this.timerValue.style.borderColor = "firebrick"
        this.timer
        clearInterval(this.secondTimer);
        this.secondTimer = setInterval(() => {
            // Apply new style which transitions smoothly due to the CSS
            this.timerValue.style.fontSize = "16px";
            this.timerValue.style.fontWeight = "normal";
            this.timerValue.style.color = "#134293";
            this.timerValue.style.width = "20px";
            this.timerValue.style.borderColor = "#134293"
        }, 1000);
      }
    },
    connectToWebSocket(endpoint) {
      const url = new URL(endpoint, location.href);
      url.protocol = 'wss';
      this.ws = new WebSocket(url);
      this.ws.addEventListener('message', (event) => {this.handleNewPixel(event)});
    },
    handleNewPixel(event) {
      const pixel = JSON.parse(event.data);
      
      if (!this.loaded) {
        this.savedPixels.push(pixel);
      } else {
        this.place.setPixel(pixel.X, pixel.Y, new Uint8Array([pixel.Color[0], pixel.Color[1], pixel.Color[2]]));
      }
    },
    renderSavedPIxels() {
      for (const pixel of this.savedPixels) {
        this.place.setPixel(pixel.X, pixel.Y, new Uint8Array([pixel.Color[0], pixel.Color[1], pixel.Color[2]]));
      }
      this.savedPixels = [];
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
          } else {
            ev.preventDefault();
            self.drawPixel({ x: ev.clientX, y: ev.clientY }, this.color);
          }
      }
    },
    drawPixel(pos, color) {
      pos = this.glWindow.click(pos);
      if (pos) {
        const oldColor = this.glWindow.getColor(pos);
        for (let i = 0; i < oldColor.length; i++) {
          if (oldColor[i] != color[i]) {
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

      // this.swatches[this.activeSwatch].style.backgroundColor = hex;
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
@font-face {
  font-family: 'Calligrapher';
  src: url('@/fonts/calligrapher.ttf') format('truetype');
}

* {
  padding: 0;
  margin: 0;
  font-family: 'Calligrapher', Times, serif;
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
  background-image: url("@/img/bg.jpg");
  background-size: cover;
}

#ui-wrapper {
  position: fixed;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  background-color: #ffffff;
  transition: background 1s;

  color: black;
  /* filter: drop-shadow(1px 1px grey); */
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
  width: min-content;
  border: 2px solid #134293;
  background-color: #E1DBCE;
  border-radius: 10px;
  overflow: hidden;
}

.color-swatch {
  width: 30px;
  height: 30px;
  background-color: #000000;
  /* border: px solid #134293; */
  border-width: 0;
  border-style: solid;
  border-color: #134293;
  cursor: pointer;
  pointer-events: all;
  box-sizing: border-box;
}

.color-swatch:first-of-type {
  border-width: 2px 3px;
  width: 31px;
}

#color-field { 
  font-size: 18px;
  color: black;
  height: 30px;
  line-height: 30px;
  box-sizing: border-box;
  margin-left: 6px;
  border: none;
  outline: none;
  pointer-events: all;
  background-color: transparent; /* Remove white background */
  width: 8.5ch;
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
  border-radius: 10px;
  border: 2px solid #134293;
  background-color: #E1DBCE;
}

.zoom-button {
  color: #134293;
  font-family: monospace;
  width: 30px;
  height: 30px;
  border: none;
  background: none;
  outline: none;
  font-size: 24px;
  cursor: pointer;
  pointer-events: all;
  user-select: none;
}

.zoom-button:first-of-type {
  border-right: 2px solid #134293
}

#cursor-info {
  position: absolute;
  color: black;
  background-color: #E1DBCE;
  padding: 5px;
  width: 50px;
  height: 20px;
  line-height: 20px;
  text-align: center;
  border: 2px solid #134293;
  border-radius: 10px;
  top: 16px;
  left: 50%;
  transform: translateX(-50%);
  font-size: 16px;
}

#timer {
  position: absolute;
  opacity: 0;
  pointer-events: none;
  top: 60px;
  left: 50%;
  width: 20px;
  padding: 5px;
  color: black;
  border-radius: 10px;
  text-align: center;
  background-color: #E1DBCE;
  border: 2px solid #134293;
  transform: translateX(-50%);
  font-size: 16px;
  transition: all 0.2s ease-in-out;
}

#howto {
  position: absolute;
  top: 16px;
  right: 16px;
  font-size: 16px;
  text-align: center;
  border-radius: 10px;
  background-color: #E1DBCE;
  border: 2px solid #134293;
  line-height: 30px;
  width: 30px;
  height: 30px;
  cursor: help;
  user-select: none;
  z-index: 1;
  pointer-events: all;
}

@media (hover: none) {
  #help-text-desktop {
    display: none;
  }

  #help-text-mobile {
    display: inline;
  }
}

.modal-overlay {
  position: fixed;
  top: 0;
  right: 0;
  bottom: 0;
  left: 0;
  background-color: rgba(0, 0, 0, 0.5);
  opacity: 0;
  pointer-events: none;
  transition: opacity 0.1s ease-in-out;
  z-index: 100;
}

.modal {
  font-size: 18px;
  border-radius: 10px;
  display: flex;
  flex-direction: column;
  /* justify-content: center; */
  padding: 50px;
  padding-inline: 10%;
  box-sizing: border-box;
  position: fixed;
  top: 50%;
  left: 50%;
  width: 90%;
  transform: translate(-50%, -52%);
  background-image: url("@/img/paper-bg.jpg");
  background-size: cover;
  transition: all 0.5s ease-in-out;
  border: 2px solid #134293;
  text-align: center;
}

input:checked ~ .modal-overlay {
  display: block;
  opacity: 1;
  pointer-events: auto;
}
input:checked ~ .modal-overlay .modal {
  transform: translate(-50%, -50%);
}

.modal-overlay label {
  position: absolute;
  right: 16px;
  top: 16px;
  color: #134293;
}

h1, h2, h3 {
  color: #134293;
  padding-inline: 10%;
}

@media (max-width: 600px) {
  .modal {
    font-size: 14px;
  }
}

a {
  color: #134293;
}
</style>