const DEFAULT_SAMPLE_RATE = 0.05;
const FLUSH_INTERVAL_MS = 10000;

function randomSessionId() {
  return `rum-${Date.now()}-${Math.random().toString(36).slice(2, 10)}`;
}

function quantile(values, q) {
  if (!values.length) return 0;
  const sorted = [...values].sort((a, b) => a - b);
  const idx = Math.min(sorted.length - 1, Math.max(0, Math.floor((sorted.length - 1) * q)));
  return sorted[idx];
}

export default class RUMCollector {
  constructor({ endpoint = '/api/rum', configEndpoint = '/api/config' } = {}) {
    this.endpoint = endpoint;
    this.configEndpoint = configEndpoint;
    this.session = randomSessionId();
    this.version = process.env.VUE_APP_GIT_SHA || process.env.NODE_ENV || 'unknown';
    this.sampleRate = DEFAULT_SAMPLE_RATE;
    this.enabled = false;

    this.frameDeltas = [];
    this.wsRenderLatencies = [];
    this.errors = [];
    this.webVitals = { lcp: 0, inp: 0, ttfb: 0 };
    this.lastFrameTs = 0;
    this.rafId = null;
    this.flushTimer = null;
    this.lcpObserver = null;
    this.inpObserver = null;
    this.boundOnError = (event) => this.errors.push(`error:${event.message || 'unknown'}`);
    this.boundOnReject = (event) => this.errors.push(`rejection:${event.reason || 'unknown'}`);
  }

  async start() {
    await this.loadSampleRate();
    this.enabled = Math.random() < this.sampleRate;
    if (!this.enabled) return;

    window.addEventListener('error', this.boundOnError);
    window.addEventListener('unhandledrejection', this.boundOnReject);
    this.captureInitialWebVitals();
    this.startFrameLoop();
    this.flushTimer = window.setInterval(() => this.flush(), FLUSH_INTERVAL_MS);
  }

  stop() {
    if (this.rafId) cancelAnimationFrame(this.rafId);
    if (this.flushTimer) clearInterval(this.flushTimer);
    if (this.lcpObserver) this.lcpObserver.disconnect();
    if (this.inpObserver) this.inpObserver.disconnect();
    window.removeEventListener('error', this.boundOnError);
    window.removeEventListener('unhandledrejection', this.boundOnReject);
    this.flush();
  }

  recordWSRenderLatency(serverRecvMs) {
    if (!this.enabled || !serverRecvMs) return;
    const now = Date.now();
    const delta = now - Number(serverRecvMs);
    if (delta >= 0 && delta < 30000) {
      this.wsRenderLatencies.push(delta);
    }
  }

  async loadSampleRate() {
    try {
      const resp = await fetch(this.configEndpoint, { credentials: 'same-origin' });
      if (!resp.ok) return;
      const cfg = await resp.json();
      const rate = Number(cfg.rum_sample_rate);
      if (!Number.isNaN(rate) && rate >= 0 && rate <= 1) {
        this.sampleRate = rate;
      }
    } catch {
      // keep default
    }
  }

  captureInitialWebVitals() {
    try {
      const nav = performance.getEntriesByType('navigation')[0];
      if (nav && typeof nav.responseStart === 'number' && typeof nav.requestStart === 'number') {
        this.webVitals.ttfb = Math.max(0, nav.responseStart - nav.requestStart);
      }
    } catch {
      // ignore
    }

    if (typeof PerformanceObserver === 'undefined') return;
    try {
      this.lcpObserver = new PerformanceObserver((list) => {
        const entries = list.getEntries();
        const last = entries[entries.length - 1];
        if (last) this.webVitals.lcp = Number(last.startTime) || 0;
      });
      this.lcpObserver.observe({ type: 'largest-contentful-paint', buffered: true });
    } catch {
      // unsupported
    }
    try {
      this.inpObserver = new PerformanceObserver((list) => {
        const entries = list.getEntries();
        for (const e of entries) {
          const latency = Number(e.duration) || 0;
          if (latency > this.webVitals.inp) this.webVitals.inp = latency;
        }
      });
      this.inpObserver.observe({ type: 'event', buffered: true, durationThreshold: 16 });
    } catch {
      // unsupported
    }
  }

  startFrameLoop() {
    const tick = (ts) => {
      if (this.lastFrameTs > 0) {
        const delta = ts - this.lastFrameTs;
        if (delta > 0 && delta < 1000) this.frameDeltas.push(delta);
      }
      this.lastFrameTs = ts;
      this.rafId = requestAnimationFrame(tick);
    };
    this.rafId = requestAnimationFrame(tick);
  }

  async flush() {
    if (!this.enabled) return;
    const frameCount = this.frameDeltas.length;
    const fpsAvg = frameCount ? 1000 / (this.frameDeltas.reduce((a, b) => a + b, 0) / frameCount) : 0;
    const payload = {
      session: this.session,
      version: this.version,
      fps_avg: Number(fpsAvg.toFixed(2)),
      fps_p10: Number((this.frameDeltas.length ? 1000 / quantile(this.frameDeltas, 0.9) : 0).toFixed(2)),
      frame_time_ms_p95: Number(quantile(this.frameDeltas, 0.95).toFixed(2)),
      lcp_ms: Number((this.webVitals.lcp || 0).toFixed(2)),
      inp_ms: Number((this.webVitals.inp || 0).toFixed(2)),
      ttfb_ms: Number((this.webVitals.ttfb || 0).toFixed(2)),
      ws_render_lat_ms_p95: Number(quantile(this.wsRenderLatencies, 0.95).toFixed(2)),
      errors: this.errors.slice(0, 20),
    };

    this.frameDeltas = [];
    this.wsRenderLatencies = [];
    this.errors = [];

    try {
      await fetch(this.endpoint, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'same-origin',
        body: JSON.stringify(payload),
      });
    } catch {
      // keep silent: RUM must not affect gameplay
    }
  }
}

