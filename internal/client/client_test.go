package client

import (
	"context"
	"encoding/json"
	"testing"
)

func TestMockClient_Call(t *testing.T) {
	expected := json.RawMessage(`{"name": "test"}`)

	mock := &MockClient{
		CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
			if method != "app.query" {
				t.Errorf("expected method app.query, got %s", method)
			}
			return expected, nil
		},
	}

	result, err := mock.Call(context.Background(), "app.query", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if string(result) != string(expected) {
		t.Errorf("expected %s, got %s", expected, result)
	}
}

func TestMockClient_CallAndWait(t *testing.T) {
	expected := json.RawMessage(`{"status": "complete"}`)

	mock := &MockClient{
		CallAndWaitFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
			return expected, nil
		},
	}

	result, err := mock.CallAndWait(context.Background(), "app.create", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if string(result) != string(expected) {
		t.Errorf("expected %s, got %s", expected, result)
	}
}

func TestMockClient_Call_NilFunc(t *testing.T) {
	mock := &MockClient{}

	result, err := mock.Call(context.Background(), "app.query", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != nil {
		t.Errorf("expected nil result, got %s", result)
	}
}

func TestMockClient_CallAndWait_NilFunc(t *testing.T) {
	mock := &MockClient{}

	result, err := mock.CallAndWait(context.Background(), "app.create", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != nil {
		t.Errorf("expected nil result, got %s", result)
	}
}

func TestMockClient_Close(t *testing.T) {
	closeCalled := false
	mock := &MockClient{
		CloseFunc: func() error {
			closeCalled = true
			return nil
		},
	}

	err := mock.Close()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !closeCalled {
		t.Error("expected CloseFunc to be called")
	}
}

func TestMockClient_Close_NilFunc(t *testing.T) {
	mock := &MockClient{}

	err := mock.Close()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestMockClient_ImplementsSFTPMethods(t *testing.T) {
	mock := &MockClient{}

	// Verify SFTP methods exist on MockClient
	ctx := context.Background()

	_ = mock.WriteFile(ctx, "/test", []byte("content"), 0644)
	_, _ = mock.ReadFile(ctx, "/test")
	_ = mock.DeleteFile(ctx, "/test")
	_, _ = mock.FileExists(ctx, "/test")
	_ = mock.MkdirAll(ctx, "/test", 0755)
}
