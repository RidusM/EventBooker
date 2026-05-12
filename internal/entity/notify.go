package entity

type CancelledNotification struct {
	UserName   string
	UserEmail  string
	TelegramID *int64
	EventTitle string
	EventDate  string
}
