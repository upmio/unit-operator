# Log Forwarding to Agent Stdout Design

## Objective

Provide a stable and UI-friendly way to display main container logs in the product page without modifying the current `supervisord` startup model.

This feature applies to **all unit types** (MySQL, PostgreSQL, Redis, Milvus, MongoDB, ProxySQL, Sentinel, etc.) — the agent does not need to know the specific service type.

The final responsibility split is:

1. the agent forwards main container log files to the agent container stdout/stderr
2. Java reads the **agent container logs** and parses the forwarding protocol
3. the frontend renders the structured result returned by Java

## Background

All services are started under `supervisord` and their process output is redirected to files:

```ini
[program:unit_app]
stderr_logfile=%(ENV_LOG_MOUNT)s/unit_app.err.log
stdout_logfile=%(ENV_LOG_MOUNT)s/unit_app.out.log
autostart=false
```

Even if a service itself is configured with `stdout: true`, `supervisord` still captures and redirects the process output to the files above. As a result, the main container stdout is not the right source for UI log viewing.

The main container is identified by the pod annotation `kubectl.kubernetes.io/default-container`.

## Final Design Choice

Use the agent sidecar to forward the main container log files.

This choice is intentional because:

1. the current `supervisord` model cannot be changed
2. existing file-based logging behavior must remain intact
3. the operator can add log forwarding without coupling the UI directly to file access

## Source of Truth in Code

The implementation lives in:

- `pkg/agent/app/logtail/logtail.go` — generic logtail daemon, parameterized by `UNIT_TYPE`
- `pkg/agent/cmd/daemon.go` — registers the logtail daemon for all unit types

The logtail daemon is registered **before** the per-type switch, so every unit type gets log forwarding automatically.

## Runtime Flow

```text
Main container process (any service type)
  -> ${LOG_MOUNT}/unit_app.out.log
  -> ${LOG_MOUNT}/unit_app.err.log

agent logtail daemon
  -> reads UNIT_TYPE env to compute prefix
  -> tails both files
  -> forwards stdout lines to agent stdout
  -> forwards stderr lines to agent stderr
  -> prefixes every forwarded line with a typed marker

Java service
  -> reads the agent container logs from Kubernetes
  -> parses protocol prefixes
  -> returns structured log entries to the frontend

Frontend
  -> renders structured logs
  -> handles styling, filtering, searching, and scrolling
```

## Forwarding Protocol

Each forwarded line uses one of the following stable prefix patterns:

```text
[<UNIT_TYPE>-STDOUT] <original stdout line>
[<UNIT_TYPE>-STDERR] <original stderr line>
```

Examples:

```text
[MILVUS-STDOUT] 2026-03-24 18:00:00 INFO start query node
[MYSQL-STDERR] 2026-03-24 18:00:00 ERROR connection refused
[REDIS-STDOUT] 2026-03-24 18:00:00 Ready to accept connections
[POSTGRESQL-STDOUT] 2026-03-24 18:00:00 LOG database system is ready
```

These prefixes are part of the cross-component contract.

Rules:

1. `UNIT_TYPE` is uppercased in the prefix (e.g., `mysql` → `MYSQL`)
2. agent emits these prefixes exactly
3. Java is responsible for parsing them — the regex pattern `\[([A-Z-]+)-(STDOUT|STDERR)\] (.*)` covers all types
4. frontend must not depend on raw string prefix parsing
5. the original raw line should be preserved downstream for troubleshooting

## Agent Implementation Details

### Forwarded files

The logtail daemon reads:

- `unit_app.out.log` — forwarded to agent stdout
- `unit_app.err.log` — forwarded to agent stderr

### Prefix generation

Prefixes are dynamically generated from `UNIT_TYPE`:

- `stdoutPrefix = fmt.Sprintf("[%s-STDOUT] ", strings.ToUpper(unitType))`
- `stderrPrefix = fmt.Sprintf("[%s-STDERR] ", strings.ToUpper(unitType))`

### Tailing behavior

The implementation uses native Go logic based on `bufio.Scanner`.

Important behavior:

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

The frontend should not parse the protocol prefix strings directly.

## Testing Expectations

The current operator-side implementation should be validated by:

1. unit tests for prefix generation across all unit types
2. unit tests for prefix forwarding behavior
3. unit tests for file-not-found retry behavior
4. Java-side parser tests for stdout, stderr, and unknown lines (generic regex-based)
5. integration tests verifying that Kubernetes log retrieval from the agent container contains prefixed lines

## Related Documents

- `docs/design/java-milvus-log-prompt.md`
- `docs/design/frontend-milvus-log-prompt.md`
