package server

import (
	"context"

	"github.com/songfei1983/adreq/model"
)

type ImpProcessor interface {
	ProcessImp(ctx context.Context, imp *model.Imp) ([]*model.Bid, error)
}
