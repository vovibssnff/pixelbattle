<template>
  <div @contextmenu.prevent>
    <canvas
      id="viewport-canvas" @touchstart="onTouchStart"
      @mousedown="onMouseDown" @touchend="onTouchEnd"
      @mousemove="coordsUpdate" @contextmenu.prevent
    ></canvas>
    <div id="ui-wrapper" hide="true">
      <p id="loading-p"></p>

      <nav id="social-bar" aria-label="Social links">
        <a
          target="_blank"
          rel="noopener noreferrer"
          href="https://vk.ru/itmomegabattle"
          class="social-link"
        >
          <svg class="social-icon" viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg" aria-hidden="true"><path d="M12.77 19.15c-7.8 0-12.25-5.34-12.42-14.22h3.9c.12 6.5 3 9.26 5.27 9.83V4.93h3.68v5.61c2.24-.24 4.6-2.78 5.4-5.61h3.68a10.9 10.9 0 0 1-5.92 7.13 11.34 11.34 0 0 1 6.93 7.09h-4.06a7.16 7.16 0 0 0-6.03-5.06v5.06h-.43z"/></svg>
          <span class="social-label">VK</span>
        </a>
        <a
          target="_blank"
          rel="noopener noreferrer"
          href="https://t.me/itmomegabattle"
          class="social-link"
        >
          <svg class="social-icon" viewBox="0 0 1024 1024" xmlns="http://www.w3.org/2000/svg" aria-hidden="true"><path d="M417.28 795.733333l11.946667-180.48 327.68-295.253333c14.506667-13.226667-2.986667-19.626667-22.186667-8.106667L330.24 567.466667 155.306667 512c-37.546667-10.666667-37.973333-36.693333 8.533333-55.466667l681.386667-262.826666c31.146667-14.08 61.013333 7.68 49.066666 55.466666l-116.053333 546.56c-8.106667 38.826667-31.573333 48.213333-64 30.293334L537.6 695.466667l-84.906667 82.346666c-9.813333 9.813333-17.92 17.92-35.413333 17.92z"/></svg>
          <span class="social-label">Telegram</span>
        </a>
        <div id="cursor-info">
          <span id="x-coordinate">{{ Math.floor(val_x) }}</span>, <span id="y-coordinate">{{ Math.floor(val_y) }}</span>
        </div>
      </nav>

      <div id="running-line" aria-hidden="true">
        <div id="tales-array">
          <div ref="firstTales" class="tales"></div>
          <div ref="secondTales" class="tales"></div>
        </div>
      </div>

      <label id="howto" for="checkbox">?</label>
      <input id="checkbox" hidden type="checkbox">
      <label for="checkbox" class="modal-overlay">
        <div class="modal">
          <h1>О мудром владении цветами и пергаментом</h1>
          <div class="modal-body">
            <p><strong>На компьютере:</strong></p>
            <p>ПКМ — рисовать пиксель</p>
            <p>ЛКМ — двигать холст</p>
            <p>CTRL + ПКМ — скопировать цвет</p>
            <p>+/− — масштаб</p>
            <br/>
            <p><strong>На телефоне:</strong></p>
            <p>Касание — рисовать пиксель (режим «Перо»)</p>
            <p>Свайп — двигать холст</p>
            <p>Щипок — масштаб</p>
            <p>Долгое нажатие — скопировать цвет</p>
            <p>Переключайте режим «Перо» / «Рука» кнопкой внизу</p>
            <br/>
            <p><a target="_blank" href="https://color-hex.com">Цветная книга мудрецов</a> — в помощь для подбора цветов</p>
            <br/>
            <p>Время течет, как река, и ставить пиксели дозволено лишь раз в один удар сердца.</p>
            <p>Пусть же рука твоя будет тверда, а дух — неколебим!</p>
          </div>
          <label for="checkbox" class="modal-close-btn">✕</label>
        </div>
      </label>

      <button
        id="inspect-toggle"
        :class="{ active: inspectMode }"
        type="button"
        @click="toggleInspectMode"
      >
        {{ inspectMode ? '🔍 On' : '🔍' }}
      </button>
      <div v-if="pixelInfo" id="pixel-info-card">
        <div class="pixel-info-title">
          <span>Pixel {{ pixelInfo.x }}, {{ pixelInfo.y }}</span>
          <button type="button" class="pixel-info-close" @click="pixelInfo = null">×</button>
        </div>
        <div class="pixel-info-row">
          <span>Color</span>
          <span class="pixel-info-value">
            <span class="pixel-color-swatch" :style="{ backgroundColor: pixelInfoHex }"></span>
            {{ pixelInfoHex }}
          </span>
        </div>
        <div class="pixel-info-row">
          <span>Placed by</span>
          <button type="button" class="pixel-info-copy" @click="copyPixelUserID">
            {{ pixelPlacedBy }}
          </button>
        </div>
        <div class="pixel-info-row">
          <span>Faculty</span>
          <span class="pixel-info-value">{{ pixelInfo.faculty || '—' }}</span>
        </div>
        <div class="pixel-info-row">
          <span>Placed at</span>
          <span class="pixel-info-value">{{ pixelPlacedAt }}</span>
        </div>
        <div v-if="pixelInfoLoading" class="pixel-info-loading">Loading…</div>
      </div>

      <div id="bottom-bar">
        <div id="color-wrapper">
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
        </div>

        <div id="timer">
          {{ seconds }}
        </div>

        <button
          v-if="isTouchDevice"
          id="draw-mode-toggle"
          :class="{ active: drawMode }"
          type="button"
          @click="toggleDrawMode"
        >
          {{ drawMode ? '✏️' : '✋' }}
        </button>

        <div id="zoom-wrapper">
          <button id="zoom-out" class="zoom-button" @click="() => {zoomOut(1.2);}">−</button>
          <button id="zoom-in" class="zoom-button" @click="() => {zoomIn(1.2);}">+</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
