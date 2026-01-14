# SSH Session Semaphore Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add a counting semaphore to limit concurrent SSH sessions, preventing "open failed" errors when Terraform runs many resources in parallel.

**Architecture:** Buffered channel as counting semaphore in `SSHClient`. Default limit of 10, configurable via `max_sessions` provider attribute in SSH block. Semaphore acquired before creating sessions, released via defer.

**Tech Stack:** Go standard library (buffered channels), Terraform Plugin Framework

---

## Task 1: Add MaxSessions to SSHConfig

**Files:**
- Modify: `internal/client/ssh.go:24-31`
- Test: `internal/client/ssh_test.go`

**Step 1: Write test for MaxSessions in SSHConfig validation**

Add to `internal/client/ssh_test.go`:

```go
func TestSSHConfig_Validate_MaxSessions(t *testing.T) {
	tests := []struct {
		name        string
		maxSessions int
		wantDefault int
	}{
		{"zero uses default", 0, 0},      // 0 means use default (validated at SSHClient level)
		{"negative uses default", -5, 0}, // negative means use default
		{"positive preserved", 25, 0},    // positive values preserved as-is
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &SSHConfig{
				Host:               "truenas.local",
				PrivateKey:         testPrivateKey,
				HostKeyFingerprint: testHostKeyFingerprint,
				MaxSessions:        tt.maxSessions,
			}
			err := config.Validate()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			// MaxSessions should be unchanged by Validate (defaulting happens in NewSSHClient)
			if config.MaxSessions != tt.maxSessions {
				t.Errorf("MaxSessions = %d, want %d", config.MaxSessions, tt.maxSessions)
			}
		})
	}
}
```

**Step 2: Run test to verify it fails**

```bash
cd /Users/sh/Projects/personal/terraform-provider-truenas && go test ./internal/client/... -run TestSSHConfig_Validate_MaxSessions -v
```

Expected: FAIL with "unknown field 'MaxSessions'"

**Step 3: Add MaxSessions field to SSHConfig**

In `internal/client/ssh.go`, modify `SSHConfig` struct (around line 24):

```go
// SSHConfig holds configuration for SSH connection to TrueNAS.
type SSHConfig struct {
	Host               string
	Port               int
	User               string
	PrivateKey         string
	HostKeyFingerprint string
	MaxSessions        int // Maximum concurrent SSH sessions (0 = default of 10)
}
```

**Step 4: Run test to verify it passes**

```bash
cd /Users/sh/Projects/personal/terraform-provider-truenas && go test ./internal/client/... -run TestSSHConfig_Validate_MaxSessions -v
```

Expected: PASS

**Step 5: Commit**

```bash
git add internal/client/ssh.go internal/client/ssh_test.go
git commit -m "feat(ssh): add MaxSessions field to SSHConfig"
```

---

## Task 2: Add sessionSem to SSHClient and Initialize

**Files:**
- Modify: `internal/client/ssh.go:186-209`
- Test: `internal/client/ssh_test.go`

**Step 1: Write test for semaphore initialization**

Add to `internal/client/ssh_test.go`:

```go
func TestNewSSHClient_SessionSemaphore(t *testing.T) {
	tests := []struct {
		name            string
		maxSessions     int
		wantSemaphoreCap int
	}{
		{"default when zero", 0, 10},
		{"default when negative", -5, 10},
		{"custom value", 25, 25},
		{"small value", 3, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &SSHConfig{
				Host:               "truenas.local",
				PrivateKey:         testPrivateKey,
				HostKeyFingerprint: testHostKeyFingerprint,
				MaxSessions:        tt.maxSessions,
			}
			client, err := NewSSHClient(config)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if cap(client.sessionSem) != tt.wantSemaphoreCap {
				t.Errorf("sessionSem capacity = %d, want %d", cap(client.sessionSem), tt.wantSemaphoreCap)
			}
		})
	}
}
```

**Step 2: Run test to verify it fails**

```bash
cd /Users/sh/Projects/personal/terraform-provider-truenas && go test ./internal/client/... -run TestNewSSHClient_SessionSemaphore -v
```

Expected: FAIL with "client.sessionSem undefined"

**Step 3: Add sessionSem field and initialize it**

In `internal/client/ssh.go`, modify `SSHClient` struct (around line 186):

