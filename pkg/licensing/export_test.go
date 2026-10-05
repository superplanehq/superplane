package licensing

import "time"

func (s *Service) SetClock(now func() time.Time) {
	s.now = now
}
