<template>
  <div @contextmenu.prevent>
    <canvas
      id="viewport-canvas" @touchstart="onTouchStart"
      @mousedown="onMouseDown" @touchend="onTouchEnd"
      @mousemove="coordsUpdate" @contextmenu.prevent
    ></canvas>
    <div id="ui-wrapper" hide="true">
      <p id="loading-p"></p>
      <div id="color-wrapper">
        <!-- <div id="color-swatch"></div> -->
        <div
          v-for="(swatchHex, index) in palette" 
          :key="index" 
          ref="swatches" 
          :item="swatchHex" 
          class="color-swatch" 
          :style="{ backgroundColor: swatchHex }" 
          @click="(ev) => {
            ev.preventDefault();
            selectSwatch(index);
          }" 
        ></div>

        <input id="color-field" type="text" :placeholder="palette[activeSwatch]" :value="palette[activeSwatch]" @change="onChange" />
        <!-- <input @change="onChange" id="color-field" type="text" placeholder="#000000" value="#000000" /> -->
      </div>  

      <label id="howto" for="checkbox">?</label>
      <input id="checkbox" hidden type="checkbox">
      <label for="checkbox" class="modal-overlay">
        <div class="modal">
          <!-- <h1>Добро пожаловать на Pixelbattle!</h1> -->
          <h1>О мудром владении цветами и пергаментом на веб-сайте</h1><br/><br/>

          Слушай же, о путник, и ведай, как управляться с волшебной палитрой:
          <br/><br/>
          ПКМ (передвижение пальца на телефоне) — держи перо крепко, дабы узоры чертить.<br/>
          ЛКМ (касание экрана на телефоне) — ведай путь свой верно, дабы по холсту странствовать.<br/>
          CTRL + ПКМ (зажать 1 секунду на телефоне) — познай цвет истинный, да занеси его в палитру свою.<br/><br/>
          <span>В помощь тебе — <a target="_blank" href="https://color-hex.com">цветная книга мудрецов</a></span><br/>
          <br/>
          Время течет, как река, и ставить пиксели дозволено лишь раз в один удар сердца.
          <br/>
          Пусть же рука твоя будет тверда, а дух — неколебим. Веди перо свое, дабы создать творение, коему позавидуют сами небеса!
          <label for="checkbox">X</label>
        </div>
      </label>

      <div id="zoom-wrapper">
        <button id="zoom-out" class="zoom-button" @click="() => {zoomOut(1.2);}">-</button>
        <button id="zoom-in" class="zoom-button" @click="() => {zoomIn(1.2);}">+</button>
      </div>
      <div id="cursor-info">
        <span id="x-coordinate">{{ Math.floor(val_x) }}</span>, <span id="y-coordinate">{{ Math.floor(val_y) }}</span>
      </div>
      <div id="timer">
        {{ seconds }}
      </div>
      <div id="running-line">
        <div id="ad">
          <a target="_blank" href="https://vk.ru/itmomegabattle" class="ad-link">
            <svg class="svg-icon" style="width:26px;vertical-align:middle;fill:currentColor;overflow:hidden;" viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg"><path d="M12.77 19.15c-7.8 0-12.25-5.34-12.42-14.22h3.9c.12 6.5 3 9.26 5.27 9.83V4.93h3.68v5.61c2.24-.24 4.6-2.78 5.4-5.61h3.68a10.9 10.9 0 0 1-5.92 7.13 11.34 11.34 0 0 1 6.93 7.09h-4.06a7.16 7.16 0 0 0-6.03-5.06v5.06h-.43z"/></svg>
            <span>VK</span>
          </a>
          <a target="_blank" href="https://t.me/itmomegabattle" class="ad-link">
            <svg class="svg-icon" style="width:26px;vertical-align:middle;fill:currentColor;overflow:hidden;" viewBox="0 0 1024 1024" xmlns="http://www.w3.org/2000/svg"><path d="M417.28 795.733333l11.946667-180.48 327.68-295.253333c14.506667-13.226667-2.986667-19.626667-22.186667-8.106667L330.24 567.466667 155.306667 512c-37.546667-10.666667-37.973333-36.693333 8.533333-55.466667l681.386667-262.826666c31.146667-14.08 61.013333 7.68 49.066666 55.466666l-116.053333 546.56c-8.106667 38.826667-31.573333 48.213333-64 30.293334L537.6 695.466667l-84.906667 82.346666c-9.813333 9.813333-17.92 17.92-35.413333 17.92z"/></svg>
            <span>Telegram</span>
          </a>
        </div>
        <div id="tales-array">
          <div ref="firstTales" class="tales"></div>
          <div ref="secondTales" class="tales"></div>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
