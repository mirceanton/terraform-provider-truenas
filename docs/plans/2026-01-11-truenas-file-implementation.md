# truenas_file Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add a `truenas_file` resource to deploy configuration files to TrueNAS via SFTP.

**Architecture:** Extend the SSH client with SFTP capabilities using `github.com/pkg/sftp`. Create a new `file` resource that supports two path modes: `host_path` + `relative_path` (references managed directory) or standalone `path`. Use SHA256 checksums for drift detection.

**Tech Stack:** Go, Terraform Plugin Framework, `github.com/pkg/sftp`, `golang.org/x/crypto/ssh`

---

## Phase 1: SFTP Client Extension

### Task 1.1: Add SFTP Dependency

**Files:**
- Modify: `go.mod`

**Step 1: Add the sftp package**

```bash
go get github.com/pkg/sftp
```

**Step 2: Verify dependency added**

Run: `grep sftp go.mod`
Expected: `github.com/pkg/sftp v1.x.x`

**Step 3: Commit**

```bash
jj describe -m "chore: add github.com/pkg/sftp dependency"
jj new
```

---

### Task 1.2: Extend Client Interface

**Files:**
- Modify: `internal/client/client.go`
- Test: `internal/client/client_test.go`

**Step 1: Write failing test for new interface methods**

Add to `internal/client/client_test.go`:

```go
func TestMockClient_ImplementsSFTPMethods(t *testing.T) {
	mock := &MockClient{}

	// Verify SFTP methods exist on MockClient
	ctx := context.Background()

	_, _ = mock.WriteFile(ctx, "/test", []byte("content"), 0644)
	_, _ = mock.ReadFile(ctx, "/test")
	_ = mock.DeleteFile(ctx, "/test")
	_, _ = mock.FileExists(ctx, "/test")
	_ = mock.MkdirAll(ctx, "/test", 0755)
}
```

**Step 2: Run test to verify it fails**

Run: `mise run test`
Expected: FAIL - methods not defined on MockClient

**Step 3: Update Client interface and MockClient**

Replace `internal/client/client.go`:

```go
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

	// FileExists checks if a file exists on the remote system.
	FileExists(ctx context.Context, path string) (bool, error)

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
	FileExistsFunc  func(ctx context.Context, path string) (bool, error)
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

func (m *MockClient) FileExists(ctx context.Context, path string) (bool, error) {
	if m.FileExistsFunc != nil {
		return m.FileExistsFunc(ctx, path)
	}
	return false, nil
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
```

**Step 4: Run test to verify it passes**

Run: `mise run test`
Expected: PASS

**Step 5: Commit**

```bash
jj describe -m "feat(client): extend Client interface with SFTP methods"
jj new
```

---

### Task 1.3: Implement SFTP Methods - WriteFile

**Files:**
- Modify: `internal/client/ssh.go`
- Test: `internal/client/sftp_test.go` (create new file)

**Step 1: Write failing tests for WriteFile**

Create `internal/client/sftp_test.go`:

```go
package client

import (
	"context"
	"errors"
	"io/fs"
	"testing"
)

// mockSFTPClient is a test double for sftpClient interface
type mockSFTPClient struct {
	createFunc     func(path string) (sftpFile, error)
	mkdirAllFunc   func(path string) error
	statFunc       func(path string) (fs.FileInfo, error)
	removeFunc     func(path string) error
	openFunc       func(path string) (sftpFile, error)
	chmodFunc      func(path string, mode fs.FileMode) error
	closeFunc      func() error
}

func (m *mockSFTPClient) Create(path string) (sftpFile, error) {
	if m.createFunc != nil {
		return m.createFunc(path)
	}
	return nil, nil
}

func (m *mockSFTPClient) MkdirAll(path string) error {
	if m.mkdirAllFunc != nil {
		return m.mkdirAllFunc(path)
	}
	return nil
}

func (m *mockSFTPClient) Stat(path string) (fs.FileInfo, error) {
	if m.statFunc != nil {
		return m.statFunc(path)
	}
	return nil, nil
}

func (m *mockSFTPClient) Remove(path string) error {
	if m.removeFunc != nil {
		return m.removeFunc(path)
	}
	return nil
}

func (m *mockSFTPClient) Open(path string) (sftpFile, error) {
	if m.openFunc != nil {
		return m.openFunc(path)
	}
	return nil, nil
}

func (m *mockSFTPClient) Chmod(path string, mode fs.FileMode) error {
	if m.chmodFunc != nil {
		return m.chmodFunc(path, mode)
	}
	return nil
}

func (m *mockSFTPClient) Close() error {
	if m.closeFunc != nil {
		return m.closeFunc()
	}
	return nil
}

// mockSFTPFile is a test double for sftpFile interface
type mockSFTPFile struct {
	writeFunc func(p []byte) (int, error)
	readFunc  func(p []byte) (int, error)
	closeFunc func() error
}

func (m *mockSFTPFile) Write(p []byte) (int, error) {
	if m.writeFunc != nil {
		return m.writeFunc(p)
	}
	return len(p), nil
}

func (m *mockSFTPFile) Read(p []byte) (int, error) {
	if m.readFunc != nil {
		return m.readFunc(p)
	}
	return 0, nil
}

func (m *mockSFTPFile) Close() error {
	if m.closeFunc != nil {
		return m.closeFunc()
	}
	return nil
}

func TestSSHClient_WriteFile_Success(t *testing.T) {
	config := &SSHConfig{
		Host:       "truenas.local",
		PrivateKey: testPrivateKey,
	}

	client, _ := NewSSHClient(config)

	var writtenContent []byte
	var writtenPath string

	mockFile := &mockSFTPFile{
		writeFunc: func(p []byte) (int, error) {
			writtenContent = p
			return len(p), nil
		},
	}

	mockSFTP := &mockSFTPClient{
		createFunc: func(path string) (sftpFile, error) {
			writtenPath = path
			return mockFile, nil
		},
		chmodFunc: func(path string, mode fs.FileMode) error {
			return nil
		},
	}

	client.sftpClient = mockSFTP

	err := client.WriteFile(context.Background(), "/mnt/storage/test.txt", []byte("hello world"), 0644)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if writtenPath != "/mnt/storage/test.txt" {
		t.Errorf("expected path '/mnt/storage/test.txt', got %q", writtenPath)
	}

	if string(writtenContent) != "hello world" {
		t.Errorf("expected content 'hello world', got %q", string(writtenContent))
	}
}

func TestSSHClient_WriteFile_CreateError(t *testing.T) {
	config := &SSHConfig{
		Host:       "truenas.local",
		PrivateKey: testPrivateKey,
	}

	client, _ := NewSSHClient(config)

	mockSFTP := &mockSFTPClient{
		createFunc: func(path string) (sftpFile, error) {
			return nil, errors.New("permission denied")
		},
	}

	client.sftpClient = mockSFTP

	err := client.WriteFile(context.Background(), "/mnt/storage/test.txt", []byte("hello"), 0644)
	if err == nil {
		t.Fatal("expected error for create failure")
	}
}

func TestSSHClient_WriteFile_WriteError(t *testing.T) {
	config := &SSHConfig{
		Host:       "truenas.local",
		PrivateKey: testPrivateKey,
	}

	client, _ := NewSSHClient(config)

	mockFile := &mockSFTPFile{
		writeFunc: func(p []byte) (int, error) {
			return 0, errors.New("disk full")
		},
	}

	mockSFTP := &mockSFTPClient{
		createFunc: func(path string) (sftpFile, error) {
			return mockFile, nil
		},
	}

	client.sftpClient = mockSFTP

	err := client.WriteFile(context.Background(), "/mnt/storage/test.txt", []byte("hello"), 0644)
	if err == nil {
		t.Fatal("expected error for write failure")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `mise run test`
Expected: FAIL - sftpClient, sftpFile, WriteFile not defined

**Step 3: Implement SFTP interfaces and WriteFile**

Add to `internal/client/ssh.go` (after existing imports, add `"io/fs"` and `"github.com/pkg/sftp"`):

```go
// Add these imports to the import block:
// "io/fs"
// "github.com/pkg/sftp"

