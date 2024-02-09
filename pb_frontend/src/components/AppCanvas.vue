<template>
  <div>
    <div id="guide" ref="guide" v-show="showGuide"></div>
    <canvas :width="canvasWidth" :height="canvasHeight" id="canvas" ref="canvas"></canvas>
    <div>
      <label for="colorInput">Set Color: </label>
      <input type="color" id="colorInput" v-model="currentColor">
    </div>
<!--    <div>-->
<!--      <label for="toggleGuide">Show Guide: </label>-->
<!--      <input type="checkbox" id="toggleGuide" v-model="showGuide">-->
<!--    </div>-->
    <div>
      <button type="button" @click="clearCanvas">Clear</button>
    </div>
  </div>
</template>

<script>
export default {
  data() {
    return {
      currentColor: '#009578',
      showGuide: true,
      drawingContext: null,
      canvasWidth: 900,
      canvasHeight: 900,
      canvasSize: 100,
      colorHistory: {},
      ox: 0,
      oy: 0,
      px: 0,
      py: 0,
      scx: 1,
      scy: 1,

    };
  },
  mounted() {
    this.setupCanvas();
    this.setupGuide();
    this.draw();
  },
  methods: {
    setupCanvas() {
      const canvas = this.$refs.canvas;
      this.drawingContext = canvas.getContext("2d");
      this.drawingContext.fillStyle = "#ffffff";
      this.drawingContext.fillRect(0, 0, this.canvasWidth, this.canvasHeight);
      canvas.addEventListener("wheel", this.handleWheel);
      canvas.addEventListener("dblclick", this.handleCanvasDblclick);
      canvas.addEventListener("mousedown", this.handleCanvasMouseDown);
      canvas.addEventListener("mouseup", this.handleCanvasMouseUp);
    },
    setupGuide() {
      const guide = this.$refs.guide;
      guide.style.width = `${this.canvasWidth}px`;
      guide.style.height = `${this.canvasHeight}px`;
      guide.style.gridTemplateColumns = `repeat(${this.canvasSize}, 1fr)`;
      guide.style.gridTemplateRows = `repeat(${this.canvasSize}, 1fr)`;
      for (let i = 0; i < this.canvasSize ** 2; i++) {
        const pixel = document.createElement('div');
        guide.appendChild(pixel);
      }
    },
    handleWheel(e) {

    },
    handleCanvasDblclick(e) {
      if (e.button !== 0) {
        return;
      }
      const canvasBoundingRect = this.$refs.canvas.getBoundingClientRect();
      const x = e.clientX - canvasBoundingRect.left;
      const y = e.clientY - canvasBoundingRect.top;
      const cellX = Math.floor(x / (this.canvasWidth / this.canvasSize));
      const cellY = Math.floor(y / (this.canvasHeight / this.canvasSize));
      const currentColor = this.colorHistory[`${cellX}_${cellY}`];

      if (e.ctrlKey) {
        if (currentColor) {
          this.currentColor = currentColor;
        }
      } else {
        this.fillCell(cellX, cellY);
      }
    },
    handleCanvasMouseDown(e) {

    },
    handleCanvasMouseUp(e) {
      const canvas = this.$refs.canvas;
      canvas.onmousemove = null;
    },
    draw() {
      const ctx = this.drawingContext;
      ctx.clearRect(0, 0, ctx.canvas.width, ctx.canvas.height);

      // Your specific canvas rendering logic
      for (let i = 0; i < this.canvasSize; i++) {
        for (let j = 0; j < this.canvasSize; j++) {
          const currentColor = this.colorHistory[`${i}_${j}`];
          if (currentColor) {
            const startX = i * (this.canvasWidth / this.canvasSize);
            const startY = j * (this.canvasHeight / this.canvasSize);

            ctx.fillStyle = currentColor;
            ctx.fillRect(startX, startY, (this.canvasWidth / this.canvasSize), (this.canvasHeight / this.canvasSize));
          }
        }
      }
      // Call requestAnimationFrame to keep the rendering loop going
      requestAnimationFrame(this.draw);
    },
    WtoS(wx, wy) {
      const canvasBoundingRect = this.$refs.canvas.getBoundingClientRect();
      const sx = (wx - this.ox) * this.scx + canvasBoundingRect.left;
      const sy = (wy - this.oy) * this.scy + canvasBoundingRect.top;
      return [sx, sy];
    },

// Modify the StoW function to accommodate your canvas coordinate system
    StoW(sx, sy) {
      const canvasBoundingRect = this.$refs.canvas.getBoundingClientRect();
      const wx = (sx - canvasBoundingRect.left) / this.scx + this.ox;
      const wy = (sy - canvasBoundingRect.top) / this.scy + this.oy;
      return [wx, wy];
    },
    clearCanvas() {
      const yes = confirm("Are you sure you wish to clear the canvas?");
      if (yes) {
        this.drawingContext.fillStyle = "#ffffff";
        this.drawingContext.fillRect(0, 0, this.canvasWidth, this.canvasHeight);
      }
    },
    fillCell(cellX, cellY) {
      const startX = cellX * (this.canvasWidth / this.canvasSize);
      const startY = cellY * (this.canvasHeight / this.canvasSize);

      this.drawingContext.fillStyle = this.currentColor;
      this.drawingContext.fillRect(startX, startY, (this.canvasWidth / this.canvasSize), (this.canvasHeight / this.canvasSize));
      this.colorHistory[`${cellX}_${cellY}`] = this.currentColor;
    }
  }
};
</script>

<style scoped>
body {
  background: #333333;
  color: #ffffff;
  font-family: sans-serif;
}

#canvas {
  cursor: pointer;
}

#guide {
  display: grid;
  pointer-events: none;
  position: absolute;
  border: 1px solid rgba(0, 0, 0, 0.1);
}

#guide div {
  border: 1px solid rgba(0, 0, 0, 0.1);
}

.pixel {
  border: 1px solid rgba(0, 0, 0, 0.1);
}

.pixel:hover {
  background-color: #ff0000; /* Set the hover background color */
  cursor: pointer; /* Change cursor on hover */
}

</style>
