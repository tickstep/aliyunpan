package command

import (
	"fmt"
	"testing"
)

func TestProgressiveBackoffAlg(t *testing.T) {
	for attempt := 1; attempt <= 150; attempt++ {
		delayTime := ProgressiveBackoffAlg(attempt)
		fmt.Printf("第%d次，延迟分钟: %.2f\n", attempt, delayTime.Minutes())
	}
}
