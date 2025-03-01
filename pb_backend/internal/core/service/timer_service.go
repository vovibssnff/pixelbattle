package service

import (
	"context"
	"pb_backend/internal/adapters/redis/repository"
)

type TimerService struct {
	timerRepo repository.TimerRepository
	delay     int
}

func NewTimerService(timerRepo repository.TimerRepository, delay int) *TimerService {
	return &TimerService{
		timerRepo: timerRepo,
		delay:     delay,
	}
}

func (s *TimerService) SetTimer(ctx context.Context, userid int) error {
	return s.timerRepo.SetTimer(ctx, userid, s.delay)
}

func (s *TimerService) CheckTime(ctx context.Context, userid int) (int64, error) {
	return s.timerRepo.CheckTime(ctx, userid)
}
