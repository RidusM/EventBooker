package notifier

import (
	"context"
	"fmt"
	"time"

	"ebooker/internal/entity"

	"github.com/wb-go/wbf/logger"
	"gopkg.in/gomail.v2"
)

const (
	_maxSubjectLength = 255
	_emailTimeout     = 30 * time.Second
)

type EmailNotifier struct {
	dialer *gomail.Dialer
	from   string
	log    logger.Logger
}

func NewEmailNotifier(
	smtpHost string,
	smtpPort int,
	username, password, from string,
	log logger.Logger,
) *EmailNotifier {
	return &EmailNotifier{
		dialer: gomail.NewDialer(smtpHost, smtpPort, username, password),
		from:   from,
		log:    log,
	}
}

func (e *EmailNotifier) NotifyBookingCancelled(ctx context.Context, req entity.CancelledNotification) error {
	const op = "notifier.email.NotifyBookingCancelled"
	if req.UserEmail == "" {
		e.log.LogAttrs(ctx, logger.WarnLevel, "skipping email notification: no user email",
			logger.Any("booking_id", req.BookingID),
			logger.Any("user_id", req.UserID),
		)
		return nil
	}

	subject := fmt.Sprintf("Ваша бронь на «%s» отменена", req.EventTitle)
	if len(subject) > _maxSubjectLength {
		subject = subject[:_maxSubjectLength-3] + "..."
	}

	body := fmt.Sprintf(
		`Здравствуйте, %s!<br><br>
		Ваша бронь на мероприятие <b>«%s»</b> (%s) была автоматически отменена, так как оплата не поступила в срок.<br><br>
		Если хотите, вы можете забронировать место снова.`,
		req.UserName, req.EventTitle, req.EventDate,
	)

	m := gomail.NewMessage()
	m.SetHeader("From", e.from)
	m.SetHeader("To", req.UserEmail)
	m.SetHeader("Subject", subject)
	m.SetBody("text/html", body)

	e.log.LogAttrs(ctx, logger.DebugLevel, "sending cancellation email",
		logger.String("to", req.UserEmail),
		logger.String("event", req.EventTitle),
	)

	done := make(chan error, 1)
	go func() {
		done <- e.dialer.DialAndSend(m)
	}()

	select {
	case err := <-done:
		if err != nil {
			return fmt.Errorf("%s: send email: %w", op, err)
		}
		return nil
	case <-ctx.Done():
		return fmt.Errorf("%s: context cancelled: %w", op, ctx.Err())
	case <-time.After(_emailTimeout):
		return fmt.Errorf("%s: timeout after %v", op, _emailTimeout)
	}
}
