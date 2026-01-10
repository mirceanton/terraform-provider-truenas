package client

import (
	"context"
	"encoding/json"
)

// Client defines the interface for communicating with TrueNAS.
type Client interface {
	// Call executes a midclt command and returns the parsed JSON response.
	Call(ctx context.Context, method string, params any) (json.RawMessage, error)

	// CallAndWait executes a command and waits for job completion.
	CallAndWait(ctx context.Context, method string, params any) (json.RawMessage, error)

	// Close closes the connection.
	Close() error
}

// MockClient is a test double for Client.
type MockClient struct {
	CallFunc        func(ctx context.Context, method string, params any) (json.RawMessage, error)
	CallAndWaitFunc func(ctx context.Context, method string, params any) (json.RawMessage, error)
	CloseFunc       func() error
}

func (m *MockClient) Call(ctx context.Context, method string, params any) (json.RawMessage, error) {
	if m.CallFunc != nil {
		return m.CallFunc(ctx, method, params)
	}
	return nil, nil
}

func (m *MockClient) CallAndWait(ctx context.Context, method string, params any) (json.RawMessage, error) {
	if m.CallAndWaitFunc != nil {
		return m.CallAndWaitFunc(ctx, method, params)
	}
	return nil, nil
}

func (m *MockClient) Close() error {
	if m.CloseFunc != nil {
		return m.CloseFunc()
	}
	return nil
}
