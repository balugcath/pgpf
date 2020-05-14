package transport

// Mock ...
type Mock struct {
	FIsRecovery func() (bool, error)
	FOpen       func(string) error
	FPromote    func(string) error
	FHostStatus func(string, bool) (bool, float64, error)
	PromoteDone map[string]bool
}

// IsRecovery ...
func (s *Mock) IsRecovery() (bool, error) {
	return s.FIsRecovery()
}

// Open ...
func (s *Mock) Open(host string) error {
	return s.FOpen(host)
}

// Promote ...
func (s *Mock) Promote(host string) error {
	s.PromoteDone[host] = true
	return s.FPromote(host)
}

// HostStatus ...
func (s *Mock) HostStatus(host string) (bool, float64, error) {
	return s.FHostStatus(host, s.PromoteDone[host])
}

// Close ...
func (s *Mock) Close() {
}