import GLWindow from '@/webgl/glwindow.js'
import Place from '@/webgl/place.js'
import RUMCollector from '@/utils/rum.js'

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
      isGod: null,
      rum: null,
      clientSeq: 0,
      pendingOptimistic: Object.create(null),
    }
  },
  watch: {
    loaded(newVal) {
      if (newVal) {
        this.renderSavedPIxels();
        
      }
    },
  },
  created() {
    this.setViewport();
  },
  mounted() {
    document.title='Pixelbattle';
    this.rum = new RUMCollector();
    this.rum.start();

    fetch("https://ruddnev.github.io/pixelbattle-tales/tales.txt") // Replace with your actual file URL
      .then(response => response.text())
      .then(text => {
        // Split by newlines, remove empty lines, and shuffle
        let talesArray = text.trim().replace("ё", "е").split("\n").filter(line => line.length > 0);
        talesArray.sort(() => Math.random() - 0.5);

        // Populate both .tales divs
        let firstTales = this.$refs.firstTales;
        let secondTales = this.$refs.secondTales;

        // Clear existing content
        firstTales.innerHTML = "";
        secondTales.innerHTML = "";

        // Add shuffled tales to both containers
        talesArray.forEach(tale => {
          let span = document.createElement("span");
          span.textContent = tale;
          firstTales.appendChild(span);
          secondTales.appendChild(span.cloneNode(true));
        });

        let totalLength = text.length; // Total characters in file
        let duration = totalLength / 10; // Example formula: 1s per 100 chars, min 5s

        // Apply animation duration dynamically
        firstTales.style.animationDuration = `${duration}s`;
        secondTales.style.animationDuration = `${duration}s`;

      })
      .catch(error => console.error("Error fetching tales:", error));

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
      this.initConnection("/api/canvas.png").then((snapshotMs) => {
        this.connectToWebSocket("/ws", snapshotMs);
        this.initEventListeners();
      });
    } else {
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
      // Mobile client — reserved for future UX tweaks
    }
    // this.setSwatchesArr(this.$refs.swatches);
    // this.setField(document.querySelector("#color-field"));
    // window.alert("ПКМ - рисование, ЛКМ - навигация, CTRL+ПКМ - копирование цвета в палитру, https://www.color-hex.com/ - в помощь для подбора цветов");
  },
  beforeUnmount() {
    this.rum?.stop();
  },
  methods: {
    selectSwatch(index) {
      if (this.activeSwatch == index) {
        return;
      }
      try {
        this.swatches[this.activeSwatch].style.borderWidth = '0';
        this.swatches[this.activeSwatch].style.width = '30px';
      } catch { /* swatch DOM may be missing during teardown */ }
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
      this.$data.place.loadingp.style.color = "white"
      this.$data.place.loadingp.innerHTML = "loading canvas"
      return fetch(endpoint, { cache: "no-store" })
			.then(async resp => {
				let buf = await this.$data.place.downloadProgress(resp);
				await this.$data.place.setImage(buf);
        this.loaded = true;
        this.$data.place.loadingp.innerHTML = "";
        this.$data.place.uiwrapper.setAttribute("hide", true);
        this.$data.isGod = resp.headers.get("Is-God");
        const raw = resp.headers.get("X-Snapshot-Ms");
        const ms = raw ? parseInt(raw, 10) : 0;
        return Number.isFinite(ms) ? ms : 0;
			})
      .catch(() => {
        this.$router.push('/login');
        return 0;
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
        const p = this.glWindow.fromClientXY(ev.clientX, ev.clientY);
        this.pos = this.glWindow.click(p);
        this.val_x = this.pos.x;
        this.val_y = this.pos.y;
      } catch { /* outside canvas */ }
    },
    colorsMatch(a, b) {
      return (
        Array.isArray(a) &&
        Array.isArray(b) &&
        a.length >= 3 &&
        b.length >= 3 &&
        Number(a[0]) === Number(b[0]) &&
        Number(a[1]) === Number(b[1]) &&
        Number(a[2]) === Number(b[2])
      );
    },
    applyOptimisticAndSend(x, y, color) {
      const xn = Math.floor(x);
      const yn = Math.floor(y);
      this.clientSeq = (this.clientSeq || 0) + 1;
      const seq = this.clientSeq >>> 0;
      const key = `${xn},${yn}`;
      const c = [color[0], color[1], color[2]];
      this.pendingOptimistic[key] = { seq, color: c };
      this.place.setPixel(xn, yn, new Uint8Array(c));
      this.send(xn, yn, c, seq);
    },
    send(x, y, color, seq) {
      const xn = Math.floor(x);
      const yn = Math.floor(y);
      const seq32 = seq >>> 0;
      if (
        this.ws &&
        this.ws.readyState === WebSocket.OPEN &&
        this.ws.protocol === 'pixelbattle.v2'
      ) {
        const buf = new ArrayBuffer(20);
        const v = new DataView(buf);
        const u8 = new Uint8Array(buf);
        u8[0] = 1;
        v.setUint16(1, xn, true);
        v.setUint16(3, yn, true);
        u8[5] = color[0];
        u8[6] = color[1];
        u8[7] = color[2];
        v.setUint32(8, seq32, true);
        v.setBigInt64(12, BigInt(Date.now()), true);
        this.ws.send(buf);
        return;
      }
      const pixel = {
        x: xn,
        y: yn,
        color: [color[0], color[1], color[2]],
        client_sent_ms: Date.now(),
        client_seq: seq32,
      };
      this.ws.send(JSON.stringify(pixel));
    },
    sendPixel(x, y, color) {
      // console.log(this.isGod);
      if (this.isGod=="true") {
        this.applyOptimisticAndSend(x, y, color);
        return;
      }
      if (!this.timerRunning) {
        this.timerRunning = true;
        this.applyOptimisticAndSend(x, y, color);
        
        this.seconds = 1;
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
    connectToWebSocket(endpoint, replayAfterMs) {
      const url = new URL(endpoint, location.href);
      url.protocol = 'wss';
      if (replayAfterMs != null && replayAfterMs > 0) {
        url.searchParams.set('replay_after_ms', String(replayAfterMs));
      }
      this.ws = new WebSocket(url, ['pixelbattle.v2', 'pixelbattle.v1']);
      this.ws.addEventListener('message', (event) => {this.handleNewPixel(event)});
    },
    /** Backend JSON uses x, y, color (Go json tags); tolerate legacy X, Y, Color. */
    applyRemotePixel(pixel) {
      const x = pixel.x ?? pixel.X;
      const y = pixel.y ?? pixel.Y;
      const c = pixel.color ?? pixel.Color;
      if (x == null || y == null || !Array.isArray(c) || c.length < 3) {
        return;
      }
      this.place.setPixel(
        x,
        y,
        new Uint8Array([Number(c[0]), Number(c[1]), Number(c[2])]),
      );
    },
    reconcileAndApplyPixel(pixel) {
      const serverRecvMs = pixel.server_recv_ms ?? pixel.serverRecvMs;
      if (serverRecvMs) {
        this.rum?.recordWSRenderLatency(serverRecvMs);
      }
      if (!this.loaded) {
        this.savedPixels.push(pixel);
        return;
      }
      const x = pixel.x ?? pixel.X;
      const y = pixel.y ?? pixel.Y;
      const seq = pixel.client_seq ?? pixel.clientSeq ?? 0;
      const c = pixel.color ?? pixel.Color;
      if (x == null || y == null || !Array.isArray(c) || c.length < 3) {
        return;
      }
      const key = `${x},${y}`;
      const pend = this.pendingOptimistic[key];
      const nc = [Number(c[0]), Number(c[1]), Number(c[2])];

      if (pend && seq === pend.seq) {
        if (this.colorsMatch(nc, pend.color)) {
          delete this.pendingOptimistic[key];
          return;
        }
        this.rum?.recordOptimisticCorrection('color_mismatch');
        delete this.pendingOptimistic[key];
        this.place.setPixel(x, y, new Uint8Array(nc));
        this.glWindow.draw();
        return;
      }
      if (pend && (seq === 0 || seq !== pend.seq)) {
        if (!this.colorsMatch(nc, pend.color)) {
          this.rum?.recordOptimisticCorrection(
            seq === 0 ? 'superseded_no_seq' : 'superseded',
          );
        }
        delete this.pendingOptimistic[key];
      }
      this.applyRemotePixel(pixel);
    },
    applyBinaryPixelV2(u8) {
      if (u8[0] !== 1 || (u8.length !== 20 && u8.length !== 16)) {
        this.rum?.errors?.push('ws_bad_binary');
        return;
      }
      const v = new DataView(u8.buffer, u8.byteOffset, u8.byteLength);
      const x = v.getUint16(1, true);
      const y = v.getUint16(3, true);
      let clientSeq = 0;
      let serverRecvMs;
      if (u8.length === 20) {
        clientSeq = v.getUint32(8, true);
        serverRecvMs = Number(v.getBigInt64(12, true));
      } else {
        serverRecvMs = Number(v.getBigInt64(8, true));
      }
      const pixel = {
        x,
        y,
        color: [u8[5], u8[6], u8[7]],
        server_recv_ms: serverRecvMs,
        client_seq: clientSeq,
      };
      this.reconcileAndApplyPixel(pixel);
    },
    handleNewPixel(event) {
      const d = event.data;
      if (d instanceof ArrayBuffer) {
        this.applyBinaryPixelV2(new Uint8Array(d));
        return;
      }
      if (typeof Blob !== 'undefined' && d instanceof Blob) {
        d.arrayBuffer().then((buf) => this.applyBinaryPixelV2(new Uint8Array(buf)));
        return;
      }
      if (typeof d !== 'string') {
        return;
      }
      let pixel;
      try {
        pixel = JSON.parse(d);
      } catch {
        this.rum?.errors?.push('ws_non_json');
        return;
      }
      if (pixel && pixel.event === 'RESIZE') {
        const nw = Number(pixel.width);
        const nh = Number(pixel.height);
        if (Number.isFinite(nw) && Number.isFinite(nh) && this.glWindow && this.glWindow.expandTextureTo) {
          this.pendingOptimistic = Object.create(null);
          this.glWindow.expandTextureTo(nw, nh);
          this.glWindow.draw();
        }
        return;
      }
      this.reconcileAndApplyPixel(pixel);
    },
    renderSavedPIxels() {
      for (const pixel of this.savedPixels) {
        this.applyRemotePixel(pixel);
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
      const canvasXY = this.glWindow.fromClientXY(pos.x, pos.y);
      pos = this.glWindow.click(canvasXY);
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
      const canvasXY = this.glWindow.fromClientXY(pos.x, pos.y);
      this.color = this.glWindow.getColor(this.glWindow.click(canvasXY));
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
          const tp = this.glWindow.fromClientXY(ev.touches[0].clientX, ev.touches[0].clientY);
          this.pos = this.glWindow.click(tp);
          this.val_x = this.pos.x;
          this.val_y = this.pos.y;
        } catch { /* touch outside drawable area */ }
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
  transition: border 0.1s ease-in-out;
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
  text-wrap: nowrap;
  color: black;
  background-color: #E1DBCE;
  padding: 5px;
  width: 55px;
  height: 20px;
  line-height: 20px;
  text-align: center;
  border: 2px solid #134293;
  border-radius: 10px;
  top: 46px;
  left: 50%;
  transform: translateX(-50%);
  font-size: 16px;
}

#timer {
  position: absolute;
  opacity: 0;
  pointer-events: none;
  top: 90px;
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
  top: 46px;
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

/* Only “цветная книга мудрецов” in the parchment modal — not the top bar links */
.modal a {
  color: #134293;
  text-decoration: underline;
}

#ad a.ad-link {
  color: #fff !important;
  text-decoration: none;
}

#running-line {
  width: 100%;
  position: absolute;
  top: 0;
  left: 0;
  height: 30px;
  display: flex;
  background-color: #E1DBCE;
  border-bottom: 2px solid #134293;
  pointer-events: all;
  text-decoration: none;
}

#ad {
  background-color: #134293;
  width: min-content;
  display: flex;
  color: white;
  border-radius: 0 8px 8px 0;
  padding-inline: 8px;
  text-wrap: nowrap;
  gap: 12px;
  line-height: 30px;
  font-size: 15px;
  z-index: 2;
}

.ad-link {
  color: white;
  text-decoration: none;
  display: flex;
  align-items: center;
  gap: 4px;
  opacity: 0.9;
  transition: opacity 0.15s;
}

.ad-link:hover {
  opacity: 1;
}

#tales-array {
  position: relative;
  line-height: 30px;
  box-sizing: border-box;

  display: flex;
  gap: 50px;
}

.tales {
  box-sizing: border-box;
  display: flex;
  gap: 30px;
  flex-shrink: 0;
  margin: 0;
  margin-left: 20px;
  min-width: 100%;
  animation-name: marqueeLine;
  animation-duration: 5s;
  animation-timing-function: linear;
  animation-iteration-count: infinite;
  user-select: none;
  color: black;
}

@keyframes marqueeLine {
  from {
    transform: translateX(0);
  }
  
  to {
    transform: translateX(-100%);
  }
}
</style>