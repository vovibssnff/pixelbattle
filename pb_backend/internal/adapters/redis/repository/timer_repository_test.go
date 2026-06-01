package repository

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func TestTimerRepositoryPersistsZeroCooldown(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis.Run: %v", err)
	}
	defer mr.Close()

	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer rdb.Close()

	repo := NewTimerRepo(rdb)
	ctx := context.Background()

	if _, ok, err := repo.GetCooldownSeconds(ctx); err != nil || ok {
		t.Fatalf("GetCooldownSeconds before set = ok:%v err:%v, want missing nil", ok, err)
	}
	if err := repo.SetCooldownSeconds(ctx, 0); err != nil {
		t.Fatalf("SetCooldownSeconds(0): %v", err)
	}
	got, ok, err := repo.GetCooldownSeconds(ctx)
	if err != nil {
		t.Fatalf("GetCooldownSeconds: %v", err)
	}
	if !ok || got != 0 {
		t.Fatalf("GetCooldownSeconds = %d ok:%v, want 0 true", got, ok)
	}
}

func TestTimerRepositoryZeroDelayDeletesExistingTimer(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis.Run: %v", err)
	}
	defer mr.Close()

	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer rdb.Close()

	repo := NewTimerRepo(rdb)
	ctx := context.Background()

	if err := repo.SetTimer(ctx, "user-1", 3); err != nil {
		t.Fatalf("SetTimer(3): %v", err)
	}
	mr.FastForward(time.Second)
	exists, err := repo.CheckTime(ctx, "user-1")
	if err != nil {
		t.Fatalf("CheckTime after set: %v", err)
	}
	if exists != 1 {
		t.Fatalf("CheckTime after set = %d, want 1", exists)
	}

	if err := repo.SetTimer(ctx, "user-1", 0); err != nil {
		t.Fatalf("SetTimer(0): %v", err)
	}
	exists, err = repo.CheckTime(ctx, "user-1")
	if err != nil {
		t.Fatalf("CheckTime after zero: %v", err)
	}
	if exists != 0 {
		t.Fatalf("CheckTime after zero = %d, want 0", exists)
	}
}
