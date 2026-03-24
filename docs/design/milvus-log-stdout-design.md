# Milvus Log Forwarding to Agent Stdout Design

## Objective

Provide a stable and UI-friendly way to display Milvus logs in the product page without modifying the current `supervisord` startup model.

The final responsibility split is:

1. the agent forwards Milvus log files to the agent container stdout/stderr
2. Java reads the **agent container logs** and parses the forwarding protocol
3. the frontend renders the structured result returned by Java

## Background

Milvus is started under `supervisord` and its process output is redirected to files:

```ini
[program:unit_app]
command=/usr/local/milvus/bin/milvus run %(ENV_ARCH_MODE)s
stderr_logfile=%(ENV_LOG_MOUNT)s/unit_app.err.log
stdout_logfile=%(ENV_LOG_MOUNT)s/unit_app.out.log
autostart=false
```

Even if Milvus itself is configured with `stdout: true`, `supervisord` still captures and redirects the process output to the files above. As a result, the main container stdout is not the right source for UI log viewing.

## Final Design Choice

Use the agent sidecar to forward the Milvus log files.

This choice is intentional because:

1. the current `supervisord` model cannot be changed
2. existing file-based logging behavior must remain intact
3. the operator can add log forwarding without coupling the UI directly to file access

## Source of Truth in Code

The implementation lives in:

- `pkg/agent/app/milvus/logtail.go`
- `pkg/agent/cmd/daemon.go`

For Milvus units, the daemon command registers both:

- the Milvus gRPC app
- the Milvus log forwarding daemon app

## Runtime Flow

```text
Milvus process
  -> ${LOG_MOUNT}/unit_app.out.log
  -> ${LOG_MOUNT}/unit_app.err.log

agent milvus-logtail daemon
  -> tails both files
  -> forwards stdout lines to agent stdout
  -> forwards stderr lines to agent stderr
  -> prefixes every forwarded line with a stable marker

Java service
  -> reads the agent container logs from Kubernetes
  -> parses protocol prefixes
  -> returns structured log entries to the frontend

Frontend
  -> renders structured logs
  -> handles styling, filtering, searching, and scrolling
```

## Forwarding Protocol

Each forwarded line uses one of the following stable prefixes:

```text
[MILVUS-STDOUT] <original milvus stdout line>
[MILVUS-STDERR] <original milvus stderr line>
```

These prefixes are part of the cross-component contract.

Rules:

1. agent emits these prefixes exactly
2. Java is responsible for parsing them
3. frontend must not depend on raw string prefix parsing
4. the original raw line should be preserved downstream for troubleshooting

## Current Agent Implementation Details

### Forwarded files

The current implementation reads:

- `unit_app.out.log`
- `unit_app.err.log`

### Prefixes

The current implementation emits:

- `stdoutLogPrefix = "[MILVUS-STDOUT] "`
- `stderrLogPrefix = "[MILVUS-STDERR] "`

These constants are documented in code as stable parsing protocol prefixes for downstream Java/UI consumers.

### Tailing behavior

The implementation uses native Go logic based on `bufio.Scanner`.

Important current behavior:

1. it polls incrementally rather than using `fsnotify`
2. it retries opening the file for up to 30 seconds when the file does not exist yet
3. it reopens the file when scanner errors occur, which provides basic recovery behavior
4. it writes each output line in a single write call using `prefix + line + "\n"`

### Default constraints

The current design does not rely on:

- changing `supervisord`
- a new `GrpcCall` log-streaming action
- frontend-side raw prefix parsing

## Why Java Must Parse the Protocol

Java is the correct protocol boundary for this feature because:

1. the prefixes are a backend contract, not a UI contract
2. Java can normalize stdout/stderr and log levels once for all consumers
3. Java can preserve raw lines while exposing stable structured DTOs
4. frontend logic remains focused on rendering rather than protocol interpretation

## Expected Java Output Model

Java should convert each line into a structured DTO such as:

```json
{
  "sequence": 1,
  "stream": "stdout",
  "parsed": true,
  "timestamp": "2026-03-24T10:00:00Z",
  "level": "INFO",
  "message": "start query node",
  "rawMessage": "2026-03-24 18:00:00 INFO start query node",
  "rawLine": "[MILVUS-STDOUT] 2026-03-24 18:00:00 INFO start query node"
}
```

At minimum, Java should preserve:

- `sequence`
- `stream`
- `parsed`
- `level`
- `message`
- `rawLine`

## Expected Frontend Behavior

The frontend should:

1. consume structured log entries from Java
2. render stream and level badges
3. support filtering, searching, auto-scroll, and reconnect states
4. fall back to `rawMessage` or `rawLine` when structured fields are incomplete

The frontend should not parse the `[MILVUS-STDOUT]` or `[MILVUS-STDERR]` strings directly.

## Testing Expectations

The current operator-side implementation should be validated by:

1. unit tests for prefix forwarding behavior
2. unit tests for file-not-found retry behavior
3. Java-side parser tests for stdout, stderr, and unknown lines
4. integration tests verifying that Kubernetes log retrieval from the agent container contains prefixed lines

## Related Documents

- `docs/design/java-milvus-log-prompt.md`
- `docs/design/frontend-milvus-log-prompt.md`
