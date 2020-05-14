package metric

// Mock ...
type Mock struct {
	HostStatusData map[string]int
}

// NewMock ...
func NewMock() *Mock {
	s := &Mock{
		HostStatusData: make(map[string]int),
	}
	return s
}

// ClientConnInc ...
func (Mock) ClientConnInc(string) {}

// ClientConnDec ...
func (Mock) ClientConnDec(string) {}

// TransferBytes ...
func (Mock) TransferBytes(string, string, int) {}

// StatusHost ...
func (s *Mock) StatusHost(host string, status int) {
	s.HostStatusData[host] = status
}
