package notifier

import (
	"context"
	"fmt"
	"strings"

	"ebooker/internal/entity"
)

type MultiNotifier struct {
	notifiers []Notifier
}

func NewMultiNotifier(notifiers ...Notifier) *MultiNotifier {
	return &MultiNotifier{notifiers: notifiers}
}

func (m *MultiNotifier) NotifyBookingCancelled(ctx context.Context, req entity.CancelledNotification) error {
	if len(m.notifiers) == 0 {
		return nil
	}

	var errs []string
	for _, n := range m.notifiers {
		if err := n.NotifyBookingCancelled(ctx, req); err != nil {
			errs = append(errs, err.Error())
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("notifier: multiple failures: %s", strings.Join(errs, "; "))
	}
	return nil
}
