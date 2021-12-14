package nowish_test

import (
	"testing"
	"time"

	"codeberg.org/gruf/go-nowish"
)

const precision = time.Millisecond * 2

func TestClock(t *testing.T) {
	clock := nowish.Clock{}
	stop := clock.Start(precision)
	defer stop()

	time := clock.Now()
	timeStr := clock.NowFormat()

	t.Logf("%v - %s", time, timeStr)
}

func TestClockAccuracy(t *testing.T) {
	clock := nowish.Clock{}
	stop := clock.Start(precision)
	defer stop()

	for i := 0; i < 1000; i++ {
		nowact := time.Now()
		nowish := clock.Now()

		var diff time.Duration
		if nowish.After(nowact) {
			diff = nowish.Sub(nowact)
		} else {
			diff = nowact.Sub(nowish)
		}

		if float64(diff) > float64(precision)*1.10 {
			t.Fatalf("precision check failed #%d\ndiff=%q\nish=%q\nact=%q", i, diff, nowish, nowact)
		}

		time.Sleep(precision)
	}
}

func BenchmarkClock(b *testing.B) {
	clock := nowish.Clock{}
	stop := clock.Start(precision)
	defer stop()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			now := clock.NowFormat()
			now += "0"
			_ = now
		}
	})
}

func BenchmarkTime(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			now := time.Now().Format(time.RFC822)
			now += "0"
			_ = now
		}
	})
}