// sftpClient abstracts SFTP operations for testing.
type sftpClient interface {
	Create(path string) (sftpFile, error)
	MkdirAll(path string) error
	Stat(path string) (fs.FileInfo, error)
	Remove(path string) error
	Open(path string) (sftpFile, error)
	Chmod(path string, mode fs.FileMode) error
	Close() error
}

// sftpFile abstracts SFTP file operations for testing.
type sftpFile interface {
	Write(p []byte) (int, error)
	Read(p []byte) (int, error)
	Close() error
}

// realSFTPClient wraps *sftp.Client for the interface.
type realSFTPClient struct {
	client *sftp.Client
}

func (r *realSFTPClient) Create(path string) (sftpFile, error) {
	return r.client.Create(path)
}

func (r *realSFTPClient) MkdirAll(path string) error {
	return r.client.MkdirAll(path)
}

func (r *realSFTPClient) Stat(path string) (fs.FileInfo, error) {
	return r.client.Stat(path)
}

func (r *realSFTPClient) Remove(path string) error {
	return r.client.Remove(path)
}

func (r *realSFTPClient) Open(path string) (sftpFile, error) {
	return r.client.Open(path)
}

func (r *realSFTPClient) Chmod(path string, mode fs.FileMode) error {
	return r.client.Chmod(path, mode)
}

func (r *realSFTPClient) Close() error {
	return r.client.Close()
}
```

Add `sftpClient sftpClient` field to `SSHClient` struct:

```go
type SSHClient struct {
	config        *SSHConfig
	client        *ssh.Client
	clientWrapper sshClientWrapper
	sftpClient    sftpClient
	dialer        sshDialer
	mu            sync.Mutex
}
```

Add `connectSFTP` and `WriteFile` methods:

```go
// connectSFTP establishes the SFTP connection if not already connected.
func (c *SSHClient) connectSFTP() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Already have SFTP client
	if c.sftpClient != nil {
		return nil
	}

	// Ensure SSH is connected first (unlock mutex temporarily)
	c.mu.Unlock()
	if err := c.connect(); err != nil {
		c.mu.Lock()
		return err
	}
	c.mu.Lock()

	// Create SFTP client
	sftpConn, err := sftp.NewClient(c.client)
	if err != nil {
		return fmt.Errorf("failed to create SFTP client: %w", err)
	}

	c.sftpClient = &realSFTPClient{client: sftpConn}
	return nil
}

// WriteFile writes content to a file on the remote system.
func (c *SSHClient) WriteFile(ctx context.Context, path string, content []byte, mode fs.FileMode) error {
	// Connect SFTP if needed (skip if already mocked)
	if c.sftpClient == nil {
		if err := c.connectSFTP(); err != nil {
			return err
		}
	}

	// Create the file
	file, err := c.sftpClient.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create file %q: %w", path, err)
	}
	defer file.Close()

	// Write content
	_, err = file.Write(content)
	if err != nil {
		return fmt.Errorf("failed to write to file %q: %w", path, err)
	}

	// Set permissions
	if err := c.sftpClient.Chmod(path, mode); err != nil {
		return fmt.Errorf("failed to set permissions on %q: %w", path, err)
	}

	return nil
}
```

**Step 4: Run test to verify it passes**

Run: `mise run test`
Expected: PASS

**Step 5: Commit**

```bash
jj describe -m "feat(client): implement WriteFile SFTP method"
jj new
```

---

### Task 1.4: Implement SFTP Methods - ReadFile

**Files:**
- Modify: `internal/client/ssh.go`
- Test: `internal/client/sftp_test.go`

**Step 1: Write failing tests for ReadFile**

Add to `internal/client/sftp_test.go`:

```go
func TestSSHClient_ReadFile_Success(t *testing.T) {
	config := &SSHConfig{
		Host:       "truenas.local",
		PrivateKey: testPrivateKey,
	}

	client, _ := NewSSHClient(config)

	content := []byte("file content here")
	mockFile := &mockSFTPFile{
		readFunc: func(p []byte) (int, error) {
			copy(p, content)
			return len(content), nil
		},
	}

	mockSFTP := &mockSFTPClient{
		openFunc: func(path string) (sftpFile, error) {
			if path != "/mnt/storage/test.txt" {
				t.Errorf("expected path '/mnt/storage/test.txt', got %q", path)
			}
			return mockFile, nil
		},
		statFunc: func(path string) (fs.FileInfo, error) {
			return &mockFileInfo{size: int64(len(content))}, nil
		},
	}

	client.sftpClient = mockSFTP

	result, err := client.ReadFile(context.Background(), "/mnt/storage/test.txt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if string(result) != "file content here" {
		t.Errorf("expected 'file content here', got %q", string(result))
	}
}

