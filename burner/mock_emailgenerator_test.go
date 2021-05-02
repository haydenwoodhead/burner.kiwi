package burner

import "github.com/stretchr/testify/mock"

type MockEmailGenerator struct {
	mock.Mock
}

func (m *MockEmailGenerator) GetHosts() []string {
	args := m.Called()
	return args.Get(0).([]string)
}

func (m *MockEmailGenerator) HostsContains(host string) bool {
	args := m.Called(host)
	return args.Bool(0)
}

func (m *MockEmailGenerator) NewRandom() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockEmailGenerator) NewFromUserAndHost(r string, h string) (string, error) {
	args := m.Called(r, h)
	return args.String(0), args.Error(1)
}

func (m *MockEmailGenerator) VerifyUser(r string) error {
	args := m.Called(r)
	return args.Error(0)
}

func (m *MockEmailGenerator) VerifyHost(h string) error {
	args := m.Called(h)
	return args.Error(0)
}
