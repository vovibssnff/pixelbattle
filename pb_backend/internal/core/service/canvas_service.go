package service

import (
	"context"
	"fmt"
	"pb_backend/internal/core/domain"
	"pb_backend/internal/utils"
	"time"

	"github.com/sirupsen/logrus"
)

type CanvasService struct {
	canvasRepo domain.CanvasRepository
}

func NewCanvasService(canvasRepo domain.CanvasRepository) *CanvasService {
	return &CanvasService{
		canvasRepo: canvasRepo,
	}
}

// WritePixel writes a pixel to the canvas.
func (s *CanvasService) WritePixel(ctx context.Context, p *domain.Pixel) error {
	redisPixel := &domain.RedisPixel{
		UserId:    p.Userid,
		Faculty:   p.Faculty,
		Color:     p.Color,
		Timestamp: time.Now().Unix(),
	}
	serializedRedisPixel, err := utils.SerializeRedisPixel(redisPixel)
	if err != nil {
		return err
	}
	return s.canvasRepo.WritePixel(ctx, p.X, p.Y, serializedRedisPixel)
}

// InitializeCanvas initializes the canvas with default pixels.
func (s *CanvasService) InitializeCanvas(ctx context.Context, height uint, width uint) error {
	logrus.Infof("Starting canvas initialization with dimensions %dx%d", width, height)
	totalPixels := height * width
	initialized := uint(0)

	for i := 0; i < int(height); i++ {
		for j := 0; j < int(width); j++ {
			redisPixel := &domain.RedisPixel{
				UserId:    "",
				Faculty:   "",
				Color:     []uint{255, 255, 255},
				Timestamp: time.Now().Unix(),
			}
			serializedRedisPixel, err := utils.SerializeRedisPixel(redisPixel)
			if err != nil {
				return fmt.Errorf("failed to serialize pixel at (%d,%d): %v", j, i, err)
			}
			err = s.canvasRepo.WritePixel(ctx, uint(j), uint(i), serializedRedisPixel)
			if err != nil {
				return fmt.Errorf("failed to write pixel at (%d,%d): %v", j, i, err)
			}
			initialized++
			if initialized%1000 == 0 {
				logrus.Infof("Initialized %d/%d pixels (%.2f%%)", initialized, totalPixels, float64(initialized)/float64(totalPixels)*100)
			}
		}
	}
	logrus.Info("Canvas initialization completed")
	if err := s.canvasRepo.SetCanvasDimensions(ctx, width, height); err != nil {
		return fmt.Errorf("set canvas dimensions: %w", err)
	}
	return nil
}

func (s *CanvasService) inferMaxExtentFromKeys(ctx context.Context) (uint, uint, error) {
	canvasData, err := s.canvasRepo.GetCanvas(ctx)
	if err != nil {
		return 0, 0, err
	}
	var maxX, maxY uint
	for key := range canvasData {
		x, y, err := utils.ParseRedisPixelKey(key)
		if err != nil {
			continue
		}
		if x > maxX {
			maxX = x
		}
		if y > maxY {
			maxY = y
		}
	}
	if len(canvasData) == 0 {
		return 0, 0, nil
	}
	return maxX + 1, maxY + 1, nil
}

// CanvasDimensions returns stored logical size or infers from existing pixel keys (ADR-003).
func (s *CanvasService) CanvasDimensions(ctx context.Context) (uint, uint, error) {
	w, h, err := s.canvasRepo.GetCanvasDimensions(ctx)
	if err != nil {
		return 0, 0, err
	}
	if w > 0 && h > 0 {
		return w, h, nil
	}
	return s.inferMaxExtentFromKeys(ctx)
}

// ExpandCanvas adds white cells so the canvas grows to width×height (expand-only). See ADR-003.
func (s *CanvasService) ExpandCanvas(ctx context.Context, width, height uint) error {
	curW, curH, err := s.CanvasDimensions(ctx)
	if err != nil {
		return err
	}
	if curW == 0 || curH == 0 {
		return fmt.Errorf("canvas dimensions unknown")
	}
	if width < curW || height < curH {
		return fmt.Errorf("resize: expand only (current %dx%d)", curW, curH)
	}
	if width == curW && height == curH {
		return nil
	}
	redisPixel := &domain.RedisPixel{
		UserId:    "",
		Faculty:   "",
		Color:     []uint{255, 255, 255},
		Timestamp: time.Now().Unix(),
	}
	white, err := utils.SerializeRedisPixel(redisPixel)
	if err != nil {
		return err
	}
	for y := uint(0); y < height; y++ {
		for x := uint(0); x < width; x++ {
			if x >= curW || y >= curH {
				if err := s.canvasRepo.WritePixel(ctx, x, y, white); err != nil {
					return fmt.Errorf("write expand cell (%d,%d): %w", x, y, err)
				}
			}
		}
	}
	return s.canvasRepo.SetCanvasDimensions(ctx, width, height)
}

