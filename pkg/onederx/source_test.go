package onederx

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSource(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	source := NewSource()
	source.Start(ctx)

	ok := false
	for i := 0; i < 10 && !ok; i++ {
		l2, err := source.GetL2OrderBook("BTCUSD_P", 5)
		if err == nil {
			require.NotNil(t, l2)
			require.Len(t, l2.Bid, 5)
			require.Len(t, l2.Ask, 5)
			for _, item := range append(l2.Bid, l2.Ask...) {
				require.False(t, item.Price.IsZero())
				require.NotZero(t, item.Volume)
			}
			ok = true
		} else {
			time.Sleep(time.Second)
		}
	}
	require.True(t, ok)
}