import GLWindow from '@/webgl/glwindow.js'
import Place from '@/webgl/place.js'
import RUMCollector from '@/utils/rum.js'
import { runningLineTexts } from '@/data/runningLineTexts.js'

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
      lastCursorPos: null,
      lastScalingDist: null,
      timerRunning: false,
      cooldownSeconds: 1,
      seconds: 0,
      timer: null,
      secondTimer: null,
      timerValue: null,
      palette: [],
      activeSwatch: 0,
      loaded: false,
      savedPixels: [],
      isGod: null,
      isTouchDevice: false,
      inspectMode: false,
      pixelInfo: null,
      pixelInfoLoading: false,
      pixelInfoCacheReady: false,
      pixelInfoCache: Object.create(null),
      pixelInfoTimer: null,
      pixelInfoRequestID: 0,
      rum: null,
      clientSeq: 0,
      pendingOptimistic: Object.create(null),
      pendingOutboundPixels: [],
      drawMode: true,
      touchStartPos: null,
    }
  },
  computed: {
    pixelInfoHex() {
      const c = this.pixelInfo?.color;
      if (!Array.isArray(c) || c.length < 3) return '#000000';
      return '#' + c.slice(0, 3).map((v) => {
        const n = Math.max(0, Math.min(255, Number(v) || 0));
        return n.toString(16).padStart(2, '0');
      }).join('').toUpperCase();
    },
    pixelPlacedBy() {
      const uid = this.pixelInfo?.userid;
      return uid && String(uid).trim() ? uid : 'system';
    },
    pixelPlacedAt() {
      const ts = Number(this.pixelInfo?.timestamp);
      if (!Number.isFinite(ts) || ts <= 0) return '—';
      return new Date(ts * 1000).toLocaleString();
    },
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

    this.renderRunningLine();

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
    this.isTouchDevice = /(android|webos|iphone|ipad|ipod|blackberry|windows phone)/.test(platform) || window.matchMedia?.('(hover: none)').matches;
    // this.setSwatchesArr(this.$refs.swatches);
    // this.setField(document.querySelector("#color-field"));
    // window.alert("ПКМ - рисование, ЛКМ - навигация, CTRL+ПКМ - копирование цвета в палитру, https://www.color-hex.com/ - в помощь для подбора цветов");
  },
  beforeUnmount() {
    this.rum?.stop();
    clearTimeout(this.pixelInfoTimer);
  },
  methods: {
    renderRunningLine() {
      const talesArray = [...runningLineTexts].sort(() => Math.random() - 0.5);
      const firstTales = this.$refs.firstTales;
      const secondTales = this.$refs.secondTales;
      if (!firstTales || !secondTales) return;

      firstTales.innerHTML = "";
      secondTales.innerHTML = "";

      talesArray.forEach(tale => {
        const span = document.createElement("span");
        span.textContent = tale;
        firstTales.appendChild(span);
        secondTales.appendChild(span.cloneNode(true));
      });

      const totalLength = talesArray.join("").length;
      const duration = Math.max(60, totalLength / 10);
      firstTales.style.animationDuration = `${duration}s`;
      secondTales.style.animationDuration = `${duration}s`;
    },
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
        const contentType = resp.headers.get("Content-Type") || "";
        if (resp.redirected || !contentType.startsWith("image/")) {
          this.$router.push('/login');
          return 0;
        }
				let buf = await this.$data.place.downloadProgress(resp);
				await this.$data.place.setImage(buf);
        this.loaded = true;
        this.$data.place.loadingp.innerHTML = "";
        this.$data.place.uiwrapper.setAttribute("hide", true);
        this.$data.isGod = resp.headers.get("Is-God");
        if (this.$data.isGod === 'true') {
          this.loadPixelInfoCache();
        }
        const rawCooldown = resp.headers.get("X-Pixel-Cooldown-Sec");
        if (rawCooldown !== null) {
          const cooldown = Number(rawCooldown);
          if (Number.isFinite(cooldown) && cooldown >= 0) {
            this.cooldownSeconds = Math.floor(cooldown);
          }
        }
        const raw = resp.headers.get("X-Snapshot-Ms");
        const ms = raw ? parseInt(raw, 10) : 0;
        return Number.isFinite(ms) ? ms : 0;
			})
      .catch(() => {
        this.$router.push('/login');
        return 0;
      });
    },
    refetchCanvasAfterResize() {
      fetch("/api/canvas.png", { cache: "no-store" })
        .then(async resp => {
          const contentType = resp.headers.get("Content-Type") || "";
          if (!contentType.startsWith("image/")) return;
          const buf = await this.$data.place.downloadProgress(resp);
          await this.$data.place.setImage(buf);
          this.glWindow.draw();
        })
        .catch(() => {});
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
      this.lastCursorPos = { x: ev.clientX, y: ev.clientY };
      try {
        const p = this.glWindow.fromClientXY(ev.clientX, ev.clientY);
        this.pos = this.glWindow.click(p);
        this.val_x = this.pos.x;
        this.val_y = this.pos.y;
        if (this.inspectMode && !this.isTouchDevice) {
          this.schedulePixelInfo(this.pos.x, this.pos.y);
        }
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
      this.invalidatePixelInfo(xn, yn, c, true);
      this.place.setPixel(xn, yn, new Uint8Array(c));
      this.send(xn, yn, c, seq);
    },
    queueOutboundPixel(pixel) {
      this.pendingOutboundPixels.push(pixel);
      if (this.pendingOutboundPixels.length > 256) {
        this.pendingOutboundPixels.shift();
        this.rum?.errors?.push('ws_outbound_queue_overflow');
      }
    },
    flushOutboundPixels() {
      const pending = this.pendingOutboundPixels.splice(0);
      for (const pixel of pending) {
        this.send(pixel.x, pixel.y, pixel.color, pixel.seq, true);
      }
    },
    send(x, y, color, seq, fromQueue = false) {
      const xn = Math.floor(x);
      const yn = Math.floor(y);
      const seq32 = seq >>> 0;
      if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
        if (!fromQueue) {
          this.queueOutboundPixel({ x: xn, y: yn, color: [color[0], color[1], color[2]], seq: seq32 });
        }
        return false;
      }
      if (
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
        return true;
      }
      const pixel = {
        x: xn,
        y: yn,
        color: [color[0], color[1], color[2]],
        client_sent_ms: Date.now(),
        client_seq: seq32,
      };
      this.ws.send(JSON.stringify(pixel));
      return true;
    },
    sendPixel(x, y, color) {
      // console.log(this.isGod);
      if (this.isGod=="true") {
        this.applyOptimisticAndSend(x, y, color);
        return;
      }
      const cooldown = Math.max(0, Number(this.cooldownSeconds) || 0);
      if (cooldown === 0) {
        this.clearPlacementTimer();
        this.applyOptimisticAndSend(x, y, color);
        return;
      }
      if (!this.timerRunning) {
        this.timerRunning = true;
        this.applyOptimisticAndSend(x, y, color);
        
        this.seconds = cooldown;
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
    clearPlacementTimer() {
      clearInterval(this.timer);
      clearInterval(this.secondTimer);
      this.timerRunning = false;
      this.seconds = 0;
      if (this.timerValue) {
        this.timerValue.style.opacity = 0;
      }
    },
    connectToWebSocket(endpoint, replayAfterMs) {
      const url = new URL(endpoint, location.href);
      url.protocol = 'wss';
      if (replayAfterMs != null && replayAfterMs > 0) {
        url.searchParams.set('replay_after_ms', String(replayAfterMs));
      }
      this.ws = new WebSocket(url, ['pixelbattle.v2', 'pixelbattle.v1']);
      this.ws.addEventListener('open', () => {
        this.connected = true;
        this.flushOutboundPixels();
      });
      this.ws.addEventListener('close', () => {
        this.connected = false;
      });
      this.ws.addEventListener('error', () => {
        this.connected = false;
        this.rum?.errors?.push('ws_error');
      });
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
      this.invalidatePixelInfo(x, y, [Number(c[0]), Number(c[1]), Number(c[2])], true);
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
          this.invalidatePixelInfo(x, y, nc, true);
          return;
        }
        this.rum?.recordOptimisticCorrection('color_mismatch');
        delete this.pendingOptimistic[key];
        this.place.setPixel(x, y, new Uint8Array(nc));
        this.invalidatePixelInfo(x, y, nc, true);
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
          this.refetchCanvasAfterResize();
        }
        return;
      }
      if (pixel && pixel.event === 'COOLDOWN') {
        const seconds = Number(pixel.seconds);
        if (Number.isFinite(seconds) && seconds >= 0) {
          this.cooldownSeconds = Math.floor(seconds);
          if (this.cooldownSeconds === 0) {
            this.clearPlacementTimer();
          }
        }
        return;
      }
      if (pixel && pixel.event === 'PIXEL_INFO') {
        const info = this.normalizePixelInfo(pixel.pixel);
        if (info) {
          const key = this.pixelInfoKey(info.x, info.y);
          this.pixelInfoCache[key] = info;
          if (this.pixelInfo && this.pixelInfoKey(this.pixelInfo.x, this.pixelInfo.y) === key) {
            this.pixelInfo = info;
          }
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
    toggleInspectMode() {
      this.inspectMode = !this.inspectMode;
      if (!this.inspectMode) {
        this.pixelInfo = null;
        return;
      }
      if (!this.isTouchDevice && this.lastCursorPos) {
        this.inspectPixelAt(this.lastCursorPos);
      }
    },
    schedulePixelInfo(x, y) {
      const xn = Math.floor(x);
      const yn = Math.floor(y);
      clearTimeout(this.pixelInfoTimer);
      this.pixelInfoTimer = setTimeout(() => {
        this.fetchPixelInfo(xn, yn);
      }, 250);
    },
    inspectPixelAt(pos) {
      try {
        const canvasXY = this.glWindow.fromClientXY(pos.x, pos.y);
        const p = this.glWindow.click(canvasXY);
        this.pos = p;
        this.val_x = p.x;
        this.val_y = p.y;
        this.fetchPixelInfo(p.x, p.y);
      } catch { /* outside canvas */ }
    },
    pixelInfoKey(x, y) {
      return `${Math.floor(x)},${Math.floor(y)}`;
    },
    normalizePixelInfo(pixel) {
      if (!pixel) return null;
      const x = pixel.x ?? pixel.X;
      const y = pixel.y ?? pixel.Y;
      const color = pixel.color ?? pixel.Color;
      if (x == null || y == null || !Array.isArray(color) || color.length < 3) {
        return null;
      }
      return {
        x: Math.floor(Number(x)),
        y: Math.floor(Number(y)),
        color: [Number(color[0]), Number(color[1]), Number(color[2])],
        userid: pixel.userid ?? pixel.UserID ?? '',
        faculty: pixel.faculty ?? pixel.Faculty ?? '',
        timestamp: pixel.timestamp ?? pixel.Timestamp ?? 0,
      };
    },
    async loadPixelInfoCache() {
      if (this.isGod !== 'true') return;
      try {
        const res = await fetch('/api/admin/pixels/cache', {
          credentials: 'same-origin',
          cache: 'no-store',
        });
        if (!res.ok) return;
        const body = await res.json();
        const pixels = Array.isArray(body?.pixels) ? body.pixels : [];
        const nextCache = Object.create(null);
        for (const raw of pixels) {
          const pixel = this.normalizePixelInfo(raw);
          if (pixel) {
            nextCache[this.pixelInfoKey(pixel.x, pixel.y)] = pixel;
          }
        }
        this.pixelInfoCache = nextCache;
        this.pixelInfoCacheReady = true;
        if (this.pixelInfo) {
          const key = this.pixelInfoKey(this.pixelInfo.x, this.pixelInfo.y);
          this.pixelInfo = this.pixelInfoCache[key] || this.pixelInfo;
        }
      } catch { /* per-pixel fetch remains as fallback */ }
    },
    async fetchPixelInfo(x, y) {
      const xn = Math.floor(x);
      const yn = Math.floor(y);
      const key = this.pixelInfoKey(xn, yn);
      if (this.pixelInfoCache[key]) {
        this.pixelInfo = this.pixelInfoCache[key];
        return;
      }
      const reqID = ++this.pixelInfoRequestID;
      this.pixelInfoLoading = true;
      try {
        const res = await fetch(`/api/pixels/info?x=${encodeURIComponent(xn)}&y=${encodeURIComponent(yn)}`, {
          credentials: 'same-origin',
          cache: 'no-store',
        });
        if (!res.ok) {
          return;
        }
        const body = await res.json();
        if (reqID !== this.pixelInfoRequestID) return;
        const pixel = this.normalizePixelInfo(body?.pixel);
        if (pixel) {
          this.pixelInfoCache[key] = pixel;
          this.pixelInfo = pixel;
        }
      } catch {
        // Keep the previous card visible; the next hover or cache refresh can recover.
      } finally {
        if (reqID === this.pixelInfoRequestID) {
          this.pixelInfoLoading = false;
        }
      }
    },
    invalidatePixelInfo(x, y, color = null, revalidate = false) {
      const key = this.pixelInfoKey(x, y);
      const visible = this.pixelInfo && this.pixelInfoKey(this.pixelInfo.x, this.pixelInfo.y) === key;
      const previous = this.pixelInfoCache[key] || (visible ? this.pixelInfo : null);
      if (previous && color) {
        if (this.colorsMatch(previous.color, color) && !revalidate) {
          if (visible) {
            this.pixelInfo = previous;
          }
          return;
        }
        const updated = {
          ...previous,
          color: [Number(color[0]), Number(color[1]), Number(color[2])],
        };
        if (visible) {
          this.pixelInfo = updated;
        }
      }
      delete this.pixelInfoCache[key];
      if (revalidate && visible) {
        clearTimeout(this.pixelInfoTimer);
        this.pixelInfoTimer = setTimeout(() => {
          this.fetchPixelInfo(x, y);
        }, 150);
      }
    },
    copyPixelUserID() {
      const uid = this.pixelInfo?.userid;
      if (!uid || !navigator.clipboard) return;
      navigator.clipboard.writeText(uid);
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
    toggleDrawMode() {
      this.drawMode = !this.drawMode;
    },
    onTouchMove(ev) {
      this.touchID++;
      if (this.touchScaling && ev.touches.length !== 1) {
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
        if (!this.drawMode) {
          this.glWindow.move(movePos.x - this.lastMovePos.x, movePos.y - this.lastMovePos.y);
          this.glWindow.draw();
        }
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
    },
    onTouchStart(ev) {
      ev.preventDefault();
      let thisTouch = this.touchID;
      this.touchstartTime = Date.now();
      this.touchStartPos = { x: ev.touches[0].clientX, y: ev.touches[0].clientY };
      this.lastMovePos = { x: ev.touches[0].clientX, y: ev.touches[0].clientY };
      if (ev.touches.length === 2) {
        this.touchScaling = true;
        this.lastScalingDist = null;
      }
      setTimeout(() => {
        if (!this.inspectMode && thisTouch === this.$data.touchID) {
          this.pickColor(this.lastMovePos);
          if (navigator.vibrate) navigator.vibrate(30);
        }
      }, 400);
    },
    onTouchEnd(ev) {
      this.$data.touchID++;
      const elapsed = Date.now() - this.touchstartTime;
      const dx = this.lastMovePos.x - this.touchStartPos.x;
      const dy = this.lastMovePos.y - this.touchStartPos.y;
      const movedDist = Math.sqrt(dx * dx + dy * dy);
      const isTap = elapsed < 300 && movedDist < 12;

      if (isTap) {
        if (this.inspectMode) {
          this.inspectPixelAt(this.lastMovePos);
        } else if (this.drawMode) {
          this.drawPixel(this.lastMovePos, this.color);
        }
      }
      if (ev.touches.length === 0) {
        this.touchScaling = false;
      }
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
  touch-action: none;
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
}

#ui-wrapper #bottom-bar {
  visibility: hidden;
}

#ui-wrapper[hide=true] {
  pointer-events: none;
  background: none;
}

#ui-wrapper[hide=true] #bottom-bar {
  visibility: visible;
}

#ui-wrapper[hide=true] #social-bar,
#ui-wrapper[hide=true] #running-line,
#ui-wrapper[hide=true] #howto,
#ui-wrapper[hide=true] #inspect-toggle,
#ui-wrapper[hide=true] #pixel-info-card,
#ui-wrapper[hide=true] #cursor-info {
  pointer-events: auto;
}