func TestSSHClient_ReadFile_NotFound(t *testing.T) {
	config := &SSHConfig{
		Host:       "truenas.local",
		PrivateKey: testPrivateKey,
	}

	client, _ := NewSSHClient(config)

	mockSFTP := &mockSFTPClient{
		openFunc: func(path string) (sftpFile, error) {
			return nil, errors.New("file not found")
		},
	}

	client.sftpClient = mockSFTP

	_, err := client.ReadFile(context.Background(), "/mnt/storage/missing.txt")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

// mockFileInfo implements fs.FileInfo for testing
type mockFileInfo struct {
	size int64
}

func (m *mockFileInfo) Name() string       { return "test" }
func (m *mockFileInfo) Size() int64        { return m.size }
func (m *mockFileInfo) Mode() fs.FileMode  { return 0644 }
func (m *mockFileInfo) ModTime() time.Time { return time.Now() }
func (m *mockFileInfo) IsDir() bool        { return false }
func (m *mockFileInfo) Sys() any           { return nil }
```

Add `"time"` to imports in sftp_test.go.

**Step 2: Run test to verify it fails**

Run: `mise run test`
Expected: FAIL - ReadFile not defined

**Step 3: Implement ReadFile**

Add to `internal/client/ssh.go`:

```go
// ReadFile reads the content of a file from the remote system.
func (c *SSHClient) ReadFile(ctx context.Context, path string) ([]byte, error) {
	// Connect SFTP if needed (skip if already mocked)
	if c.sftpClient == nil {
		if err := c.connectSFTP(); err != nil {
			return nil, err
		}
	}

	// Get file size first
	info, err := c.sftpClient.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("failed to stat file %q: %w", path, err)
	}

	// Open the file
	file, err := c.sftpClient.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %q: %w", path, err)
	}
	defer file.Close()

	// Read content
	content := make([]byte, info.Size())
	_, err = file.Read(content)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %q: %w", path, err)
	}

	return content, nil
}
```

**Step 4: Run test to verify it passes**

Run: `mise run test`
Expected: PASS

**Step 5: Commit**

```bash
jj describe -m "feat(client): implement ReadFile SFTP method"
jj new
```

---

### Task 1.5: Implement SFTP Methods - DeleteFile, FileExists, MkdirAll

**Files:**
- Modify: `internal/client/ssh.go`
- Test: `internal/client/sftp_test.go`

**Step 1: Write failing tests**

Add to `internal/client/sftp_test.go`:

```go
func TestSSHClient_DeleteFile_Success(t *testing.T) {
	config := &SSHConfig{
		Host:       "truenas.local",
		PrivateKey: testPrivateKey,
	}

	client, _ := NewSSHClient(config)

	var deletedPath string
	mockSFTP := &mockSFTPClient{
		removeFunc: func(path string) error {
			deletedPath = path
			return nil
		},
	}

	client.sftpClient = mockSFTP

	err := client.DeleteFile(context.Background(), "/mnt/storage/test.txt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if deletedPath != "/mnt/storage/test.txt" {
		t.Errorf("expected path '/mnt/storage/test.txt', got %q", deletedPath)
	}
}

func TestSSHClient_DeleteFile_NotFound(t *testing.T) {
	config := &SSHConfig{
		Host:       "truenas.local",
		PrivateKey: testPrivateKey,
	}

	client, _ := NewSSHClient(config)

	mockSFTP := &mockSFTPClient{
		removeFunc: func(path string) error {
			return errors.New("no such file")
		},
	}

	client.sftpClient = mockSFTP

	err := client.DeleteFile(context.Background(), "/mnt/storage/missing.txt")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestSSHClient_FileExists_True(t *testing.T) {
	config := &SSHConfig{
		Host:       "truenas.local",
		PrivateKey: testPrivateKey,
	}

	client, _ := NewSSHClient(config)

	mockSFTP := &mockSFTPClient{
		statFunc: func(path string) (fs.FileInfo, error) {
			return &mockFileInfo{}, nil
		},
	}

	client.sftpClient = mockSFTP

	exists, err := client.FileExists(context.Background(), "/mnt/storage/test.txt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !exists {
		t.Error("expected file to exist")
	}
}

func TestSSHClient_FileExists_False(t *testing.T) {
	config := &SSHConfig{
		Host:       "truenas.local",
		PrivateKey: testPrivateKey,
	}

	client, _ := NewSSHClient(config)

	mockSFTP := &mockSFTPClient{
		statFunc: func(path string) (fs.FileInfo, error) {
			return nil, os.ErrNotExist
		},
	}

	client.sftpClient = mockSFTP

	exists, err := client.FileExists(context.Background(), "/mnt/storage/missing.txt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if exists {
		t.Error("expected file to not exist")
	}
}

func TestSSHClient_MkdirAll_Success(t *testing.T) {
	config := &SSHConfig{
		Host:       "truenas.local",
		PrivateKey: testPrivateKey,
	}

	client, _ := NewSSHClient(config)

	var createdPath string
	mockSFTP := &mockSFTPClient{
		mkdirAllFunc: func(path string) error {
			createdPath = path
			return nil
		},
	}

	client.sftpClient = mockSFTP

	err := client.MkdirAll(context.Background(), "/mnt/storage/apps/myapp/config", 0755)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if createdPath != "/mnt/storage/apps/myapp/config" {
		t.Errorf("expected path '/mnt/storage/apps/myapp/config', got %q", createdPath)
	}
}

func TestSSHClient_MkdirAll_PermissionDenied(t *testing.T) {
	config := &SSHConfig{
		Host:       "truenas.local",
		PrivateKey: testPrivateKey,
	}

	client, _ := NewSSHClient(config)

	mockSFTP := &mockSFTPClient{
		mkdirAllFunc: func(path string) error {
			return errors.New("permission denied")
		},
	}

	client.sftpClient = mockSFTP

	err := client.MkdirAll(context.Background(), "/mnt/storage/protected", 0755)
	if err == nil {
		t.Fatal("expected error for permission denied")
	}
}
```

Add `"os"` to imports in sftp_test.go.

**Step 2: Run test to verify it fails**

Run: `mise run test`
Expected: FAIL - DeleteFile, FileExists, MkdirAll not defined

**Step 3: Implement remaining SFTP methods**

Add to `internal/client/ssh.go`:

```go
// DeleteFile removes a file from the remote system.
func (c *SSHClient) DeleteFile(ctx context.Context, path string) error {
	// Connect SFTP if needed (skip if already mocked)
	if c.sftpClient == nil {
		if err := c.connectSFTP(); err != nil {
			return err
		}
	}

	if err := c.sftpClient.Remove(path); err != nil {
		return fmt.Errorf("failed to delete file %q: %w", path, err)
	}

	return nil
}

// FileExists checks if a file exists on the remote system.
func (c *SSHClient) FileExists(ctx context.Context, path string) (bool, error) {
	// Connect SFTP if needed (skip if already mocked)
	if c.sftpClient == nil {
		if err := c.connectSFTP(); err != nil {
			return false, err
		}
	}

	_, err := c.sftpClient.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to stat file %q: %w", path, err)
	}

	return true, nil
}