```go
// SSHClient implements Client interface using SSH/midclt.
type SSHClient struct {
	config        *SSHConfig
	client        *ssh.Client
	clientWrapper sshClientWrapper
	sftpClient    sftpClient
	dialer        sshDialer
	mu            sync.Mutex
	sessionSem    chan struct{} // limits concurrent SSH sessions
}
```

Modify `NewSSHClient` function (around line 199):

```go
// NewSSHClient creates a new SSH client for TrueNAS.
func NewSSHClient(config *SSHConfig) (*SSHClient, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	maxSessions := config.MaxSessions
	if maxSessions <= 0 {
		maxSessions = 10 // default
	}

	return &SSHClient{
		config:     config,
		dialer:     &defaultDialer{},
		sessionSem: make(chan struct{}, maxSessions),
	}, nil
}
```

**Step 4: Run test to verify it passes**

```bash
cd /Users/sh/Projects/personal/terraform-provider-truenas && go test ./internal/client/... -run TestNewSSHClient_SessionSemaphore -v
```

Expected: PASS

**Step 5: Commit**

```bash
git add internal/client/ssh.go internal/client/ssh_test.go
git commit -m "feat(ssh): add session semaphore to SSHClient"
```

---

## Task 3: Add acquireSession Helper Method

**Files:**
- Modify: `internal/client/ssh.go`
- Test: `internal/client/ssh_test.go`

**Step 1: Write test for acquireSession**

Add to `internal/client/ssh_test.go`:

```go
func TestSSHClient_AcquireSession(t *testing.T) {
	config := &SSHConfig{
		Host:               "truenas.local",
		PrivateKey:         testPrivateKey,
		HostKeyFingerprint: testHostKeyFingerprint,
		MaxSessions:        2,
	}
	client, err := NewSSHClient(config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Initially empty
	if len(client.sessionSem) != 0 {
		t.Errorf("sessionSem should be empty, got %d", len(client.sessionSem))
	}

	// Acquire first slot
	release1 := client.acquireSession()
	if len(client.sessionSem) != 1 {
		t.Errorf("sessionSem should have 1 item, got %d", len(client.sessionSem))
	}

	// Acquire second slot
	release2 := client.acquireSession()
	if len(client.sessionSem) != 2 {
		t.Errorf("sessionSem should have 2 items, got %d", len(client.sessionSem))
	}

	// Release first slot
	release1()
	if len(client.sessionSem) != 1 {
		t.Errorf("sessionSem should have 1 item after release, got %d", len(client.sessionSem))
	}

	// Release second slot
	release2()
	if len(client.sessionSem) != 0 {
		t.Errorf("sessionSem should be empty after both releases, got %d", len(client.sessionSem))
	}
}
```

**Step 2: Run test to verify it fails**

```bash
cd /Users/sh/Projects/personal/terraform-provider-truenas && go test ./internal/client/... -run TestSSHClient_AcquireSession -v
```

Expected: FAIL with "client.acquireSession undefined"

**Step 3: Implement acquireSession**

In `internal/client/ssh.go`, add after the `NewSSHClient` function (around line 215):

```go
// acquireSession blocks until a session slot is available and returns a release function.
func (c *SSHClient) acquireSession() func() {
	c.sessionSem <- struct{}{}
	return func() {
		<-c.sessionSem
	}
}
```

**Step 4: Run test to verify it passes**

```bash
cd /Users/sh/Projects/personal/terraform-provider-truenas && go test ./internal/client/... -run TestSSHClient_AcquireSession -v
```

Expected: PASS

**Step 5: Commit**

```bash
git add internal/client/ssh.go internal/client/ssh_test.go
git commit -m "feat(ssh): add acquireSession helper method"
```

---

## Task 4: Add Semaphore to Call Method

**Files:**
- Modify: `internal/client/ssh.go:277-311` (Call method)
- Test: `internal/client/ssh_test.go`

**Step 1: Write test for semaphore limiting in Call**

Add to `internal/client/ssh_test.go`:

