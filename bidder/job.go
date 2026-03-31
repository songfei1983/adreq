package bidder

import "context"

type Job = func(ctx context.Context)
