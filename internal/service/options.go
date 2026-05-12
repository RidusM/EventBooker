package service

import "time"

type Option func(*EventService)

func BookEventTTL(ttl time.Duration) Option {
	return func(s *EventService) {
		s.bookEventTTL = ttl
	}
}