```go
func TestSSHClient_Call_RespectsSemaphore(t *testing.T) {
	var activeCount int32
	var maxActive int32

	mockClient := &mockSSHClient{
		newSessionFunc: func() (sshSession, error) {
			current := atomic.AddInt32(&activeCount, 1)
			// Track max concurrent
			for {
				old := atomic.LoadInt32(&maxActive)
				if current <= old || atomic.CompareAndSwapInt32(&maxActive, old, current) {
					break
				}
			}
			return &mockSession{
				combinedOutputFunc: func(cmd string) ([]byte, error) {
					time.Sleep(50 * time.Millisecond) // Simulate work
					return []byte(`{"result": "ok"}`), nil
				},
				closeFunc: func() error {
					atomic.AddInt32(&activeCount, -1)
					return nil
				},
			}, nil
		},
	}

	client := &SSHClient{
		config:        &SSHConfig{Host: "test", PrivateKey: testPrivateKey, HostKeyFingerprint: testHostKeyFingerprint},
		clientWrapper: mockClient,
		sessionSem:    make(chan struct{}, 2), // Limit to 2 concurrent
	}

	// Launch 5 concurrent calls
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = client.Call(context.Background(), "test.method", nil)
		}()
	}
	wg.Wait()

	// Max concurrent should never exceed semaphore limit
	if atomic.LoadInt32(&maxActive) > 2 {
		t.Errorf("max concurrent sessions = %d, want <= 2", maxActive)
	}
}
```

**Step 2: Run test to verify it fails**

```bash
cd /Users/sh/Projects/personal/terraform-provider-truenas && go test ./internal/client/... -run TestSSHClient_Call_RespectsSemaphore -v
```

Expected: FAIL with max concurrent > 2 (since semaphore not yet used in Call)

Note: You'll need to add these imports at the top of the test file:
```go
import (
	"sync"
	"sync/atomic"
	"time"
)
```

**Step 3: Add semaphore acquisition to Call method**

In `internal/client/ssh.go`, modify the `Call` method (around line 277):

```go
// Call executes a midclt command and returns the parsed JSON response.
// Note: The ctx parameter is accepted for interface compatibility and future
// use (e.g., command cancellation, timeouts) but is not currently used.
func (c *SSHClient) Call(ctx context.Context, method string, params any) (json.RawMessage, error) {
	// Acquire session slot (blocks if at limit)
	release := c.acquireSession()
	defer release()

	// Ensure we're connected (only if not already mocked)
	if c.clientWrapper == nil {
		if err := c.connect(); err != nil {
			return nil, err
		}
	}

	// Build command (use sudo for non-root users with sudo access)
	cmd := fmt.Sprintf("sudo midclt call %s", method)
	paramsStr, err := serializeParams(params)
	if err != nil {
		return nil, err
	}
	cmd += paramsStr

	// Create session
	session, err := c.clientWrapper.NewSession()
	if err != nil {
		return nil, err
	}
	defer session.Close()

	// Execute command - use CombinedOutput to capture stderr for error messages
	output, err := session.CombinedOutput(cmd)
	if err != nil {
		// Include the output (which contains stderr) in the error message
		if len(output) > 0 {
			return nil, fmt.Errorf("%w: %s", err, string(output))
		}
		return nil, err
	}

	return json.RawMessage(output), nil
}
```

**Step 4: Run test to verify it passes**

```bash
cd /Users/sh/Projects/personal/terraform-provider-truenas && go test ./internal/client/... -run TestSSHClient_Call_RespectsSemaphore -v
```

Expected: PASS

**Step 5: Commit**

```bash
git add internal/client/ssh.go internal/client/ssh_test.go
git commit -m "feat(ssh): add semaphore to Call method"
```

---

## Task 5: Add Semaphore to CallAndWait Method

**Files:**
- Modify: `internal/client/ssh.go:317-353` (CallAndWait method)

**Step 1: Write test for semaphore in CallAndWait**

Add to `internal/client/ssh_test.go`:

```go
func TestSSHClient_CallAndWait_RespectsSemaphore(t *testing.T) {
	var activeCount int32
	var maxActive int32

	mockClient := &mockSSHClient{
		newSessionFunc: func() (sshSession, error) {
			current := atomic.AddInt32(&activeCount, 1)
			for {
				old := atomic.LoadInt32(&maxActive)
				if current <= old || atomic.CompareAndSwapInt32(&maxActive, old, current) {
					break
				}
			}
			return &mockSession{
				combinedOutputFunc: func(cmd string) ([]byte, error) {
					time.Sleep(50 * time.Millisecond)
					return []byte(`{}`), nil
				},
				closeFunc: func() error {
					atomic.AddInt32(&activeCount, -1)
					return nil
				},
			}, nil
		},
	}

	client := &SSHClient{
		config:        &SSHConfig{Host: "test", PrivateKey: testPrivateKey, HostKeyFingerprint: testHostKeyFingerprint},
		clientWrapper: mockClient,
		sessionSem:    make(chan struct{}, 2),
	}

	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = client.CallAndWait(context.Background(), "test.method", nil)
		}()
	}
	wg.Wait()

	if atomic.LoadInt32(&maxActive) > 2 {
		t.Errorf("max concurrent sessions = %d, want <= 2", maxActive)
	}
}
```

