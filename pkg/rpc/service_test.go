package rpc

import (
	"context"
	"fmt"
	"net"
	"sync"
	"testing"
	"time"

	"quotes/api"
	"quotes/pkg/types"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

type testSource struct{}

func (testSource) GetL2OrderBook(symbol string, size int) (types.L2OrderBook, error) {
	return types.L2OrderBook{
		Bid: []*types.L2OrderBookItem{
			{
				Price:  decimal.New(100, 0),
				Volume: 100,
				Time:   time.Now(),
			},
		},
		Ask: []*types.L2OrderBookItem{
			{
				Price:  decimal.New(200, 0),
				Volume: 200,
				Time:   time.Now(),
			},
		},
	}, nil
}

func TestGetL2OrderBook(t *testing.T) {
	var allDone sync.WaitGroup
	defer allDone.Wait()

	service := NewService()
	service.AddSource(&testSource{})
	server := grpc.NewServer()
	api.RegisterQuotesServer(server, service)

	lsn, err := net.Listen("tcp", "localhost:0")
	require.Nil(t, err)
	defer lsn.Close()
	addr := fmt.Sprintf("localhost:%v", lsn.Addr().(*net.TCPAddr).Port)

	allDone.Add(1)
	go func() {
		defer allDone.Done()
		_ = server.Serve(lsn)
	}()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	conn, err := grpc.DialContext(ctx, addr, grpc.WithInsecure(), grpc.WithBlock())
	require.Nil(t, err)
	client := api.NewQuotesClient(conn)

	req := api.L2OrderBookRequest{
		Symbol:   "BTCUSD_P",
		Size:     10,
		Interval: 100,
	}
	stream, err := client.GetL2OrderBook(context.Background(), &req)
	require.Nil(t, err)
	l2, err := stream.Recv()
	require.Nil(t, err)
	require.NotNil(t, l2)
	require.Equal(t, req.Symbol, l2.Symbol)
	require.Len(t, l2.Bid, 1)
	require.Len(t, l2.Ask, 1)
}
