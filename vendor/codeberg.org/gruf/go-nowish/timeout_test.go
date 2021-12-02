package nowish_test

import (
	"testing"
	"time"

	"codeberg.org/gruf/go-nowish"
)

func TestTimeoutDidTimeout(t *testing.T) {
	timedOut := false
	onTimeout := func() {
		t.Log("Successfully timed out")
		timedOut = true
	}

	to := nowish.NewTimeout()
	to.Start(time.Second, onTimeout)
	time.Sleep(time.Second * 2)
	to.Cancel()
	if !timedOut {
		t.Fatal("Expected timeout")
	}
}

func TestTimoutNeverStarted(t *testing.T) {
	to := nowish.NewTimeout()
	to.Cancel()
}

func TestTimeoutNoTimeout(t *testing.T) {
	to := nowish.NewTimeout()
	to.Start(time.Second, func() {
		t.Fatal("Unexpected timeout")
	})
	to.Cancel()
}

func TestTimeoutReuse(t *testing.T) {
	onTimeout := func() {
		t.Fatal("Unexpected timeout")
	}

	to := nowish.NewTimeout()
	to.Start(time.Second, onTimeout)
	to.Cancel()
	to.Start(time.Second, onTimeout)
	to.Cancel()
}

func TestTimeoutReuseMulti(t *testing.T) {
	to := nowish.NewTimeout()
	for i := 0; i < 1000; i++ {
		catchPanic(
			func() {
				to.Start(time.Millisecond, func() {
					t.Logf("WARN [%d]: unexpected timeout", i)
				})
			},
			func(r interface{}) {
				t.Logf("WARN [%d]: %v", i, r)
				time.Sleep(time.Microsecond)
			},
		)
		to.Cancel()
	}
}

func catchPanic(fn func(), onPanic func(interface{})) {
	defer func() {
		r := recover()
		if r != nil {
			onPanic(r)
		}
	}()
	fn()
}