/* ── Top bar: social links + coordinates ── */
#social-bar {
  position: absolute;
  top: 0;
  left: 0;
  width: 100%;
  height: 36px;
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 0 12px;
  padding-top: env(safe-area-inset-top, 0px);
  box-sizing: border-box;
  background-color: #134293;
  border-bottom: 2px solid #0f3270;
  pointer-events: all;
  z-index: 11;
}

.social-link {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  color: #fff;
  text-decoration: none;
  font-family: system-ui, -apple-system, "Segoe UI", sans-serif;
  font-size: 14px;
  font-weight: 600;
  line-height: 1;
  white-space: nowrap;
  opacity: 0.95;
  transition: opacity 0.15s;
  min-height: 36px;
}

.social-link:hover,
.social-link:focus {
  opacity: 1;
  outline: none;
}

.social-icon {
  width: 20px;
  height: 20px;
  flex-shrink: 0;
  fill: currentColor;
}

.social-label {
  display: inline;
}

#cursor-info {
  margin-left: auto;
  text-wrap: nowrap;
  color: #fff;
  font-size: 13px;
  font-family: monospace;
  opacity: 0.85;
}

/* ── Running line (marquee) ── */
#running-line {
  width: 100%;
  position: absolute;
  top: 36px;
  left: 0;
  height: 28px;
  display: flex;
  align-items: stretch;
  background-color: #E1DBCE;
  border-bottom: 2px solid #134293;
  pointer-events: all;
  z-index: 10;
  overflow: hidden;
}