// EnsureCanvasInitialized repairs partial canvases by filling missing Redis cells.
// This prevents a single surviving pixel key from making startup skip initialization.
func (s *CanvasService) EnsureCanvasInitialized(ctx context.Context, height uint, width uint) error {
	canvasData, err := s.canvasRepo.GetCanvas(ctx)
	if err != nil {
		return err
	}
	if len(canvasData) == 0 {
		logrus.Info("Canvas empty; initializing with white pixels")
		return s.InitializeCanvas(ctx, height, width)
	}

	w, h, err := s.CanvasDimensions(ctx)
	if err != nil {
		return err
	}
	if w == 0 || h == 0 {
		w, h = width, height
	}
	// Never shrink below configured env size (e.g. after interrupted init left a partial grid).
	if width > w {
		w = width
	}
	if height > h {
		h = height
	}

	present := make(map[string]struct{}, len(canvasData))
	for key := range canvasData {
		x, y, err := utils.ParseRedisPixelKey(key)
		if err != nil {
			return err
		}
		if x < w && y < h {
			present[fmt.Sprintf("%d:%d", x, y)] = struct{}{}
		}
	}
	expected := w * h
	if uint(len(present)) == expected {
		return s.canvasRepo.SetCanvasDimensions(ctx, w, h)
	}

	redisPixel := &domain.RedisPixel{
		UserId:    "",
		Faculty:   "",
		Color:     []uint{255, 255, 255},
		Timestamp: time.Now().Unix(),
	}
	white, err := utils.SerializeRedisPixel(redisPixel)
	if err != nil {
		return err
	}
	missing := uint(0)
	for y := uint(0); y < h; y++ {
		for x := uint(0); x < w; x++ {
			key := fmt.Sprintf("%d:%d", x, y)
			if _, ok := present[key]; ok {
				continue
			}
			if err := s.canvasRepo.WritePixel(ctx, x, y, white); err != nil {
				return fmt.Errorf("repair missing canvas cell (%d,%d): %w", x, y, err)
			}
			missing++
		}
	}
	logrus.Infof("Repaired %d missing canvas pixels for %dx%d canvas", missing, w, h)
	return s.canvasRepo.SetCanvasDimensions(ctx, w, h)
}

// IsCanvasInitialized checks if the canvas is already initialized.
func (s *CanvasService) IsCanvasInitialized(ctx context.Context) bool {
	return s.canvasRepo.CheckInitialized(ctx)
}

// GetCanvas retrieves the current state of the canvas.
func (s *CanvasService) GetCanvas(ctx context.Context, img *domain.Image) error {
	canvasData, err := s.canvasRepo.GetCanvas(ctx)
	if err != nil {
		return err
	}
	for key, values := range canvasData {
		if len(values) == 0 {
			continue
		}
		var deserialized domain.RedisPixel
		if err := utils.DeserializeRedisPixel([]byte(values[0]), &deserialized); err != nil {
			return err
		}
		x, y, err := utils.ParseRedisPixelKey(key)
		if err != nil {
			return err
		}
		pixel := domain.Pixel{
			X:     x,
			Y:     y,
			Color: deserialized.Color,
		}
		img.Data = append(img.Data, pixel)
	}
	return nil
}

func (s *CanvasService) GetPixelInfo(ctx context.Context, x, y uint) (domain.PixelInfo, error) {
	rp, err := s.canvasRepo.GetLatestPixel(ctx, x, y)
	if err != nil {
		return domain.PixelInfo{}, err
	}
	return domain.PixelInfo{
		X:         x,
		Y:         y,
		Color:     rp.Color,
		UserID:    rp.UserId,
		Faculty:   rp.Faculty,
		Timestamp: rp.Timestamp,
	}, nil
}

func (s *CanvasService) GetPixelInfoCache(ctx context.Context) ([]domain.PixelInfo, error) {
	canvasData, err := s.canvasRepo.GetCanvas(ctx)
	if err != nil {
		return nil, err
	}
	pixels := make([]domain.PixelInfo, 0, len(canvasData))
	for key, values := range canvasData {
		if len(values) == 0 {
			continue
		}
		x, y, err := utils.ParseRedisPixelKey(key)
		if err != nil {
			return nil, err
		}
		var rp domain.RedisPixel
		if err := utils.DeserializeRedisPixel([]byte(values[0]), &rp); err != nil {
			return nil, err
		}
		pixels = append(pixels, domain.PixelInfo{
			X:         x,
			Y:         y,
			Color:     rp.Color,
			UserID:    rp.UserId,
			Faculty:   rp.Faculty,
			Timestamp: rp.Timestamp,
		})
	}
	return pixels, nil
}

// GetHeatMap retrieves the heatmap data for the canvas.
func (s *CanvasService) GetHeatMap(ctx context.Context) ([]domain.HeatMapUnit, error) {
	heatmapData, err := s.canvasRepo.LoadHeatMap(ctx)
	if err != nil {
		return nil, err
	}
	res := make([]domain.HeatMapUnit, 0)
	for key, length := range heatmapData {
		x, y, err := utils.ParseRedisPixelKey(key)
		if err != nil {
			return nil, err
		}
		res = append(res, domain.HeatMapUnit{X: x, Y: y, Len: uint(length)})
	}
	return res, nil
}

// CreateImage creates a new image with the specified height and width.
func (s *CanvasService) CreateImage(h, w uint) *domain.Image {
	return &domain.Image{
		Height: h,
		Width:  w,
		Data:   make([]domain.Pixel, 0),
	}
}
