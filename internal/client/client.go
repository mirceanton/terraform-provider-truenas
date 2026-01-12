package client

import (
	"context"
	"encoding/json"
	"io/fs"
)

// Client defines the interface for communicating with TrueNAS.
type Client interface {
	// Call executes a midclt command and returns the parsed JSON response.
	Call(ctx context.Context, method string, params any) (json.RawMessage, error)

	// CallAndWait executes a command and waits for job completion.
	CallAndWait(ctx context.Context, method string, params any) (json.RawMessage, error)

	// WriteFile writes content to a file on the remote system.
	WriteFile(ctx context.Context, path string, content []byte, mode fs.FileMode) error

	// ReadFile reads the content of a file from the remote system.
	ReadFile(ctx context.Context, path string) ([]byte, error)

	// DeleteFile removes a file from the remote system.
	DeleteFile(ctx context.Context, path string) error

	// RemoveDir removes an empty directory from the remote system.
	RemoveDir(ctx context.Context, path string) error

	// RemoveAll recursively removes a directory and all its contents.
	RemoveAll(ctx context.Context, path string) error

	// FileExists checks if a file exists on the remote system.
	FileExists(ctx context.Context, path string) (bool, error)

	// Chown changes the ownership of a file or directory.
	Chown(ctx context.Context, path string, uid, gid int) error

	// MkdirAll creates a directory and all parent directories.
	MkdirAll(ctx context.Context, path string, mode fs.FileMode) error

	// Close closes the connection.
	Close() error
}

// MockClient is a test double for Client.
type MockClient struct {
	CallFunc        func(ctx context.Context, method string, params any) (json.RawMessage, error)
	CallAndWaitFunc func(ctx context.Context, method string, params any) (json.RawMessage, error)
	WriteFileFunc   func(ctx context.Context, path string, content []byte, mode fs.FileMode) error
	ReadFileFunc    func(ctx context.Context, path string) ([]byte, error)
	DeleteFileFunc  func(ctx context.Context, path string) error
	RemoveDirFunc   func(ctx context.Context, path string) error
	RemoveAllFunc   func(ctx context.Context, path string) error
	FileExistsFunc  func(ctx context.Context, path string) (bool, error)
	ChownFunc       func(ctx context.Context, path string, uid, gid int) error
	MkdirAllFunc    func(ctx context.Context, path string, mode fs.FileMode) error
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

func (m *MockClient) WriteFile(ctx context.Context, path string, content []byte, mode fs.FileMode) error {
	if m.WriteFileFunc != nil {
		return m.WriteFileFunc(ctx, path, content, mode)
	}
	return nil
}

func (m *MockClient) ReadFile(ctx context.Context, path string) ([]byte, error) {
	if m.ReadFileFunc != nil {
		return m.ReadFileFunc(ctx, path)
	}
	return nil, nil
}

func (m *MockClient) DeleteFile(ctx context.Context, path string) error {
	if m.DeleteFileFunc != nil {
		return m.DeleteFileFunc(ctx, path)
	}
	return nil
}

func (m *MockClient) RemoveDir(ctx context.Context, path string) error {
	if m.RemoveDirFunc != nil {
		return m.RemoveDirFunc(ctx, path)
	}
	return nil
}

func (m *MockClient) RemoveAll(ctx context.Context, path string) error {
	if m.RemoveAllFunc != nil {
		return m.RemoveAllFunc(ctx, path)
	}
	return nil
}

func (m *MockClient) FileExists(ctx context.Context, path string) (bool, error) {
	if m.FileExistsFunc != nil {
		return m.FileExistsFunc(ctx, path)
	}
	return false, nil
}

func (m *MockClient) Chown(ctx context.Context, path string, uid, gid int) error {
	if m.ChownFunc != nil {
		return m.ChownFunc(ctx, path, uid, gid)
	}
	return nil
}

func (m *MockClient) MkdirAll(ctx context.Context, path string, mode fs.FileMode) error {
	if m.MkdirAllFunc != nil {
		return m.MkdirAllFunc(ctx, path, mode)
	}
	return nil
}

func (m *MockClient) Close() error {
	if m.CloseFunc != nil {
		return m.CloseFunc()
	}
	return nil
}
