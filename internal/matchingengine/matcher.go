package matchingengine

import (
	"context"
	"fmt"
	"math/big"

	"github.com/fathimasithara01/multitrade-platform/internal/order"
)

func (e *Engine) findBestCandidate(ctx context.Context, assetID int64, side string, incoming *order.Order) (*order.Order, error) {
	candidates, err := e.orderRepo.GetOpenOrdersForMatching(ctx, nil, assetID, side)
	if err != nil {
		return nil, err
	}

	incomingPrice := mustParseBig(incoming.Price)

	for i := range candidates {
		c := &candidates[i]
		if c.UserID == incoming.UserID {
			continue
		}
		candidatePrice := mustParseBig(c.Price)

		crosses := false
		if incoming.Side == order.OrderSideBuy {
			crosses = candidatePrice.Cmp(incomingPrice) <= 0
		} else {
			crosses = candidatePrice.Cmp(incomingPrice) >= 0
		}
		if crosses {
			return c, nil
		}
	}
	return nil, nil
}

func isOpen(status string) bool {
	return status == order.OrderStatusPending || status == order.OrderStatusPartiallyFilled
}

func newStatus(filled, total *big.Float) string {
	if filled.Cmp(total) >= 0 {
		return order.OrderStatusFilled
	}
	return order.OrderStatusPartiallyFilled
}

func mustParseBig(s string) *big.Float {
	f, _ := new(big.Float).SetPrec(128).SetString(s)
	return f
}

func bigZero() *big.Float {
	return new(big.Float).SetPrec(128)
}

func minBig(a, b *big.Float) *big.Float {
	if a.Cmp(b) <= 0 {
		return new(big.Float).Copy(a)
	}
	return new(big.Float).Copy(b)
}

func fmt8f(f *big.Float) string {
	return fmt.Sprintf("%.8f", f)
}
