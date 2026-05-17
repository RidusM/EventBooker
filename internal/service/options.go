package service

type Option func(*EventService)

func BookingTTLMin(ttl int) Option {
	return func(s *EventService) {
		s.bookingTTLMin = ttl
	}
}