#tales-array {
  position: relative;
  line-height: 28px;
  box-sizing: border-box;
  flex: 1 1 auto;
  min-width: 0;
  overflow: hidden;
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
  font-size: 14px;
}

@keyframes marqueeLine {
  from { transform: translateX(0); }
  to   { transform: translateX(-100%); }
}

/* ── Help button ── */
#howto {
  position: absolute;
  top: 72px;
  right: 12px;
  font-size: 18px;
  text-align: center;
  border-radius: 12px;
  background-color: #E1DBCE;
  border: 2px solid #134293;
  line-height: 44px;
  width: 44px;
  height: 44px;
  cursor: help;
  user-select: none;
  z-index: 1;
  pointer-events: all;
}

/* ── Inspect toggle ── */
#inspect-toggle {
  position: absolute;
  top: 72px;
  left: 12px;
  min-width: 44px;
  min-height: 44px;
  padding: 8px 14px;
  color: #134293;
  background-color: #E1DBCE;
  border: 2px solid #134293;
  border-radius: 12px;
  cursor: pointer;
  pointer-events: all;
  user-select: none;
  font-size: 16px;
  transition: background-color 0.15s ease-in-out, transform 0.15s ease-in-out;
}

#inspect-toggle:hover,
#inspect-toggle:focus {
  background-color: #f3eadb;
  outline: none;
}

