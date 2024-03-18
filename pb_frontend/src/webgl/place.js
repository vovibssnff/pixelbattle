export default class Place {
    loaded;
    socket;
    loadingp;
    uiwrapper;
    glWindow;
    allowDraw;

    constructor(glWindow, uiwrapper, loadingp) {
        this.loaded = false;
        this.socket = null;
        this.loadingp = loadingp;
        this.uiwrapper = uiwrapper;
        this.glWindow = glWindow;
        this.allowDraw = true;
    }

    // initConnection(endpoint) {
    //     fetch(endpoint)
	// 		.then(async resp => {
	// 			let buf = await this.downloadProgress(resp);
	// 			await this.setImage(buf);
	// 		});
    // }

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

    prepare(color) {
        let b = new Uint8Array(11);
        for (let i = 0; i < 3; i++) {
            b[8 + i] = color[i];
        }
        return b;
    }

    setPixel(x, y, color) {
        let b = new Uint8Array(11);
        this.putUint32(b.buffer, 0, x);
        this.putUint32(b.buffer, 4, y);
        for (let i = 0; i < 3; i++) {
            b[8 + i] = color[i];
        }
        this.glWindow.setPixelColor(x, y, color);
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