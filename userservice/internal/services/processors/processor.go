package processors

import (
	"context"
)

type Processor interface {
	ProcessEvent(ctx context.Context, payload []byte) error
}