#inspect-toggle:active {
  transform: scale(0.96);
}

#inspect-toggle.active {
  color: #ffffff;
  background-color: #134293;
}

/* ── Pixel info card ── */
#pixel-info-card {
  position: absolute;
  top: 124px;
  left: 12px;
  width: 280px;
  padding: 14px;
  color: black;
  background-color: #E1DBCE;
  border: 2px solid #134293;
  border-radius: 12px;
  box-sizing: border-box;
  pointer-events: all;
  z-index: 3;
  box-shadow: 0 8px 24px rgba(19, 66, 147, 0.25);
  font-size: 14px;
}

.pixel-info-title,
.pixel-info-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}

.pixel-info-title {
  margin-bottom: 8px;
  color: #134293;
  font-size: 18px;
}

.pixel-info-row {
  min-height: 28px;
}

.pixel-info-value,
.pixel-info-copy {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  color: #134293;
  font-family: monospace;
}

.pixel-info-copy,
.pixel-info-close {
  min-height: 44px;
  border: none;
  background: transparent;
  cursor: pointer;
  pointer-events: all;
}

.pixel-info-copy:hover,
.pixel-info-copy:focus,
.pixel-info-close:hover,
.pixel-info-close:focus {
  text-decoration: underline;
  outline: none;
}

