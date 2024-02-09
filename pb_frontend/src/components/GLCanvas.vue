<template>
  <div>
    <canvas id="viewport-canvas"></canvas>
    <div id="ui-wrapper">
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
      place: null,
      glWindow: null,
      colorField: null,
      colorSwatch: null,
      color: null,
      dragdown: null,
      touchID: null,
      touchScaling: null,
      lastMovePos: null,
      lastScalingDist: null,
      touchstartTime: null
    }
  },
  mounted() {
    this.initializeGLWindow();
    this.initializePlace();
    this.initializeGUI();
  },
  methods: {
    initializeGLWindow () {
      this.glWindow = new GLWindow(document.querySelector("#viewport-canvas"));
    },
    initializePlace() {
      this.place = new Place(this.glWindow);
    },
    initializeGUI() {

      const glWindow = new GLWindow(document.querySelector("#viewport-canvas"));
      const place = new Place(glWindow);

      this.color = new Uint8Array([0, 0, 0]);
      this.dragdown = false;
      this.touchID = 0;
      this.touchScaling = false;
      this.lastMovePos = {x: 0, y: 0};
      this.lastScalingDist = 0;

      this.colorField = document.querySelector("#color-field");
      this.colorSwatch = document.querySelector("#color-swatch");

      let cvs = document.querySelector("#viewport-canvas");

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
        let zoom = this.glWindow.getZoom();
        if (ev.deltaY > 0) {
          zoom /= 1.05;
        } else {
          zoom *= 1.05;
        }
        this.glWindow.setZoom(zoom);
        this.glWindow.draw();
      });

      // document.querySelector("#zoom-in").addEventListener("click", () => {
      //   zoomIn(1.2);
      // });
      //
      // document.querySelector("#zoom-out").addEventListener("click", () => {
      //   zoomOut(1.2);
      // });

      window.addEventListener("resize", ev => {
        this.glWindow.updateViewScale();
        this.glWindow.draw();
      });

      cvs.addEventListener("mousedown", (ev) => {
        switch (ev.button) {
          case 0:
            this.dragdown = true;
            this.lastMovePos = { x: ev.clientX, y: ev.clientY };
            break;
          case 1:
            pickColor({ x: ev.clientX, y: ev.clientY });
            break;
          case 2:
            if (ev.ctrlKey) {
              pickColor({ x: ev.clientX, y: ev.clientY });
            } else {
              drawPixel({ x: ev.clientX, y: ev.clientY }, color);
            }
        }
      });

      document.addEventListener("mouseup", (ev) => {
        this.dragdown = false;
        document.body.style.cursor = "auto";
      });

      document.addEventListener("mousemove", (ev) => {
        const movePos = { x: ev.clientX, y: ev.clientY };
        if (this.dragdown) {
          this.glWindow.move(movePos.x - this.lastMovePos.x, movePos.y - this.lastMovePos.y);
          this.glWindow.draw();
          document.body.style.cursor = "grab";
        }
        this.lastMovePos = movePos;
      });

      cvs.addEventListener("touchstart", (ev) => {
        let thisTouch = this.touchID;
        this.touchstartTime = (new Date()).getTime();
        this.lastMovePos = { x: ev.touches[0].clientX, y: ev.touches[0].clientY };
        if (ev.touches.length === 2) {
          this.touchScaling = true;
          this.lastScalingDist = null;
        }

        setTimeout(() => {
          if (thisTouch == this.touchID) {
            pickColor(this.lastMovePos);
            navigator.vibrate(200);
          }
        }, 350);
      });

      document.addEventListener("touchend", (ev) => {
        this.touchID++;
        let elapsed = (new Date()).getTime() - this.touchstartTime;
        if (elapsed < 100) {
          if (drawPixel(this.lastMovePos, this.color)) {
            navigator.vibrate(10);
          };
        }
        if (ev.touches.length === 0) {
          this.touchScaling = false;
        }
      });

      document.addEventListener("touchmove", (ev) => {
        this.touchID++;
        if (this.touchScaling) {
          let dist = Math.hypot(
              ev.touches[0].pageX - ev.touches[1].pageX,
              ev.touches[0].pageY - ev.touches[1].pageY);
          if (this.lastScalingDist != null) {
            let delta = this.lastScalingDist - dist;
            if (delta < 0) {
              zoomIn(1 + Math.abs(delta) * 0.003);
            } else {
              zoomOut(1 + Math.abs(delta) * 0.003);
            }
          }
          this.lastScalingDist = dist;
        } else {
          let movePos = { x: ev.touches[0].clientX, y: ev.touches[0].clientY };
          this.glWindow.move(movePos.x - this.lastMovePos.x, movePos.y - this.lastMovePos.y);
          this.glWindow.draw();
          this.lastMovePos = movePos;
        }
      });

      cvs.addEventListener("contextmenu", () => { return false; });

      this.colorField.addEventListener("change", ev => {
        let hex = this.colorField.value.replace(/[^A-Fa-f0-9]/g, "").toUpperCase();
        hex = hex.substring(0, 6);
        while (hex.length < 6) {
          hex += "0";
        }
        this.color[0] = parseInt(hex.substring(0, 2), 16);
        this.color[1] = parseInt(hex.substring(2, 4), 16);
        this.color[2] = parseInt(hex.substring(4, 6), 16);
        hex = "#" + hex;
        this.colorField.value = hex;
        this.colorSwatch.style.backgroundColor = hex;
      });

      // ***************************************************
      // ***************************************************
      // Helper Functions
      //
      const pickColor = (pos) => {
        this.color = this.glWindow.getColor(this.glWindow.click(pos));
        let hex = "#";
        for (let i = 0; i < this.color.length; i++) {
          let d = this.color[i].toString(16);
          if (d.length == 1) d = "0" + d;
          hex += d;
        }
        this.colorField.value = hex.toUpperCase();
        this.colorSwatch.style.backgroundColor = hex;
      }

      const drawPixel = (pos, color) => {
        pos = this.glWindow.click(pos);
        if (pos) {
          const oldColor = this.glWindow.getColor(pos);
          for (let i = 0; i < oldColor.length; i++) {
            if (oldColor[i] != this.color[i]) {
              this.place.setPixel(pos.x, pos.y, color);
              return true;
            }
          }
        }
        return false;
      }

      const zoomIn = (factor) => {
        let zoom = this.glWindow.getZoom();
        this.glWindow.setZoom(zoom * factor);
        this.glWindow.draw();
      }

      const zoomOut = (factor) => {
        let zoom = this.glWindow.getZoom();
        this.glWindow.setZoom(zoom / factor);
        this.glWindow.draw();
      }
    }
  }
}
</script>

<style scoped>

</style>