**Step 2: Run test to verify it fails**

```bash
cd /Users/sh/Projects/personal/terraform-provider-truenas && go test ./internal/client/... -run TestSSHClient_CallAndWait_RespectsSemaphore -v
```

Expected: FAIL with max concurrent > 2

**Step 3: Add semaphore acquisition to CallAndWait method**

In `internal/client/ssh.go`, modify the `CallAndWait` method (around line 317):

```go
// CallAndWait executes a command and waits for job completion.
// Uses midclt's -j flag to wait for long-running jobs.
// Note: The response is not parsed as it contains unparseable progress output.
// Callers should query the resource state separately after this returns.
func (c *SSHClient) CallAndWait(ctx context.Context, method string, params any) (json.RawMessage, error) {
	// Acquire session slot (blocks if at limit)
	release := c.acquireSession()
	defer release()

	// Ensure we're connected (only if not already mocked)
	if c.clientWrapper == nil {
		if err := c.connect(); err != nil {
			return nil, err
		}
	}

	// Build command with -j flag for job waiting
	cmd := fmt.Sprintf("sudo midclt call -j %s", method)
	paramsStr, err := serializeParams(params)
	if err != nil {
		return nil, err
	}
	cmd += paramsStr

	// Create session
	session, err := c.clientWrapper.NewSession()
	if err != nil {
		return nil, err
	}
	defer session.Close()

	// Execute command - use CombinedOutput to capture stderr for error messages
	output, err := session.CombinedOutput(cmd)
	if err != nil {
		// Strip ANSI codes for cleaner error messages
		cleaned := ansiRegex.ReplaceAll(output, nil)
		if len(cleaned) > 0 {
			return nil, fmt.Errorf("%w: %s", err, string(cleaned))
		}
		return nil, err
	}

	// Success - return nil as callers should query state separately
	return nil, nil
}
```

**Step 4: Run test to verify it passes**

```bash
cd /Users/sh/Projects/personal/terraform-provider-truenas && go test ./internal/client/... -run TestSSHClient_CallAndWait_RespectsSemaphore -v
```

Expected: PASS

**Step 5: Commit**

```bash
git add internal/client/ssh.go internal/client/ssh_test.go
git commit -m "feat(ssh): add semaphore to CallAndWait method"
```

---

## Task 6: Add Semaphore to runSudo Method

**Files:**
- Modify: `internal/client/ssh.go:449-480` (runSudo method)

**Step 1: Write test for semaphore in runSudo**

Add to `internal/client/ssh_test.go`:

```go
func TestSSHClient_runSudo_RespectsSemaphore(t *testing.T) {
	var activeCount int32
	var maxActive int32

	mockClient := &mockSSHClient{
		newSessionFunc: func() (sshSession, error) {
			current := atomic.AddInt32(&activeCount, 1)
			for {
				old := atomic.LoadInt32(&maxActive)
				if current <= old || atomic.CompareAndSwapInt32(&maxActive, old, current) {
					break
				}
			}
			return &mockSession{
				combinedOutputFunc: func(cmd string) ([]byte, error) {
					time.Sleep(50 * time.Millisecond)
					return []byte{}, nil
				},
				closeFunc: func() error {
					atomic.AddInt32(&activeCount, -1)
					return nil
				},
			}, nil
		},
	}

	client := &SSHClient{
		config:        &SSHConfig{Host: "test", PrivateKey: testPrivateKey, HostKeyFingerprint: testHostKeyFingerprint},
		clientWrapper: mockClient,
		sessionSem:    make(chan struct{}, 2),
	}

	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = client.runSudo(context.Background(), "ls", "-la")
		}()
	}
	wg.Wait()

	if atomic.LoadInt32(&maxActive) > 2 {
		t.Errorf("max concurrent sessions = %d, want <= 2", maxActive)
	}
}
```