.pixel-info-close {
  width: 44px;
  color: #134293;
  font-size: 20px;
}

.pixel-color-swatch {
  width: 16px;
  height: 16px;
  border: 1px solid #134293;
  border-radius: 4px;
}

.pixel-info-loading {
  margin-top: 8px;
  color: #134293;
}

/* ── Bottom bar (palette + timer + mode + zoom) ── */
#bottom-bar {
  position: absolute;
  bottom: 0;
  left: 0;
  right: 0;
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  padding-bottom: calc(8px + env(safe-area-inset-bottom, 0px));
  background: linear-gradient(transparent, rgba(225, 219, 206, 0.85) 30%);
  pointer-events: all;
  z-index: 5;
}

#color-wrapper {
  display: flex;
  flex-direction: row;
  flex-shrink: 1;
  min-width: 0;
  border: 2px solid #134293;
  background-color: #E1DBCE;
  border-radius: 12px;
  overflow: hidden;
}

.color-swatch {
  width: 36px;
  height: 40px;
  background-color: #000000;
  border-width: 0;
  border-style: solid;
  border-color: #134293;
  cursor: pointer;
  pointer-events: all;
  box-sizing: border-box;
  transition: border 0.1s ease-in-out;
  flex-shrink: 0;
}

.color-swatch:first-of-type {
  border-width: 2px 3px;
  width: 37px;
}

