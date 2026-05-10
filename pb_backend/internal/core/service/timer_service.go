package service

import (
	"context"
	"fmt"
	"pb_backend/internal/adapters/redis/repository"
	"sync"
)

type TimerService struct {
	timerRepo repository.TimerRepository
	mu        sync.RWMutex
	delay     int
}

func NewTimerService(timerRepo repository.TimerRepository, delay int) *TimerService {
	if delay <= 0 {
		delay = 3
	}
	return &TimerService{
		timerRepo: timerRepo,
		delay:     delay,
	}
}

func (s *TimerService) SetTimer(ctx context.Context, userid string) error {
	s.mu.RLock()
	d := s.delay
	s.mu.RUnlock()
	return s.timerRepo.SetTimer(ctx, userid, d)
}

func (s *TimerService) CheckTime(ctx context.Context, userid string) (int64, error) {
	return s.timerRepo.CheckTime(ctx, userid)
}

// SetCooldownSeconds atomically updates the placement cooldown Redis TTL (1..3600).
func (s *TimerService) SetCooldownSeconds(sec int) error {
	if sec < 1 || sec > 3600 {
		return fmt.Errorf("cooldown %d out of range (1..3600)", sec)
	}
	s.mu.Lock()
	s.delay = sec
	s.mu.Unlock()
	return nil
}

func (s *TimerService) CooldownSeconds() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.delay
}
