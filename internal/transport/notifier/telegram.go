package notifier

import (
	"context"
	"fmt"
	"strings"
	"time"

	"ebooker/internal/entity"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/wb-go/wbf/logger"
)

const _tgTimeout = 30 * time.Second

type TelegramNotifier struct {
	bot *tgbotapi.BotAPI
	log logger.Logger
}

func NewTelegramNotifier(token string, log logger.Logger) (*TelegramNotifier, error) {
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, fmt.Errorf("telegram notifier: init bot: %w", err)
	}
	return &TelegramNotifier{bot: bot, log: log}, nil
}

func (t *TelegramNotifier) StartPolling(
	ctx context.Context,
	onSubscribe func(ctx context.Context, username string, chatID *int64, startPayload string) error,
) {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := t.bot.GetUpdatesChan(u)

	for {
		select {
		case update := <-updates:
			if update.Message == nil || !update.Message.IsCommand() {
				continue
			}
			if update.Message.Command() != "start" {
				continue
			}

			username := update.Message.From.UserName
			if username == "" {
				msg := tgbotapi.NewMessage(
					update.Message.Chat.ID,
					"Для привязки аккаунта необходим username в Telegram.",
				)
				_, _ = t.bot.Send(msg)
				continue
			}

			chatID := update.Message.Chat.ID
			startPayload := update.Message.CommandArguments()

			t.log.LogAttrs(ctx, logger.DebugLevel, "received /start command",
				logger.String("username", username),
				logger.Int64("chat_id", chatID),
				logger.String("payload", startPayload),
			)

			if err := onSubscribe(ctx, username, &chatID, startPayload); err != nil {
				t.log.LogAttrs(ctx, logger.ErrorLevel, "failed to handle subscription",
					logger.String("username", username),
					logger.Any("error", err),
				)
				msg := tgbotapi.NewMessage(chatID, "Произошла ошибка при привязке аккаунта. Попробуйте позже.")
				_, _ = t.bot.Send(msg)
				continue
			}

			responseText := "✅ Вы успешно зарегистрированы в системе уведомлений."
			if startPayload != "" {
				responseText = "✅ Аккаунт успешно привязан! Теперь вы будете получать уведомления."
			}

			msg := tgbotapi.NewMessage(chatID, responseText)
			_, _ = t.bot.Send(msg)

		case <-ctx.Done():
			return
		}
	}
}

func (t *TelegramNotifier) NotifyBookingCancelled(ctx context.Context, req entity.CancelledNotification) error {
	const op = "notifier.telegram.NotifyBookingCancelled"
	if req.TelegramID == nil || *req.TelegramID == 0 {
		return nil
	}

	text := fmt.Sprintf(
		"❌ *Бронь отменена*\n\nЗдравствуйте, %s\\!\n\nВаша бронь на мероприятие *%s* \\(%s\\) "+
			"автоматически отменена из\\-за истечения срока оплаты\\.\n\nВы можете забронировать место снова\\.",
		escapeMarkdown(req.UserName),
		escapeMarkdown(req.EventTitle),
		escapeMarkdown(req.EventDate),
	)

	msg := tgbotapi.NewMessage(*req.TelegramID, text)
	msg.ParseMode = tgbotapi.ModeMarkdownV2

	t.log.LogAttrs(ctx, logger.DebugLevel, "sending cancellation telegram",
		logger.Int64("chat_id", *req.TelegramID),
		logger.String("event", req.EventTitle),
	)

	done := make(chan error, 1)
	go func() {
		_, err := t.bot.Send(msg)
		done <- err
	}()

	select {
	case err := <-done:
		if err != nil {
			return fmt.Errorf("%s: send telegram: %w", op, err)
		}
		return nil
	case <-ctx.Done():
		return fmt.Errorf("%s: context cancelled: %w", op, ctx.Err())
	case <-time.After(_tgTimeout):
		return fmt.Errorf("%s: timeout after %v", op, _tgTimeout)
	}
}

func escapeMarkdown(s string) string {
	replacer := strings.NewReplacer(
		"_", "\\_", "*", "\\*", "[", "\\[", "]", "\\]",
		"(", "\\(", ")", "\\)", "~", "\\~", "`", "\\`",
		">", "\\>", "#", "\\#", "+", "\\+", "-", "\\-",
		"=", "\\=", "|", "\\|", "{", "\\{", "}", "\\}",
		".", "\\.", "!", "\\!",
	)
	return replacer.Replace(s)
}