#color-field {
  font-size: 16px;
  color: black;
  height: 40px;
  line-height: 40px;
  box-sizing: border-box;
  margin-left: 4px;
  border: none;
  outline: none;
  pointer-events: all;
  background-color: transparent;
  width: 8ch;
  flex-shrink: 0;
}

#timer {
  opacity: 0;
  pointer-events: none;
  min-width: 28px;
  padding: 4px 6px;
  color: #134293;
  border-radius: 10px;
  text-align: center;
  background-color: #E1DBCE;
  border: 2px solid #134293;
  font-size: 16px;
  transition: all 0.2s ease-in-out;
  flex-shrink: 0;
}

/* ── Draw / Pan mode toggle ── */
#draw-mode-toggle {
  flex-shrink: 0;
  width: 48px;
  height: 44px;
  border: 2px solid #134293;
  border-radius: 12px;
  background-color: #E1DBCE;
  color: #134293;
  font-size: 22px;
  cursor: pointer;
  pointer-events: all;
  user-select: none;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: background-color 0.15s, transform 0.1s;
}

#draw-mode-toggle.active {
  background-color: #134293;
  border-color: #0f3270;
}

#draw-mode-toggle:active {
  transform: scale(0.94);
}

/* ── Zoom ── */
#zoom-wrapper {
  flex-shrink: 0;
  margin-left: auto;
  border-radius: 12px;
  border: 2px solid #134293;
  background-color: #E1DBCE;
  display: flex;
}

