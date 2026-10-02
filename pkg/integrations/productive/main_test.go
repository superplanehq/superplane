package productive

import (
	"os"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	sleep = func(time.Duration) {}
	os.Exit(m.Run())
}
