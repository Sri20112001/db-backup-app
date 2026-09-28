package services

import (
	"testing"
	"time"

	"github.com/backup-saas/server/internal/middleware"
)

func TestRateLimiterBudget(t *testing.T) {
	rl := middleware.NewRateLimiter(3, time.Minute)
	for i := 0; i < 3; i++ {
		if !rl.Allow("1.2.3.4") {
			t.Fatalf("attempt %d denied inside budget", i+1)
		}
	}
	if rl.Allow("1.2.3.4") {
		t.Fatal("attempt over budget allowed")
	}
	if !rl.Allow("5.6.7.8") {
		t.Fatal("fresh key denied")
	}
}

func TestRateLimiterWindow(t *testing.T) {
	rl := middleware.NewRateLimiter(1, 20*time.Millisecond)
	if !rl.Allow("k") {
		t.Fatal("first attempt denied")
	}
	if rl.Allow("k") {
		t.Fatal("immediate second attempt allowed")
	}
	time.Sleep(30 * time.Millisecond)
	if !rl.Allow("k") {
		t.Fatal("attempt after window denied")
	}
}
