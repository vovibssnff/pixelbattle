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
	if delay < 0 {
		delay = 3
	}
	return &TimerService{
		timerRepo: timerRepo,
		delay:     delay,
	}
}

func (s *TimerService) SetTimer(ctx context.Context, userid string) error {
	d, err := s.CooldownSeconds(ctx)
	if err != nil {
		return err
	}
	return s.timerRepo.SetTimer(ctx, userid, d)
}

func (s *TimerService) CheckTime(ctx context.Context, userid string) (int64, error) {
	return s.timerRepo.CheckTime(ctx, userid)
}

// SetCooldownSeconds atomically updates the placement cooldown Redis TTL (0..3600).
// 0 disables per-user placement cooldown while leaving other protections intact.
func (s *TimerService) SetCooldownSeconds(ctx context.Context, sec int) error {
	if sec < 0 || sec > 3600 {
		return fmt.Errorf("cooldown %d out of range (0..3600)", sec)
	}
	if err := s.timerRepo.SetCooldownSeconds(ctx, sec); err != nil {
		return err
	}
	s.mu.Lock()
	s.delay = sec
	s.mu.Unlock()
	return nil
}

func (s *TimerService) CooldownSeconds(ctx context.Context) (int, error) {
	if sec, ok, err := s.timerRepo.GetCooldownSeconds(ctx); err != nil {
		return 0, err
	} else if ok {
		s.mu.Lock()
		s.delay = sec
		s.mu.Unlock()
		return sec, nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.delay, nil
}
