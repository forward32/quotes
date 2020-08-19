package onederx

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"quotes/pkg/types"
)

const (
	onederxWsUrl = "wss://api.onederx.com/v1/ws"

	defaultRetryInterval = time.Second
	defaultReadTimeout   = time.Second * 5
	defaultWriteTimeout  = time.Second * 5
)

type Source struct {
	sync.RWMutex
	l2BySymbol map[string]*L2OrderBook
}

func NewSource() *Source {
	return &Source{
		l2BySymbol: make(map[string]*L2OrderBook),
	}
}

func (s *Source) Start(ctx context.Context) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				log.Printf("stop source: context cancelled")
				return
			default:
				if err := s.receiveData(ctx); err != nil {
					log.Printf("receiving failed: %v", err)
				}
				log.Printf("sleep for %s", defaultRetryInterval)
				time.Sleep(defaultRetryInterval)
			}
		}
	}()
}

func (s *Source) GetL2OrderBook(symbol string, size int) (types.L2OrderBook, error) {
	s.RLock()
	defer s.RUnlock()

	l2, ok := s.l2BySymbol[symbol]
	if !ok {
		return types.L2OrderBook{}, fmt.Errorf("no data for symbol %s", symbol)
	}

	return types.L2OrderBook{
		Bid: l2.GetBid(size),
		Ask: l2.GetAsk(size),
	}, nil
}

func (s *Source) receiveData(ctx context.Context) error {
	dialer := websocket.Dialer{}
	conn, _, err := dialer.DialContext(ctx, onederxWsUrl, nil)
	if err != nil {
		return err
	}

	if err := conn.SetWriteDeadline(time.Now().Add(defaultWriteTimeout)); err != nil {
		return err
	}
	if err := conn.WriteJSON(GetWsL2SubscribeRequest()); err != nil {
		return err
	}

	var header struct{ Type string }
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			if err := conn.SetReadDeadline(time.Now().Add(defaultReadTimeout)); err != nil {
				return err
			}

			mt, data, err := conn.ReadMessage()
			if err != nil {
				return err
			}

			if mt != websocket.TextMessage {
				return fmt.Errorf("unexpected message type %d", mt)
			}

			if err := json.Unmarshal(data, &header); err != nil {
				return err
			}

			switch header.Type {
			case "snapshot":
				err = s.onSnapshot(data)
			case "update":
				err = s.onUpdate(data)
			default:
			}
			if err != nil {
				return err
			}
		}
	}
}

func (s *Source) onSnapshot(data []byte) error {
	var snapshot WsL2Snapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return err
	}

	s.Lock()
	defer s.Unlock()

	l2 := NewL2OrderBook()
	s.l2BySymbol[snapshot.Params.Symbol] = l2

	for _, items := range [][]*WsL2Item{
		snapshot.Payload.Snapshot,
		snapshot.Payload.Updates,
	} {
		for _, item := range items {
			side := types.SideFromString(item.Side)
			tm := time.Unix(0, item.Timestamp)
			l2.Apply(item.Price, side, item.Volume, tm)
		}
	}

	log.Printf("snapshot applied: symbol=%s, bid=%d, ask=%d",
		snapshot.Params.Symbol, l2.bid.Len(), l2.ask.Len())

	return nil
}

func (s *Source) onUpdate(data []byte) error {
	var update WsL2Update
	if err := json.Unmarshal(data, &update); err != nil {
		return err
	}

	s.Lock()
	defer s.Unlock()

	l2, ok := s.l2BySymbol[update.Params.Symbol]
	if !ok {
		log.Printf("inconsistent update for symbol %s", update.Params.Symbol)
		return nil
	}

	side := types.SideFromString(update.Payload.Side)
	tm := time.Unix(0, update.Payload.Timestamp)
	l2.Apply(update.Payload.Price, side, update.Payload.Volume, tm)

	log.Printf("update applied: symbol=%s, bid=%d, ask=%d",
		update.Params.Symbol, l2.bid.Len(), l2.ask.Len())

	return nil
}
