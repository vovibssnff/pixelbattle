package service

import (
	"context"
	"fmt"
	"pb_backend/internal/adapters/redis/repository"
	"pb_backend/internal/core/domain"
	"pb_backend/internal/utils"
	"time"

	"github.com/sirupsen/logrus"
)

type CanvasService struct {
	canvasRepo repository.CanvasRepository
}

func NewCanvasService(canvasRepo repository.CanvasRepository) *CanvasService {
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
				UserId:    1,
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
	return nil
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
		var deserialized domain.RedisPixel
		if err := utils.DeserializeRedisPixel([]byte(values[0]), &deserialized); err != nil {
			return err
		}
		var y, x uint
		_, err = fmt.Sscanf(key, "pixel:%d:%d", &y, &x)
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

// GetHeatMap retrieves the heatmap data for the canvas.
func (s *CanvasService) GetHeatMap(ctx context.Context) ([]domain.HeatMapUnit, error) {
	heatmapData, err := s.canvasRepo.LoadHeatMap(ctx)
	if err != nil {
		return nil, err
	}
	res := make([]domain.HeatMapUnit, 0)
	for key, length := range heatmapData {
		var y, x uint
		_, err := fmt.Sscanf(key, "pixel:%d:%d", &y, &x)
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