**Step 2: Run test to verify it fails**

```bash
cd /Users/sh/Projects/personal/terraform-provider-truenas && go test ./internal/client/... -run TestSSHClient_runSudo_RespectsSemaphore -v
```

Expected: FAIL with max concurrent > 2

**Step 3: Add semaphore acquisition to runSudo method**

In `internal/client/ssh.go`, modify the `runSudo` method (around line 449):

```go
// runSudo executes a command with sudo via SSH.
func (c *SSHClient) runSudo(ctx context.Context, args ...string) error {
	// Acquire session slot (blocks if at limit)
	release := c.acquireSession()
	defer release()

	// Ensure we're connected (only if not already mocked)
	if c.clientWrapper == nil {
		if err := c.connect(); err != nil {
			return err
		}
	}

	// Build command with proper escaping
	var escaped []string
	for _, arg := range args {
		escaped = append(escaped, shellescape.Quote(arg))
	}
	cmd := "sudo " + strings.Join(escaped, " ")

	// Create session
	session, err := c.clientWrapper.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	// Execute command
	output, err := session.CombinedOutput(cmd)
	if err != nil {
		if len(output) > 0 {
			return fmt.Errorf("%w: %s", err, string(output))
		}
		return err
	}
	return nil
}
```

**Step 4: Run test to verify it passes**

```bash
cd /Users/sh/Projects/personal/terraform-provider-truenas && go test ./internal/client/... -run TestSSHClient_runSudo_RespectsSemaphore -v
```

Expected: PASS

**Step 5: Commit**

```bash
git add internal/client/ssh.go internal/client/ssh_test.go
git commit -m "feat(ssh): add semaphore to runSudo method"
```

---

## Task 7: Add Semaphore to ReadFile Method

**Files:**
- Modify: `internal/client/ssh.go:424-446` (ReadFile method)

**Step 1: Write test for semaphore in ReadFile**

Add to `internal/client/ssh_test.go`:

```go
func TestSSHClient_ReadFile_RespectsSemaphore(t *testing.T) {
	var activeCount int32
	var maxActive int32

	mockSFTP := &mockSFTPClient{
		openFunc: func(path string) (sftpFile, error) {
			current := atomic.AddInt32(&activeCount, 1)
			for {
				old := atomic.LoadInt32(&maxActive)
				if current <= old || atomic.CompareAndSwapInt32(&maxActive, old, current) {
					break
				}
			}
			return &mockSFTPFile{
				readFunc: func(p []byte) (int, error) {
					time.Sleep(50 * time.Millisecond)
					return 0, io.EOF
				},
				closeFunc: func() error {
					atomic.AddInt32(&activeCount, -1)
					return nil
				},
			}, nil
		},
	}

	client := &SSHClient{
		config:     &SSHConfig{Host: "test", PrivateKey: testPrivateKey, HostKeyFingerprint: testHostKeyFingerprint},
		sftpClient: mockSFTP,
		sessionSem: make(chan struct{}, 2),
	}

	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = client.ReadFile(context.Background(), "/test/file")
		}()
	}
	wg.Wait()

	if atomic.LoadInt32(&maxActive) > 2 {
		t.Errorf("max concurrent sessions = %d, want <= 2", maxActive)
	}
}
```

**Step 2: Run test to verify it fails**

```bash
cd /Users/sh/Projects/personal/terraform-provider-truenas && go test ./internal/client/... -run TestSSHClient_ReadFile_RespectsSemaphore -v
```

Expected: FAIL with max concurrent > 2

**Step 3: Add semaphore acquisition to ReadFile method**

In `internal/client/ssh.go`, modify the `ReadFile` method (around line 424):

```go
// ReadFile reads the content of a file from the remote system.
func (c *SSHClient) ReadFile(ctx context.Context, path string) ([]byte, error) {
	// Acquire session slot (blocks if at limit)
	release := c.acquireSession()
	defer release()

	// Connect SFTP if needed (skip if already mocked)
	if c.sftpClient == nil {
		if err := c.connectSFTP(); err != nil {
			return nil, err
		}
	}

	// Open the file
	file, err := c.sftpClient.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %q: %w", path, err)
	}
	defer file.Close()

	// Read all content - handles partial reads correctly
	content, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %q: %w", path, err)
	}

	return content, nil
}
```

