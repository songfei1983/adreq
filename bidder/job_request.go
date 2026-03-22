package bidder

import "context"

type jobRequest struct {
	ctx context.Context
	job Job
}
