package service

import (
	"context"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"pb_backend/internal/adapters/influxdb"
	"pb_backend/internal/core/domain"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type HeatmapService struct {
	influxDB         *influxdb.InfluxDBAdapter
	telegramBotToken string
	telegramChatID   string
}

func NewHeatmapService(influxDB *influxdb.InfluxDBAdapter, telegramBotToken, telegramChatID string) *HeatmapService {
	return &HeatmapService{
		influxDB:         influxDB,
		telegramBotToken: telegramBotToken,
		telegramChatID:   telegramChatID,
	}
}

// RecordPixelChange records a pixel change in InfluxDB
func (s *HeatmapService) RecordPixelChange(ctx context.Context, p *domain.Pixel) error {
	return s.influxDB.RecordPixelChange(ctx, *p)
}

// GetHeatmapData returns heatmap data for the last hour
func (s *HeatmapService) GetHeatmapData(ctx context.Context) (map[string]int, error) {
	return s.influxDB.GetHeatmapData(ctx)
}

// CheckAndAlertMostActiveZone checks the most active zone and sends an alert if needed
func (s *HeatmapService) CheckAndAlertMostActiveZone(ctx context.Context, resolution int, threshold int) error {
	zoneX, zoneY, err := s.influxDB.GetMostActiveZone(ctx, resolution)
	if err != nil {
		return fmt.Errorf("failed to get most active zone: %w", err)
	}

	// Get the heatmap data for this zone
	heatmap, err := s.influxDB.GetHeatmapData(ctx)
	if err != nil {
		return fmt.Errorf("failed to get heatmap data: %w", err)
	}

	// Count activity in this zone
	zoneActivity := 0
	for key, count := range heatmap {
		var x, y int
		fmt.Sscanf(key, "%d,%d", &x, &y)
		if x/resolution == zoneX/resolution && y/resolution == zoneY/resolution {
			zoneActivity += count
		}
	}

	if zoneActivity >= threshold {
		// Create a temporary image of the zone
		img := image.NewRGBA(image.Rect(0, 0, resolution, resolution))
		// TODO: Fill the image with actual pixel data from Redis

		// Save the image
		tempDir := os.TempDir()
		imgPath := filepath.Join(tempDir, fmt.Sprintf("active_zone_%d.png", time.Now().Unix()))
		file, err := os.Create(imgPath)
		if err != nil {
			return fmt.Errorf("failed to create image file: %w", err)
		}
		defer os.Remove(imgPath)
		defer file.Close()

		if err := png.Encode(file, img); err != nil {
			return fmt.Errorf("failed to encode image: %w", err)
		}

		// Send alert to Telegram
		message := fmt.Sprintf("High activity detected in zone (%d,%d)!\nActivity level: %d", zoneX, zoneY, zoneActivity)
		if err := s.sendTelegramAlert(message, imgPath); err != nil {
			return fmt.Errorf("failed to send telegram alert: %w", err)
		}
	}

	return nil
}

func (s *HeatmapService) sendTelegramAlert(message, imagePath string) error {
	bot, err := tgbotapi.NewBotAPI(s.telegramBotToken)
	if err != nil {
		return fmt.Errorf("failed to create telegram bot: %w", err)
	}

	// Send text message
	chatID, _ := strconv.ParseInt(s.telegramChatID, 10, 64)
	msg := tgbotapi.NewMessage(chatID, message)
	if _, err := bot.Send(msg); err != nil {
		return fmt.Errorf("failed to send telegram message: %w", err)
	}

	// Send image
	photo := tgbotapi.NewPhoto(chatID, tgbotapi.FilePath(imagePath))
	if _, err := bot.Send(photo); err != nil {
		return fmt.Errorf("failed to send telegram photo: %w", err)
	}

	return nil
}
