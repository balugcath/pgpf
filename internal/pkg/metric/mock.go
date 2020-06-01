package metric

// Mock ...
type Mock struct {
}

// NewMock ...
func NewMock() *Mock {
	s := &Mock{}
	return s
}

// ClientConnInc ...
func (Mock) ClientConnInc(string) {}

// ClientConnDec ...
func (Mock) ClientConnDec(string) {}

// TransferBytes ...
func (Mock) TransferBytes(string, string, int) {}