.zoom-button {
  color: #134293;
  font-family: monospace;
  width: 44px;
  height: 44px;
  border: none;
  background: none;
  outline: none;
  font-size: 24px;
  cursor: pointer;
  pointer-events: all;
  user-select: none;
}

.zoom-button:first-of-type {
  border-right: 2px solid #134293;
}

/* ── Loading ── */
#loading-p {
  position: absolute;
  top: 50%;
  left: 50%;
  font-size: 1.4em;
  transform: translate(-50%, -50%);
}

/* ── Help modal ── */
.modal-overlay {
  position: fixed;
  top: 0;
  right: 0;
  bottom: 0;
  left: 0;
  background-color: rgba(0, 0, 0, 0.5);
  opacity: 0;
  pointer-events: none;
  transition: opacity 0.15s ease-in-out;
  z-index: 100;
  display: flex;
  align-items: center;
  justify-content: center;
}

.modal {
  font-size: 16px;
  border-radius: 14px;
  display: flex;
  flex-direction: column;
  padding: 32px 24px;
  box-sizing: border-box;
  width: 92%;
  max-width: 520px;
  max-height: 85vh;
  overflow-y: auto;
  -webkit-overflow-scrolling: touch;
  background-image: url("@/img/paper-bg.jpg");
  background-size: cover;
  border: 2px solid #134293;
  text-align: center;
  position: relative;
  transform: translateY(8px);
  transition: transform 0.3s ease-in-out;
}

.modal-body {
  text-align: left;
  margin-top: 16px;
  line-height: 1.5;
}

.modal-body p {
  margin: 2px 0;
}

.modal-close-btn {
  position: absolute;
  right: 8px;
  top: 8px;
  width: 44px;
  height: 44px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #134293;
  font-size: 20px;
  cursor: pointer;
}

input:checked ~ .modal-overlay {
  display: flex;
  opacity: 1;
  pointer-events: auto;
}

input:checked ~ .modal-overlay .modal {
  transform: translateY(0);
}

h1, h2, h3 {
  color: #134293;
}

h1 {
  font-size: 20px;
  padding-right: 36px;
}

.modal a {
  color: #134293;
  text-decoration: underline;
}

/* ── Mobile: <=600px ── */
@media (max-width: 600px) {
  .social-label {
    display: none;
  }

  #social-bar {
    height: 32px;
    gap: 10px;
    padding: 0 8px;
    padding-top: env(safe-area-inset-top, 0px);
  }

  #running-line {
    top: 32px;
    height: 24px;
  }

  #tales-array {
    line-height: 24px;
  }

  .tales {
    font-size: 12px;
  }

  #howto {
    top: 62px;
    right: 8px;
  }

  #inspect-toggle {
    top: 62px;
    left: 8px;
    padding: 6px 10px;
    font-size: 14px;
  }

  #pixel-info-card {
    top: auto;
    right: 8px;
    bottom: 72px;
    left: 8px;
    width: auto;
  }

  .modal {
    font-size: 14px;
    padding: 24px 16px;
    width: 96%;
    max-height: 80vh;
  }

  h1 {
    font-size: 17px;
  }

  #bottom-bar {
    gap: 6px;
    padding: 6px 8px;
    padding-bottom: calc(6px + env(safe-area-inset-bottom, 0px));
  }

  .color-swatch {
    width: 32px;
    height: 36px;
  }

  .color-swatch:first-of-type {
    width: 33px;
  }

  #color-field {
    width: 7ch;
    font-size: 14px;
    height: 36px;
    line-height: 36px;
  }

  .zoom-button {
    width: 40px;
    height: 40px;
    font-size: 22px;
  }

  #draw-mode-toggle {
    width: 44px;
    height: 40px;
    font-size: 20px;
  }
}

/* ── Very small screens ── */
@media (max-width: 380px) {
  #color-field {
    display: none;
  }

  .color-swatch {
    width: 28px;
    height: 34px;
  }

  .color-swatch:first-of-type {
    width: 29px;
  }
}
</style>
