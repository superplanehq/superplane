package licensing

import "time"

func (s *Service) SetClock(now func() time.Time) {
	s.now = now
}

func (k *KeySync) SetClock(now func() time.Time) {
	k.now = now
}