**Step 4: Run test to verify it passes**

```bash
cd /Users/sh/Projects/personal/terraform-provider-truenas && go test ./internal/client/... -run TestSSHClient_ReadFile_RespectsSemaphore -v
```

Expected: PASS

**Step 5: Commit**

```bash
git add internal/client/ssh.go internal/client/ssh_test.go
git commit -m "feat(ssh): add semaphore to ReadFile method"
```

---

## Task 8: Add max_sessions to Provider Schema

**Files:**
- Modify: `internal/provider/provider.go:26-31,66-86,127-132`

**Step 1: Add MaxSessions to SSHBlockModel**

In `internal/provider/provider.go`, modify `SSHBlockModel` (around line 26):

```go
// SSHBlockModel describes the SSH configuration block.
type SSHBlockModel struct {
	Port               types.Int64  `tfsdk:"port"`
	User               types.String `tfsdk:"user"`
	PrivateKey         types.String `tfsdk:"private_key"`
	HostKeyFingerprint types.String `tfsdk:"host_key_fingerprint"`
	MaxSessions        types.Int64  `tfsdk:"max_sessions"`
}
```

**Step 2: Add max_sessions to schema**

In `internal/provider/provider.go`, modify the SSH block schema (around line 66):

```go
"ssh": schema.SingleNestedBlock{
	Description: "SSH connection configuration.",
	Attributes: map[string]schema.Attribute{
		"port": schema.Int64Attribute{
			Description: "SSH port. Defaults to 22.",
			Optional:    true,
		},
		"user": schema.StringAttribute{
			Description: "SSH username. Defaults to 'root'.",
			Optional:    true,
		},
		"private_key": schema.StringAttribute{
			Description: "SSH private key content.",
			Required:    true,
			Sensitive:   true,
		},
		"host_key_fingerprint": schema.StringAttribute{
			Description: "SHA256 fingerprint of the TrueNAS server's SSH host key. " +
				"Get it with: ssh-keyscan <host> 2>/dev/null | ssh-keygen -lf -",
			Required:  true,
			Sensitive: false,
		},
		"max_sessions": schema.Int64Attribute{
			Description: "Maximum concurrent SSH sessions. Defaults to 10. " +
				"Increase for large deployments, decrease if you see connection errors.",
			Optional: true,
		},
	},
},
```

**Step 3: Pass MaxSessions to SSHConfig**

In `internal/provider/provider.go`, modify the Configure method (around line 127):

```go
// Set optional values if provided
if !config.SSH.Port.IsNull() {
	sshConfig.Port = int(config.SSH.Port.ValueInt64())
}
if !config.SSH.User.IsNull() {
	sshConfig.User = config.SSH.User.ValueString()
}
if !config.SSH.MaxSessions.IsNull() {
	sshConfig.MaxSessions = int(config.SSH.MaxSessions.ValueInt64())
}
```

**Step 4: Run all tests**

```bash
cd /Users/sh/Projects/personal/terraform-provider-truenas && go test ./... -v
```

Expected: All tests PASS

**Step 5: Commit**

```bash
git add internal/provider/provider.go
git commit -m "feat(provider): add max_sessions configuration option"
```

---

## Task 9: Run Full Test Suite and Verify

**Step 1: Run all tests**

```bash
cd /Users/sh/Projects/personal/terraform-provider-truenas && go test ./... -v
```

Expected: All tests PASS

**Step 2: Build provider**

```bash
cd /Users/sh/Projects/personal/terraform-provider-truenas && go build ./...
```

Expected: Build succeeds with no errors

**Step 3: Run linter (if available)**

```bash
cd /Users/sh/Projects/personal/terraform-provider-truenas && golangci-lint run 2>/dev/null || echo "Linter not installed, skipping"
```

**Step 4: Commit any fixes if needed**

If linting reveals issues, fix them and commit.

---

## Task 10: Update Documentation

**Files:**
- Modify: `docs/index.md` or equivalent provider documentation

**Step 1: Check for existing docs**

```bash
ls -la /Users/sh/Projects/personal/terraform-provider-truenas/docs/
```

**Step 2: Add max_sessions to provider documentation**

If provider docs exist, add documentation for the new `max_sessions` attribute in the SSH block.

**Step 3: Commit documentation**

```bash
git add docs/
git commit -m "docs: add max_sessions configuration option"
```
