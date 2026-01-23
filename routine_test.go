package jobq

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestRoutineGroup_Run(t *testing.T) {
	t.Run("execute single function", func(t *testing.T) {
		rg := new(routineGroup)

		var counter atomic.Int32

		rg.Run(func() {
			counter.Add(1)
		})
		rg.Wait()
		load := counter.Load()
		require.Equal(t, int32(1), load, "expected counter to be 1, got=%d", load)
	})
	t.Run("execute multiple functions", func(t *testing.T) {
		rg := new(routineGroup)

		var counter atomic.Int32

		for i := 0; i < 10; i++ {
			rg.Run(func() {
				counter.Add(1)
				time.Sleep(100 * time.Millisecond)
			})
		}
		rg.Wait()
		load := counter.Load()
		require.Equal(t, int32(10), load, "expected counter to be 10, got=%d", load)
	})
}
