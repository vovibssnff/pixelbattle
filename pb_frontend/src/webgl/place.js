export default class Place {
    loaded;
    socket;
    loadingp;
    uiwrapper;
    glWindow;
    allowDraw;

    constructor(glWindow) {
        this.loaded = false;
        this.socket = null;
        this.loadingp = null;
        this.uiwrapper = null;
        this.glWindow = glWindow;
        this.allowDraw = true;
    }

    initConnection(endpoint) {
        fetch(endpoint)
			.then(async resp => {
				let buf = await this.downloadProgress(resp);
				await this.setImage(buf);
			});
    }

    async downloadProgress(resp) {
        let len = resp.headers.get("Content-Length");
		let a = new Uint8Array(len);
		let pos = 0;
		let reader = resp.body.getReader();
		while (true) {
			let { done, value } = await reader.read();
			if (value) {
				a.set(value, pos);
				pos += value.length;
			}
			if (done) break;
		}
		return a;
    }

    connect(path) {
        this.socket = new WebSocket(path);
        const socketMessage = async (event) => {
            let b = await event.data.arrayBuffer();
            if (this.allowDraw == null) {
                let view = new DataView(b);
                this.allowDraw = view.getUint8(0) === 1;
                if (!this.allowDraw) {
                    this.keyPrompt();
                }
            } else {
                this.handleSocketSetPixel(b);
            }
        };

        const socketClose = (event) => {
            this.socket = null;
        };

        const socketError = (event) => {
            console.error("Error making WebSocket connection.");
            alert("Failed to connect.");
            this.socket.close();
        };

        this.socket.addEventListener("message", socketMessage);
        this.socket.addEventListener("close", socketClose);
        this.socket.addEventListener("error", socketError);
    }

    prepare(color) {
        let b = new Uint8Array(11);
        for (let i = 0; i < 3; i++) {
            b[8 + i] = color[i];
        }
        return b;
    }

    setPixel(x, y, color, higlightMode) {
        let b = new Uint8Array(11);
        this.putUint32(b.buffer, 0, x);
        this.putUint32(b.buffer, 4, y);
        for (let i = 0; i < 3; i++) {
            b[8 + i] = color[i];
        }
        if (higlightMode==true) {
            console.log("here");
            this.glWindow.setHiglighting(x, y, color);
        } else {
            this.glWindow.setPixelColor(x, y, color);
        }
        this.glWindow.draw();
    }

    handleSocketSetPixel(b) {
        if (this.loaded) {
            let x = this.getUint32(b, 0);
            let y = this.getUint32(b, 4);
            let color = new Uint8Array(b.slice(8));
            this.glWindow.setPixelColor(x, y, color);
            this.glWindow.draw();
        }
    }

    async setImage(data) {
        let img = new Image();
        let blob = new Blob([data], { type: "image/png" });
        let blobUrl = URL.createObjectURL(blob);
        img.src = blobUrl;
        this.glWindow.setTexture(img);
        this.glWindow.draw();
        let promise = new Promise((resolve, reject) => {
			img.onload = () => {
				this.glWindow.setTexture(img);
				this.glWindow.draw();
				resolve();
			};
			img.onerror = reject;
		});
		await promise;
    }

    putUint32(b, offset, n) {
        let view = new DataView(b);
        view.setUint32(offset, n, false);
    }

    getUint32(b, offset) {
        let view = new DataView(b);
        return view.getUint32(offset, false);
    }

    keyPrompt() {
        let key = prompt("This canvas uses a whitelist.\n\nIf you don't have a key you can still view the canvas but you will not be able to draw.\n\nTo request an access key you can create an issue on the GitHub project.\n\nIf you already have one, enter it here.", "");
        fetch("./verifykey?key="+key)
            .then(async resp => {
                if (resp.ok) {
                    window.location.reload();
                } else {
                    alert("Bad key.")
                }
            });
    }
}