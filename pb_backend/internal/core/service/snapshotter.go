package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	"pb_backend/internal/core/domain"
	"pb_backend/internal/utils"
)

// CanvasSnapshotter periodically renders the canvas to PNG for fast cold load
// (plan §11.3). Serves GET /api/canvas.png with ETag + X-Snapshot-Ms; optional
// atomic file write for static serving.
type CanvasSnapshotter struct {
	canvas   domain.CanvasService
	height   uint
	width    uint
	interval time.Duration
	filePath string

	mu     sync.RWMutex
	png    []byte
	etag   string
	unixMs int64

	stop chan struct{}
	wg   sync.WaitGroup
}

// NewCanvasSnapshotter builds a snapshotter. filePath may be empty (no disk write).
func NewCanvasSnapshotter(canvas domain.CanvasService, height, width uint, interval time.Duration, filePath string) *CanvasSnapshotter {
	if interval <= 0 {
		interval = 2 * time.Second
	}
	return &CanvasSnapshotter{
		canvas:   canvas,
		height:   height,
		width:    width,
		interval: interval,
		filePath: filePath,
		stop:     make(chan struct{}),
	}
}

// Start launches the refresh loop after an initial synchronous refresh.
func (s *CanvasSnapshotter) Start() {
	s.refresh()
	s.wg.Add(1)
	go s.loop()
	logrus.Infof("canvas_snapshotter: started (interval=%s, file=%q)", s.interval, s.filePath)
}

// Stop waits for the background loop to exit.
func (s *CanvasSnapshotter) Stop() {
	close(s.stop)
	s.wg.Wait()
}

func (s *CanvasSnapshotter) loop() {
	defer s.wg.Done()
	t := time.NewTicker(s.interval)
	defer t.Stop()
	for {
		select {
		case <-s.stop:
			return
		case <-t.C:
			s.refresh()
		}
	}
}

func (s *CanvasSnapshotter) refresh() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	w, h, dimErr := s.canvas.CanvasDimensions(ctx)
	if dimErr != nil || w == 0 || h == 0 {
		w, h = s.width, s.height
	}
	img := s.canvas.CreateImage(h, w)
	if err := s.canvas.GetCanvas(ctx, img); err != nil {
		logrus.Errorf("canvas_snapshotter: GetCanvas: %v", err)
		return
	}
	b, err := utils.GetImageBytes(img)
	if err != nil {
		logrus.Errorf("canvas_snapshotter: encode png: %v", err)
		return
	}

	sum := sha256.Sum256(b)
	etag := fmt.Sprintf(`"%s"`, hex.EncodeToString(sum[:16]))
	ms := time.Now().UnixMilli()

	s.mu.Lock()
	s.png = append([]byte(nil), b...)
	s.etag = etag
	s.unixMs = ms
	s.mu.Unlock()

	if s.filePath != "" {
		if err := atomicWriteFile(s.filePath, b); err != nil {
			logrus.Warnf("canvas_snapshotter: write %s: %v", s.filePath, err)
		}
	}
}

func atomicWriteFile(path string, data []byte) error {
	dir := filepath.Dir(path)
	base := filepath.Base(path)
	tmp, err := os.CreateTemp(dir, "."+base+".tmp-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	return os.Rename(tmpPath, path)
}

// Get returns the latest PNG, strong ETag, and wall-clock ms when that snapshot
// was committed (for WS replay_after_ms).
func (s *CanvasSnapshotter) Get() (png []byte, etag string, unixMs int64, ok bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if len(s.png) == 0 {
		return nil, "", 0, false
	}
	return append([]byte(nil), s.png...), s.etag, s.unixMs, true
}
