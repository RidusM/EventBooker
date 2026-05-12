package notifier

import (
	"context"

	"ebooker/internal/entity"
)

type Notifier interface {
	NotifyBookingCancelled(ctx context.Context, req entity.CancelledNotification) error
}