// MkdirAll creates a directory and all parent directories.
func (c *SSHClient) MkdirAll(ctx context.Context, path string, mode fs.FileMode) error {
	// Connect SFTP if needed (skip if already mocked)
	if c.sftpClient == nil {
		if err := c.connectSFTP(); err != nil {
			return err
		}
	}

	if err := c.sftpClient.MkdirAll(path); err != nil {
		return fmt.Errorf("failed to create directory %q: %w", path, err)
	}

	return nil
}
```

Add `"os"` to imports in ssh.go.

**Step 4: Run test to verify it passes**

Run: `mise run test`
Expected: PASS

**Step 5: Commit**

```bash
jj describe -m "feat(client): implement DeleteFile, FileExists, MkdirAll SFTP methods"
jj new
```

---

## Phase 2: File Resource Schema & Validation

### Task 2.1: Create File Resource Scaffold

**Files:**
- Create: `internal/resources/file.go`
- Test: `internal/resources/file_test.go`

**Step 1: Write failing tests for resource scaffold**

Create `internal/resources/file_test.go`:

```go
package resources

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func TestNewFileResource(t *testing.T) {
	r := NewFileResource()
	if r == nil {
		t.Fatal("expected non-nil resource")
	}

	// Verify it implements the required interfaces
	var _ resource.Resource = r
	var _ resource.ResourceWithConfigure = r.(*FileResource)
	var _ resource.ResourceWithImportState = r.(*FileResource)
	var _ resource.ResourceWithValidateConfig = r.(*FileResource)
}

