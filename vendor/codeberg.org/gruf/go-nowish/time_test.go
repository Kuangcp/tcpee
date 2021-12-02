package nowish_test

import (
	"fmt"
	"io"
	"testing"
	"time"

	"codeberg.org/gruf/go-nowish"
)

const precision = time.Millisecond

func TestClock(t *testing.T) {
	clock := nowish.Clock{}
	stop := clock.Start(precision)
	defer stop()

	time := clock.Now()
	timeStr := clock.NowFormat()

	t.Logf("%v - %s", time, timeStr)
}

func BenchmarkClock(b *testing.B) {
	clock := nowish.Clock{}
	stop := clock.Start(precision)
	defer stop()

	b.RunParallel(func(p *testing.PB) {
		for p.Next() {
			formattedTime := clock.NowFormat()
			fmt.Fprintln(io.Discard, formattedTime)
		}
	})
}
