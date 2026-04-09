# Log Forwarding to Agent Stdout Design

## Objective

Provide a stable and UI-friendly way to display main container logs in the product page without modifying the current `supervisord` startup model.

This feature applies to **all unit types** (MySQL, PostgreSQL, Redis, Milvus, MongoDB, ProxySQL, Sentinel, etc.) — the agent does not need to know the specific service type.

The final responsibility split is:

1. the agent scans `LOG_MOUNT` for all `*.log` files and forwards them to agent stdout
2. Java reads the **agent container logs** and parses the forwarding protocol
3. the frontend renders the structured result returned by Java

## Background

All services are started under `supervisord` and their process output is redirected to files under `LOG_MOUNT`. Other service-specific log files may also exist in the same directory.

The main container is identified by the pod annotation `kubectl.kubernetes.io/default-container`.

## Final Design Choice

Use the agent sidecar to forward all log files under `LOG_MOUNT`.

This choice is intentional because:

1. the current `supervisord` model cannot be changed
2. existing file-based logging behavior must remain intact
3. the operator can add log forwarding without coupling the UI directly to file access
4. all log files in the directory are forwarded, not just supervisord output

## Source of Truth in Code

The implementation lives in:

- `pkg/agent/app/logtail/logtail.go` — generic logtail daemon, scans `LOG_MOUNT` for `*.log` files
- `pkg/agent/cmd/daemon.go` — registers the logtail daemon for all unit types

The logtail daemon is registered **before** the per-type switch, so every unit type gets log forwarding automatically.

## Runtime Flow

```text
Main container process (any service type)
  -> ${LOG_MOUNT}/*.log  (unit_app.out.log, unit_app.err.log, slow-query.log, etc.)

agent logtail daemon
  -> scans LOG_MOUNT for *.log files
  -> re-scans every 5 seconds to discover new files
  -> tails each file independently
  -> forwards every line to agent stdout
  -> prefixes every line with [<UNIT_TYPE>:<filename>]

Java service
  -> reads the agent container logs from Kubernetes
  -> parses protocol prefixes
  -> returns structured log entries to the frontend

Frontend
  -> renders structured logs
  -> handles styling, filtering, searching, and scrolling
```

## Forwarding Protocol

Each forwarded line uses the following stable prefix pattern:

```text
[<UNIT_TYPE>:<filename>] <original log line>
```

Examples:

```text
[MYSQL:unit_app.out.log] 2026-04-09 18:00:00 ready for connections
[MYSQL:unit_app.err.log] 2026-04-09 18:00:00 ERROR connection refused
[MYSQL:slow-query.log] # Time: 2026-04-09T18:00:00.000000Z
[REDIS:unit_app.out.log] Ready to accept connections
[MILVUS:unit_app.out.log] start query node
[POSTGRESQL:postgresql.log] LOG:  database system is ready
```

Rules:

1. `UNIT_TYPE` is uppercased in the prefix (e.g., `mysql` → `MYSQL`)
2. `filename` is the base name of the log file (no directory path)
3. agent emits these prefixes exactly
4. Java is responsible for parsing them — the regex `\[([A-Z-]+):([^\]]+)\] (.*)` covers all cases
5. frontend must not depend on raw string prefix parsing
6. the original raw line should be preserved downstream for troubleshooting
7. all output goes to agent **stdout** (no stdout/stderr split)

## Agent Implementation Details

### File discovery

The logtail daemon scans `LOG_MOUNT` using the glob pattern `*.log`:

- initial scan on startup
- periodic re-scan every 5 seconds to discover new log files
- each file is tailed at most once (tracked by path)
- non-`.log` files are ignored

### Prefix generation

Prefixes are dynamically generated per file:

```go
prefix = fmt.Sprintf("[%s:%s] ", strings.ToUpper(unitType), filename)
```

### Tailing behavior

Each file gets its own goroutine using `bufio.Scanner`:

1. polls incrementally rather than using `fsnotify`
2. retries opening the file for up to 30 seconds when the file does not exist yet
3. reopens the file when scanner errors occur for basic log rotation recovery
4. writes each output line as `prefix + line + "\n"` to `os.Stdout`

## Why Java Must Parse the Protocol

Java is the correct protocol boundary for this feature because:

1. the prefixes are a backend contract, not a UI contract
2. Java can extract unit type and filename, normalize log levels once for all consumers
3. Java can preserve raw lines while exposing stable structured DTOs
4. frontend logic remains focused on rendering rather than protocol interpretation

## Expected Java Output Model

Java should convert each line into a structured DTO such as:

```json
{
  "sequence": 1,
  "unitType": "mysql",
  "filename": "unit_app.out.log",
  "parsed": true,
  "timestamp": "2026-04-09T10:00:00Z",
  "level": "INFO",
  "message": "ready for connections",
  "rawMessage": "2026-04-09 18:00:00 INFO ready for connections",
  "rawLine": "[MYSQL:unit_app.out.log] 2026-04-09 18:00:00 INFO ready for connections"
}
```

## Testing Expectations

The current operator-side implementation should be validated by:

1. unit tests for prefix generation across all unit types and filenames
2. unit tests for directory scanning (multiple files, non-log files ignored)
3. unit tests for new file discovery on re-scan
4. unit tests for file-not-found retry behavior
5. Java-side parser tests using the generic regex
6. integration tests verifying Kubernetes log retrieval from the agent container

## Related Documents

- `docs/design/java-milvus-log-prompt.md`
- `docs/design/frontend-milvus-log-prompt.md`