func TestFileResource_Metadata(t *testing.T) {
	r := NewFileResource()

	req := resource.MetadataRequest{
		ProviderTypeName: "truenas",
	}
	resp := &resource.MetadataResponse{}

	r.Metadata(context.Background(), req, resp)

	if resp.TypeName != "truenas_file" {
		t.Errorf("expected TypeName 'truenas_file', got %q", resp.TypeName)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `mise run test`
Expected: FAIL - FileResource not defined

**Step 3: Create file resource scaffold**

Create `internal/resources/file.go`:

```go
package resources

import (
	"context"
	"fmt"

	"github.com/deevus/terraform-provider-truenas/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &FileResource{}
var _ resource.ResourceWithConfigure = &FileResource{}
var _ resource.ResourceWithImportState = &FileResource{}
var _ resource.ResourceWithValidateConfig = &FileResource{}

// FileResource defines the resource implementation.
type FileResource struct {
	client client.Client
}

// FileResourceModel describes the resource data model.
type FileResourceModel struct {
	ID           types.String `tfsdk:"id"`
	HostPath     types.String `tfsdk:"host_path"`
	RelativePath types.String `tfsdk:"relative_path"`
	Path         types.String `tfsdk:"path"`
	Content      types.String `tfsdk:"content"`
	Mode         types.String `tfsdk:"mode"`
	UID          types.Int64  `tfsdk:"uid"`
	GID          types.Int64  `tfsdk:"gid"`
	Checksum     types.String `tfsdk:"checksum"`
}

// NewFileResource creates a new FileResource.
func NewFileResource() resource.Resource {
	return &FileResource{}
}

func (r *FileResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_file"
}

func (r *FileResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a file on TrueNAS for configuration deployment.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "File identifier (the full path).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"host_path": schema.StringAttribute{
				Description: "ID of a truenas_host_path resource. Mutually exclusive with 'path'.",
				Optional:    true,
			},
			"relative_path": schema.StringAttribute{
				Description: "Path relative to host_path. Can include subdirectories (e.g., 'config/app.conf').",
				Optional:    true,
			},
			"path": schema.StringAttribute{
				Description: "Absolute path to the file. Mutually exclusive with 'host_path'/'relative_path'.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"content": schema.StringAttribute{
				Description: "Content of the file. Use templatefile() or file() to load from disk.",
				Required:    true,
				Sensitive:   true,
			},
			"mode": schema.StringAttribute{
				Description: "Unix mode (e.g., '0644'). Inherits from host_path if not specified.",
				Optional:    true,
				Computed:    true,
			},
			"uid": schema.Int64Attribute{
				Description: "Owner user ID. Inherits from host_path if not specified.",
				Optional:    true,
				Computed:    true,
			},
			"gid": schema.Int64Attribute{
				Description: "Owner group ID. Inherits from host_path if not specified.",
				Optional:    true,
				Computed:    true,
			},
			"checksum": schema.StringAttribute{
				Description: "SHA256 checksum of the file content.",
				Computed:    true,
			},
		},
	}
}

func (r *FileResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	c, ok := req.ProviderData.(client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected client.Client, got: %T.", req.ProviderData),
		)
		return
	}

	r.client = c
}

func (r *FileResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data FileResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Validation logic will be added in next task
}

func (r *FileResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Will be implemented in Phase 3
}

func (r *FileResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Will be implemented in Phase 3
}

func (r *FileResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Will be implemented in Phase 3
}

func (r *FileResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Will be implemented in Phase 3
}

func (r *FileResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
```

**Step 4: Run test to verify it passes**

Run: `mise run test`
Expected: PASS

**Step 5: Commit**

```bash
jj describe -m "feat(resources): add file resource scaffold"
jj new
```

---

### Task 2.2: Implement Schema Validation

**Files:**
- Modify: `internal/resources/file.go`
- Test: `internal/resources/file_test.go`

**Step 1: Write failing tests for validation**

Add to `internal/resources/file_test.go`:

```go
func TestFileResource_Schema(t *testing.T) {
	r := NewFileResource()

	req := resource.SchemaRequest{}
	resp := &resource.SchemaResponse{}

	r.Schema(context.Background(), req, resp)

	// Verify required attributes
	contentAttr, ok := resp.Schema.Attributes["content"]
	if !ok {
		t.Fatal("expected 'content' attribute")
	}
	if !contentAttr.IsRequired() {
		t.Error("expected 'content' to be required")
	}

	// Verify optional attributes
	for _, attr := range []string{"host_path", "relative_path", "path", "mode", "uid", "gid"} {
		a, ok := resp.Schema.Attributes[attr]
		if !ok {
			t.Fatalf("expected '%s' attribute", attr)
		}
		if !a.IsOptional() {
			t.Errorf("expected '%s' to be optional", attr)
		}
	}

	// Verify computed attributes
	for _, attr := range []string{"id", "checksum"} {
		a, ok := resp.Schema.Attributes[attr]
		if !ok {
			t.Fatalf("expected '%s' attribute", attr)
		}
		if !a.IsComputed() {
			t.Errorf("expected '%s' to be computed", attr)
		}
	}
}

func TestFileResource_ValidateConfig_HostPathAndRelativePath(t *testing.T) {
	r := NewFileResource().(*FileResource)

	schemaResp := getFileResourceSchema(t)

	// Valid: host_path + relative_path
	configValue := createFileResourceModel(nil, "/mnt/storage/apps/myapp", "config/app.conf", nil, "content", nil, nil, nil, nil)

	req := resource.ValidateConfigRequest{
		Config: tfsdk.Config{
			Schema: schemaResp.Schema,
			Raw:    configValue,
		},
	}
	resp := &resource.ValidateConfigResponse{}

	r.ValidateConfig(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}
}

func TestFileResource_ValidateConfig_StandalonePath(t *testing.T) {
	r := NewFileResource().(*FileResource)

	schemaResp := getFileResourceSchema(t)

	// Valid: standalone path
	configValue := createFileResourceModel(nil, nil, nil, "/mnt/storage/apps/myapp/config.txt", "content", nil, nil, nil, nil)

	req := resource.ValidateConfigRequest{
		Config: tfsdk.Config{
			Schema: schemaResp.Schema,
			Raw:    configValue,
		},
	}
	resp := &resource.ValidateConfigResponse{}

	r.ValidateConfig(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}
}

func TestFileResource_ValidateConfig_BothHostPathAndPath(t *testing.T) {
	r := NewFileResource().(*FileResource)

	schemaResp := getFileResourceSchema(t)

	// Invalid: both host_path and path specified
	configValue := createFileResourceModel(nil, "/mnt/storage/apps/myapp", "config.txt", "/mnt/other/path", "content", nil, nil, nil, nil)

	req := resource.ValidateConfigRequest{
		Config: tfsdk.Config{
			Schema: schemaResp.Schema,
			Raw:    configValue,
		},
	}
	resp := &resource.ValidateConfigResponse{}

	r.ValidateConfig(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error when both host_path and path are specified")
	}
}

func TestFileResource_ValidateConfig_NeitherHostPathNorPath(t *testing.T) {
	r := NewFileResource().(*FileResource)

	schemaResp := getFileResourceSchema(t)

	// Invalid: neither host_path nor path specified
	configValue := createFileResourceModel(nil, nil, nil, nil, "content", nil, nil, nil, nil)

	req := resource.ValidateConfigRequest{
		Config: tfsdk.Config{
			Schema: schemaResp.Schema,
			Raw:    configValue,
		},
	}
	resp := &resource.ValidateConfigResponse{}

	r.ValidateConfig(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error when neither host_path nor path is specified")
	}
}

func TestFileResource_ValidateConfig_RelativePathWithoutHostPath(t *testing.T) {
	r := NewFileResource().(*FileResource)

	schemaResp := getFileResourceSchema(t)

	// Invalid: relative_path without host_path
	configValue := createFileResourceModel(nil, nil, "config.txt", nil, "content", nil, nil, nil, nil)

	req := resource.ValidateConfigRequest{
		Config: tfsdk.Config{
			Schema: schemaResp.Schema,
			Raw:    configValue,
		},
	}
	resp := &resource.ValidateConfigResponse{}

	r.ValidateConfig(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error when relative_path is specified without host_path")
	}
}

func TestFileResource_ValidateConfig_RelativePathStartsWithSlash(t *testing.T) {
	r := NewFileResource().(*FileResource)

	schemaResp := getFileResourceSchema(t)

	// Invalid: relative_path starts with /
	configValue := createFileResourceModel(nil, "/mnt/storage/apps", "/config.txt", nil, "content", nil, nil, nil, nil)

	req := resource.ValidateConfigRequest{
		Config: tfsdk.Config{
			Schema: schemaResp.Schema,
			Raw:    configValue,
		},
	}
	resp := &resource.ValidateConfigResponse{}

	r.ValidateConfig(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error when relative_path starts with /")
	}
}

func TestFileResource_ValidateConfig_RelativePathContainsDoubleDot(t *testing.T) {
	r := NewFileResource().(*FileResource)

	schemaResp := getFileResourceSchema(t)

	// Invalid: relative_path contains ..
	configValue := createFileResourceModel(nil, "/mnt/storage/apps", "../etc/passwd", nil, "content", nil, nil, nil, nil)

	req := resource.ValidateConfigRequest{
		Config: tfsdk.Config{
			Schema: schemaResp.Schema,
			Raw:    configValue,
		},
	}
	resp := &resource.ValidateConfigResponse{}

	r.ValidateConfig(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error when relative_path contains ..")
	}
}

func TestFileResource_ValidateConfig_PathNotAbsolute(t *testing.T) {
	r := NewFileResource().(*FileResource)

	schemaResp := getFileResourceSchema(t)

	// Invalid: path is not absolute
	configValue := createFileResourceModel(nil, nil, nil, "relative/path.txt", "content", nil, nil, nil, nil)

	req := resource.ValidateConfigRequest{
		Config: tfsdk.Config{
			Schema: schemaResp.Schema,
			Raw:    configValue,
		},
	}
	resp := &resource.ValidateConfigResponse{}

	r.ValidateConfig(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error when path is not absolute")
	}
}

func TestFileResource_ValidateConfig_HostPathWithoutRelativePath(t *testing.T) {
	r := NewFileResource().(*FileResource)

	schemaResp := getFileResourceSchema(t)

	// Invalid: host_path without relative_path
	configValue := createFileResourceModel(nil, "/mnt/storage/apps", nil, nil, "content", nil, nil, nil, nil)

	req := resource.ValidateConfigRequest{
		Config: tfsdk.Config{
			Schema: schemaResp.Schema,
			Raw:    configValue,
		},
	}
	resp := &resource.ValidateConfigResponse{}

	r.ValidateConfig(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error when host_path is specified without relative_path")
	}
}

// Helper functions

func getFileResourceSchema(t *testing.T) resource.SchemaResponse {
	t.Helper()
	r := NewFileResource()
	schemaReq := resource.SchemaRequest{}
	schemaResp := &resource.SchemaResponse{}
	r.Schema(context.Background(), schemaReq, schemaResp)
	return *schemaResp
}

func createFileResourceModel(id, hostPath, relativePath, path, content, mode, uid, gid, checksum interface{}) tftypes.Value {
	return tftypes.NewValue(tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"id":            tftypes.String,
			"host_path":     tftypes.String,
			"relative_path": tftypes.String,
			"path":          tftypes.String,
			"content":       tftypes.String,
			"mode":          tftypes.String,
			"uid":           tftypes.Number,
			"gid":           tftypes.Number,
			"checksum":      tftypes.String,
		},
	}, map[string]tftypes.Value{
		"id":            tftypes.NewValue(tftypes.String, id),
		"host_path":     tftypes.NewValue(tftypes.String, hostPath),
		"relative_path": tftypes.NewValue(tftypes.String, relativePath),
		"path":          tftypes.NewValue(tftypes.String, path),
		"content":       tftypes.NewValue(tftypes.String, content),
		"mode":          tftypes.NewValue(tftypes.String, mode),
		"uid":           tftypes.NewValue(tftypes.Number, uid),
		"gid":           tftypes.NewValue(tftypes.Number, gid),
		"checksum":      tftypes.NewValue(tftypes.String, checksum),
	})
}
```

Add these imports to file_test.go:

```go
import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)
```

**Step 2: Run test to verify it fails**

Run: `mise run test`
Expected: FAIL - validation not implemented

**Step 3: Implement validation logic**

Update `ValidateConfig` in `internal/resources/file.go`:

```go
func (r *FileResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data FileResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	hasHostPath := !data.HostPath.IsNull() && !data.HostPath.IsUnknown()
	hasRelativePath := !data.RelativePath.IsNull() && !data.RelativePath.IsUnknown()
	hasPath := !data.Path.IsNull() && !data.Path.IsUnknown()

	// Must provide either host_path+relative_path OR path
	if hasHostPath && hasPath {
		resp.Diagnostics.AddError(
			"Invalid Configuration",
			"Cannot specify both 'host_path' and 'path'. Use one or the other.",
		)
		return
	}

	if !hasHostPath && !hasPath {
		resp.Diagnostics.AddError(
			"Invalid Configuration",
			"Must specify either 'host_path' with 'relative_path', or 'path'.",
		)
		return
	}

	// If host_path is specified, relative_path is required
	if hasHostPath && !hasRelativePath {
		resp.Diagnostics.AddError(
			"Invalid Configuration",
			"'relative_path' is required when 'host_path' is specified.",
		)
		return
	}

	// If relative_path is specified, host_path is required
	if hasRelativePath && !hasHostPath {
		resp.Diagnostics.AddError(
			"Invalid Configuration",
			"'host_path' is required when 'relative_path' is specified.",
		)
		return
	}

	// Validate relative_path format
	if hasRelativePath {
		relativePath := data.RelativePath.ValueString()

		if strings.HasPrefix(relativePath, "/") {
			resp.Diagnostics.AddError(
				"Invalid Configuration",
				"'relative_path' must not start with '/'. It should be relative to host_path.",
			)
			return
		}

		if strings.Contains(relativePath, "..") {
			resp.Diagnostics.AddError(
				"Invalid Configuration",
				"'relative_path' must not contain '..' (path traversal not allowed).",
			)
			return
		}
	}

	// Validate path is absolute
	if hasPath {
		p := data.Path.ValueString()
		if !strings.HasPrefix(p, "/") {
			resp.Diagnostics.AddError(
				"Invalid Configuration",
				"'path' must be an absolute path (start with '/').",
			)
			return
		}
	}
}
```

Add `"strings"` to imports.

**Step 4: Run test to verify it passes**

Run: `mise run test`
Expected: PASS

**Step 5: Commit**

```bash
jj describe -m "feat(resources): implement file resource validation"
jj new
```

---

## Phase 3: File Resource CRUD Operations

### Task 3.1: Implement Create Operation

**Files:**
- Modify: `internal/resources/file.go`
- Test: `internal/resources/file_test.go`

**Step 1: Write failing tests for Create**

Add to `internal/resources/file_test.go`:

```go
func TestFileResource_Create_WithHostPath(t *testing.T) {
	var writtenPath string
	var writtenContent []byte
	var mkdirPath string

	r := &FileResource{
		client: &client.MockClient{
			MkdirAllFunc: func(ctx context.Context, path string, mode fs.FileMode) error {
				mkdirPath = path
				return nil
			},
			WriteFileFunc: func(ctx context.Context, path string, content []byte, mode fs.FileMode) error {
				writtenPath = path
				writtenContent = content
				return nil
			},
		},
	}

	schemaResp := getFileResourceSchema(t)

	planValue := createFileResourceModel(nil, "/mnt/storage/apps/myapp", "config/app.conf", nil, "hello world", "0644", 0, 0, nil)

	req := resource.CreateRequest{
		Plan: tfsdk.Plan{
			Schema: schemaResp.Schema,
			Raw:    planValue,
		},
	}

	resp := &resource.CreateResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
		},
	}

	r.Create(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	// Verify mkdir was called for parent directory
	expectedMkdir := "/mnt/storage/apps/myapp/config"
	if mkdirPath != expectedMkdir {
		t.Errorf("expected mkdir path %q, got %q", expectedMkdir, mkdirPath)
	}

	// Verify file was written
	expectedPath := "/mnt/storage/apps/myapp/config/app.conf"
	if writtenPath != expectedPath {
		t.Errorf("expected path %q, got %q", expectedPath, writtenPath)
	}

	if string(writtenContent) != "hello world" {
		t.Errorf("expected content 'hello world', got %q", string(writtenContent))
	}

	// Verify state
	var model FileResourceModel
	diags := resp.State.Get(context.Background(), &model)
	if diags.HasError() {
		t.Fatalf("failed to get state: %v", diags)
	}

	if model.Path.ValueString() != expectedPath {
		t.Errorf("expected state path %q, got %q", expectedPath, model.Path.ValueString())
	}
}

func TestFileResource_Create_WithStandalonePath(t *testing.T) {
	var writtenPath string

	r := &FileResource{
		client: &client.MockClient{
			WriteFileFunc: func(ctx context.Context, path string, content []byte, mode fs.FileMode) error {
				writtenPath = path
				return nil
			},
		},
	}

	schemaResp := getFileResourceSchema(t)

	planValue := createFileResourceModel(nil, nil, nil, "/mnt/storage/existing/config.txt", "content", "0644", 0, 0, nil)

	req := resource.CreateRequest{
		Plan: tfsdk.Plan{
			Schema: schemaResp.Schema,
			Raw:    planValue,
		},
	}

	resp := &resource.CreateResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
		},
	}

	r.Create(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	if writtenPath != "/mnt/storage/existing/config.txt" {
		t.Errorf("expected path '/mnt/storage/existing/config.txt', got %q", writtenPath)
	}
}

func TestFileResource_Create_WriteError(t *testing.T) {
	r := &FileResource{
		client: &client.MockClient{
			MkdirAllFunc: func(ctx context.Context, path string, mode fs.FileMode) error {
				return nil
			},
			WriteFileFunc: func(ctx context.Context, path string, content []byte, mode fs.FileMode) error {
				return errors.New("permission denied")
			},
		},
	}

	schemaResp := getFileResourceSchema(t)

	planValue := createFileResourceModel(nil, "/mnt/storage/apps", "config.txt", nil, "content", "0644", 0, 0, nil)

	req := resource.CreateRequest{
		Plan: tfsdk.Plan{
			Schema: schemaResp.Schema,
			Raw:    planValue,
		},
	}

	resp := &resource.CreateResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
		},
	}

	r.Create(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for write failure")
	}
}
```

Add these imports to file_test.go:

```go
import (
	"context"
	"errors"
	"io/fs"
	"testing"

	"github.com/deevus/terraform-provider-truenas/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)
```

**Step 2: Run test to verify it fails**

Run: `mise run test`
Expected: FAIL - Create not implemented

**Step 3: Implement Create**

Add helper function and update Create in `internal/resources/file.go`:

```go
import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/deevus/terraform-provider-truenas/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// computeChecksum calculates SHA256 checksum of content.
func computeChecksum(content string) string {
	hash := sha256.Sum256([]byte(content))
	return hex.EncodeToString(hash[:])
}

// resolvePath resolves the full path from host_path + relative_path or standalone path.
func (r *FileResource) resolvePath(data *FileResourceModel) string {
	if !data.HostPath.IsNull() && !data.HostPath.IsUnknown() {
		return filepath.Join(data.HostPath.ValueString(), data.RelativePath.ValueString())
	}
	return data.Path.ValueString()
}

// parseMode converts mode string to fs.FileMode.
func parseMode(mode string) fs.FileMode {
	if mode == "" {
		return 0644
	}
	m, err := strconv.ParseUint(mode, 8, 32)
	if err != nil {
		return 0644
	}
	return fs.FileMode(m)
}

func (r *FileResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data FileResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	fullPath := r.resolvePath(&data)
	content := data.Content.ValueString()
	mode := parseMode(data.Mode.ValueString())

	// If using host_path + relative_path, create parent directories
	if !data.HostPath.IsNull() && !data.HostPath.IsUnknown() {
		parentDir := filepath.Dir(fullPath)
		if err := r.client.MkdirAll(ctx, parentDir, 0755); err != nil {
			resp.Diagnostics.AddError(
				"Unable to Create Parent Directory",
				fmt.Sprintf("Unable to create directory %q: %s", parentDir, err.Error()),
			)
			return
		}
	}

	// Write the file
	if err := r.client.WriteFile(ctx, fullPath, []byte(content), mode); err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create File",
			fmt.Sprintf("Unable to write file %q: %s", fullPath, err.Error()),
		)
		return
	}

	// Set computed values
	data.ID = types.StringValue(fullPath)
	data.Path = types.StringValue(fullPath)
	data.Checksum = types.StringValue(computeChecksum(content))

	// Set defaults for mode/uid/gid if not specified
	if data.Mode.IsNull() || data.Mode.IsUnknown() {
		data.Mode = types.StringValue("0644")
	}
	if data.UID.IsNull() || data.UID.IsUnknown() {
		data.UID = types.Int64Value(0)
	}
	if data.GID.IsNull() || data.GID.IsUnknown() {
		data.GID = types.Int64Value(0)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
```

**Step 4: Run test to verify it passes**

Run: `mise run test`
Expected: PASS

**Step 5: Commit**

```bash
jj describe -m "feat(resources): implement file resource Create operation"
jj new
```

---

### Task 3.2: Implement Read Operation

**Files:**
- Modify: `internal/resources/file.go`
- Test: `internal/resources/file_test.go`

**Step 1: Write failing tests for Read**

Add to `internal/resources/file_test.go`:

```go
func TestFileResource_Read_Success(t *testing.T) {
	content := "file content"
	checksum := computeChecksum(content)

	r := &FileResource{
		client: &client.MockClient{
			FileExistsFunc: func(ctx context.Context, path string) (bool, error) {
				return true, nil
			},
			ReadFileFunc: func(ctx context.Context, path string) ([]byte, error) {
				return []byte(content), nil
			},
		},
	}

	schemaResp := getFileResourceSchema(t)

	stateValue := createFileResourceModel("/mnt/storage/test.txt", nil, nil, "/mnt/storage/test.txt", content, "0644", 0, 0, checksum)

	req := resource.ReadRequest{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
			Raw:    stateValue,
		},
	}

	resp := &resource.ReadResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
		},
	}

	r.Read(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	var model FileResourceModel
	diags := resp.State.Get(context.Background(), &model)
	if diags.HasError() {
		t.Fatalf("failed to get state: %v", diags)
	}

	if model.Checksum.ValueString() != checksum {
		t.Errorf("expected checksum %q, got %q", checksum, model.Checksum.ValueString())
	}
}

func TestFileResource_Read_FileNotFound(t *testing.T) {
	r := &FileResource{
		client: &client.MockClient{
			FileExistsFunc: func(ctx context.Context, path string) (bool, error) {
				return false, nil
			},
		},
	}

	schemaResp := getFileResourceSchema(t)

	stateValue := createFileResourceModel("/mnt/storage/test.txt", nil, nil, "/mnt/storage/test.txt", "content", "0644", 0, 0, "checksum")

	req := resource.ReadRequest{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
			Raw:    stateValue,
		},
	}

	resp := &resource.ReadResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
		},
	}

	r.Read(context.Background(), req, resp)

	// Should not error, just remove from state
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	// State should be null (removed)
	if !resp.State.Raw.IsNull() {
		t.Error("expected state to be removed when file not found")
	}
}

func TestFileResource_Read_DriftDetection(t *testing.T) {
	// Remote content is different from state
	remoteContent := "modified content"

	r := &FileResource{
		client: &client.MockClient{
			FileExistsFunc: func(ctx context.Context, path string) (bool, error) {
				return true, nil
			},
			ReadFileFunc: func(ctx context.Context, path string) ([]byte, error) {
				return []byte(remoteContent), nil
			},
		},
	}

	schemaResp := getFileResourceSchema(t)

	// State has old content/checksum
	stateValue := createFileResourceModel("/mnt/storage/test.txt", nil, nil, "/mnt/storage/test.txt", "old content", "0644", 0, 0, "old-checksum")

	req := resource.ReadRequest{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
			Raw:    stateValue,
		},
	}

	resp := &resource.ReadResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
		},
	}

	r.Read(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	var model FileResourceModel
	diags := resp.State.Get(context.Background(), &model)
	if diags.HasError() {
		t.Fatalf("failed to get state: %v", diags)
	}

	// Checksum should be updated to match remote
	expectedChecksum := computeChecksum(remoteContent)
	if model.Checksum.ValueString() != expectedChecksum {
		t.Errorf("expected checksum %q, got %q", expectedChecksum, model.Checksum.ValueString())
	}
}

// Helper to compute checksum in tests
func computeChecksum(content string) string {
	hash := sha256.Sum256([]byte(content))
	return hex.EncodeToString(hash[:])
}
```

Add `"crypto/sha256"` and `"encoding/hex"` to imports.

**Step 2: Run test to verify it fails**

Run: `mise run test`
Expected: FAIL - Read not implemented

**Step 3: Implement Read**

Update Read in `internal/resources/file.go`:

```go
func (r *FileResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data FileResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	fullPath := data.Path.ValueString()

	// Check if file exists
	exists, err := r.client.FileExists(ctx, fullPath)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Check File",
			fmt.Sprintf("Unable to check if file %q exists: %s", fullPath, err.Error()),
		)
		return
	}

	if !exists {
		// File was deleted externally, remove from state
		resp.State.RemoveResource(ctx)
		return
	}

	// Read file content for checksum calculation
	content, err := r.client.ReadFile(ctx, fullPath)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read File",
			fmt.Sprintf("Unable to read file %q: %s", fullPath, err.Error()),
		)
		return
	}

	// Update checksum to reflect actual remote state
	data.Checksum = types.StringValue(computeChecksum(string(content)))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
```

**Step 4: Run test to verify it passes**

Run: `mise run test`
Expected: PASS

**Step 5: Commit**

```bash
jj describe -m "feat(resources): implement file resource Read operation"
jj new
```

---

### Task 3.3: Implement Update and Delete Operations

**Files:**
- Modify: `internal/resources/file.go`
- Test: `internal/resources/file_test.go`

**Step 1: Write failing tests**

Add to `internal/resources/file_test.go`:

```go
func TestFileResource_Update_ContentChange(t *testing.T) {
	var writtenContent []byte

	r := &FileResource{
		client: &client.MockClient{
			WriteFileFunc: func(ctx context.Context, path string, content []byte, mode fs.FileMode) error {
				writtenContent = content
				return nil
			},
		},
	}

	schemaResp := getFileResourceSchema(t)

	oldChecksum := computeChecksum("old content")
	newChecksum := computeChecksum("new content")

	stateValue := createFileResourceModel("/mnt/storage/test.txt", nil, nil, "/mnt/storage/test.txt", "old content", "0644", 0, 0, oldChecksum)
	planValue := createFileResourceModel("/mnt/storage/test.txt", nil, nil, "/mnt/storage/test.txt", "new content", "0644", 0, 0, nil)

	req := resource.UpdateRequest{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
			Raw:    stateValue,
		},
		Plan: tfsdk.Plan{
			Schema: schemaResp.Schema,
			Raw:    planValue,
		},
	}

	resp := &resource.UpdateResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
		},
	}

	r.Update(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	if string(writtenContent) != "new content" {
		t.Errorf("expected content 'new content', got %q", string(writtenContent))
	}

	var model FileResourceModel
	diags := resp.State.Get(context.Background(), &model)
	if diags.HasError() {
		t.Fatalf("failed to get state: %v", diags)
	}

	if model.Checksum.ValueString() != newChecksum {
		t.Errorf("expected checksum %q, got %q", newChecksum, model.Checksum.ValueString())
	}
}

func TestFileResource_Delete_Success(t *testing.T) {
	var deletedPath string

	r := &FileResource{
		client: &client.MockClient{
			DeleteFileFunc: func(ctx context.Context, path string) error {
				deletedPath = path
				return nil
			},
		},
	}

	schemaResp := getFileResourceSchema(t)

	stateValue := createFileResourceModel("/mnt/storage/test.txt", nil, nil, "/mnt/storage/test.txt", "content", "0644", 0, 0, "checksum")

	req := resource.DeleteRequest{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
			Raw:    stateValue,
		},
	}

	resp := &resource.DeleteResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
		},
	}

	r.Delete(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	if deletedPath != "/mnt/storage/test.txt" {
		t.Errorf("expected path '/mnt/storage/test.txt', got %q", deletedPath)
	}
}

func TestFileResource_Delete_Error(t *testing.T) {
	r := &FileResource{
		client: &client.MockClient{
			DeleteFileFunc: func(ctx context.Context, path string) error {
				return errors.New("permission denied")
			},
		},
	}

	schemaResp := getFileResourceSchema(t)

	stateValue := createFileResourceModel("/mnt/storage/test.txt", nil, nil, "/mnt/storage/test.txt", "content", "0644", 0, 0, "checksum")

	req := resource.DeleteRequest{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
			Raw:    stateValue,
		},
	}

	resp := &resource.DeleteResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
		},
	}

	r.Delete(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for delete failure")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `mise run test`
Expected: FAIL - Update, Delete not implemented

**Step 3: Implement Update and Delete**

Update `internal/resources/file.go`:

```go
func (r *FileResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data FileResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	fullPath := r.resolvePath(&data)
	content := data.Content.ValueString()
	mode := parseMode(data.Mode.ValueString())

	// Write the updated file
	if err := r.client.WriteFile(ctx, fullPath, []byte(content), mode); err != nil {
		resp.Diagnostics.AddError(
			"Unable to Update File",
			fmt.Sprintf("Unable to write file %q: %s", fullPath, err.Error()),
		)
		return
	}

	// Update checksum
	data.Checksum = types.StringValue(computeChecksum(content))
	data.Path = types.StringValue(fullPath)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *FileResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data FileResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	fullPath := data.Path.ValueString()

	if err := r.client.DeleteFile(ctx, fullPath); err != nil {
		resp.Diagnostics.AddError(
			"Unable to Delete File",
			fmt.Sprintf("Unable to delete file %q: %s", fullPath, err.Error()),
		)
		return
	}
}
```

**Step 4: Run test to verify it passes**

Run: `mise run test`
Expected: PASS

**Step 5: Commit**

```bash
jj describe -m "feat(resources): implement file resource Update and Delete operations"
jj new
```

---

## Phase 4: Provider Integration

### Task 4.1: Register File Resource

**Files:**
- Modify: `internal/provider/provider.go`
- Test: `internal/provider/provider_test.go`

**Step 1: Write failing test**

Add to `internal/provider/provider_test.go`:

```go
func TestTrueNASProvider_Resources_IncludesFile(t *testing.T) {
	p := New("test")()

	resources := p.Resources(context.Background())

	// Find file resource
	found := false
	for _, rf := range resources {
		r := rf()
		req := resource.MetadataRequest{ProviderTypeName: "truenas"}
		resp := &resource.MetadataResponse{}
		r.Metadata(context.Background(), req, resp)

		if resp.TypeName == "truenas_file" {
			found = true
			break
		}
	}

	if !found {
		t.Error("expected truenas_file resource to be registered")
	}
}
```

Add import for `"github.com/hashicorp/terraform-plugin-framework/resource"` if not present.

**Step 2: Run test to verify it fails**

Run: `mise run test`
Expected: FAIL - truenas_file not registered

**Step 3: Register the resource**

Update `Resources` function in `internal/provider/provider.go`:

```go
func (p *TrueNASProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		resources.NewDatasetResource,
		resources.NewHostPathResource,
		resources.NewAppResource,
		resources.NewFileResource,
	}
}
```

**Step 4: Run test to verify it passes**

Run: `mise run test`
Expected: PASS

**Step 5: Run full test suite and check coverage**

Run: `mise run coverage`
Expected: All tests pass, coverage meets target

**Step 6: Commit**

```bash
jj describe -m "feat(provider): register truenas_file resource"
jj new
```

---

## Phase 5: Documentation

### Task 5.1: Generate Documentation

**Files:**
- Create: `docs/resources/file.md` (auto-generated)

**Step 1: Generate docs**

Run: `mise run docs`
Expected: Documentation generated in docs/ directory

**Step 2: Verify documentation**

Run: `cat docs/resources/file.md`
Expected: File contains schema documentation

**Step 3: Final commit**

```bash
jj describe -m "docs: add truenas_file resource documentation"
jj new
```

---

## Summary

**Total Tasks:** 12
**Files Created:** 2 (`internal/resources/file.go`, `internal/client/sftp_test.go`)
**Files Modified:** 5 (`go.mod`, `internal/client/client.go`, `internal/client/ssh.go`, `internal/provider/provider.go`, `internal/resources/file_test.go`)

**Test Coverage Target:** 100%

**Commands Reference:**
- `mise run test` - Run unit tests
- `mise run coverage` - Generate coverage report
- `mise run lint` - Run linter
- `mise run build` - Build provider
- `mise run docs` - Generate documentation
