# ClickHouse Service Type Design

Date: 2026-04-28
Source requirement: docs/specs/2026-04-27-clickhouse-service-type-spec.md
Target UPM version: 2.0
ClickHouse package version: 26.3.9.8

## Summary

`unit-operator` will support ClickHouse through the existing UnitSet, Unit, config sync, unit-agent, and GrpcCall paths. The operator will not add a ClickHouse topology controller, shard assignment logic, Keeper operation surface, new CRD kind, or compose-operator dependency.

The API server creates two UnitSets:

- `clickhouse`
- `clickhouse-keeper`

Both use version `26.3.9.8`. `clickhouse` supports the requested two or three replicas. `clickhouse-keeper` is managed as a regular UnitSet. The three-unit Keeper default is owned by the API server; `unit-operator` honors the `spec.units` value it receives.

## Architecture

ClickHouse is added as a normal engine app, following the existing MySQL, PostgreSQL, MongoDB, Redis, and Milvus patterns.

UnitSet and Unit v1alpha2 APIs remain unchanged. `UnitSet.spec.type` is already a string, so `clickhouse` and `clickhouse-keeper` can use the existing lifecycle path for template ConfigMaps, per-unit value ConfigMaps, Units, Pods, PVCs, Services, and status.

Runtime operations are available only for `clickhouse` Units. A new `pkg/agent/app/clickhouse` package provides the ClickHouse gRPC service, generated client/server bindings, server implementation, registration, and tests. The unit-agent daemon registers this app only when `UNIT_TYPE=clickhouse`. A `clickhouse-keeper` unit-agent does not expose ClickHouse backup, restore, or dynamic parameter operations.

The GrpcCall v1alpha1 API adds `clickhouse` to its type enum. The GrpcCall controller routes `type=clickhouse` and `action=logical-backup|restore|set-variable` to the ClickHouse operation client. Unsupported ClickHouse actions return a clear unsupported-action error.

## Agent Contract

Add `pkg/agent/app/clickhouse/pb/clickhouse.proto`:

```proto
syntax = "proto3";

package clickhouse;
option go_package="github.com/upmio/unit-operator/pkg/agent/app/clickhouse";

import "pkg/agent/app/common/pb/common.proto";

message LogicalBackupRequest {
  string backup_file = 1;
  string username = 2;
  common.ObjectStorage object_storage = 3;
}

message RestoreRequest {
  string backup_file = 1;
  common.ObjectStorage object_storage = 2;
}

message SetVariableRequest {
  string key = 1;
  string value = 2;
  string username = 3;
}

service ClickHouseOperation {
  rpc LogicalBackup (LogicalBackupRequest) returns (common.Empty);
  rpc Restore (RestoreRequest) returns (common.Empty);
  rpc SetVariable (SetVariableRequest) returns (common.Empty);
}
```

GrpcCall parameters continue to use protobuf JSON mapping through the existing `protojson.Unmarshal` path. For example, users can pass `backupFile` and `objectStorage` in CR YAML, and the controller will unmarshal them into the generated request.

Object storage uses the existing `common.ObjectStorage` message. Credentials follow the existing unit-agent convention: the request carries `username`, and the agent decrypts `SECRET_MOUNT/<username>` using the configured AES key.

## Agent Execution

The ClickHouse agent executes operations against the local ClickHouse process through `clickhouse-client`. It does not use `ON CLUSTER`.

Before each operation, the agent:

1. Checks the managed process through SLM `CheckProcessStarted`.
2. Reads connection settings from environment variables.
3. Falls back to local defaults when optional connection variables are absent.
4. Decrypts the management password from the mounted secret file.
5. Validates required request fields before building SQL.

Connection environment variables:

- `CLICKHOUSE_HOST`, default `127.0.0.1`
- `CLICKHOUSE_PORT`, default `9000`
- `CLICKHOUSE_SECURE`, default `false`

The implementation may add small private helpers for parsing these values, building command arguments, escaping SQL literals, and validating request fields. These helpers should stay inside the ClickHouse app package unless another engine needs them.

## Operation Semantics

`LogicalBackup` performs a full logical backup to S3-compatible object storage using ClickHouse native SQL:

```sql
BACKUP ALL TO S3('<url>', '<access_key>', '<secret_key>')
```

`Restore` restores from the same object path using native SQL:

```sql
RESTORE ALL FROM S3('<url>', '<access_key>', '<secret_key>')
```

The S3 URL is derived from `object_storage.endpoint`, `object_storage.bucket`, and `backup_file`. The implementation must normalize the path without dropping user-provided backup path segments.

`SetVariable` performs a local dynamic parameter change using ClickHouse SQL that persists beyond the current client session. The first implementation targets settings that can be applied as user-level settings:

```sql
ALTER USER <username> SETTINGS <key> = <value>
```

The API server and package metadata are responsible for validating that a requested key is dynamic and compatible with this SQL form. The agent still performs basic validation, including non-empty key/value and a conservative key character check, then executes the local ClickHouse SQL. ClickHouse remains the final authority for unsupported, readonly, or invalid settings.

## Error Handling

Errors follow the existing agent and GrpcCall behavior. Agent methods return an error, and the GrpcCall controller surfaces the failure in status instead of hiding it as successful reconciliation.

The ClickHouse app must preserve actionable errors for:

- Missing or invalid AES configuration.
- Missing `SECRET_MOUNT` or username secret file.
- SLM process check failure.
- Missing `clickhouse-client`.
- ClickHouse connection or authentication failure.
- Missing backup file name or required object storage fields.
- Invalid `SetVariable` key/value input.
- ClickHouse command stderr and non-zero exit status.

Keeper remains isolated. A Keeper UnitSet readiness failure is reflected in that UnitSet status; unit-operator does not add Keeper-specific backup, restore, or parameter operations.

## Status And Config Sync

No UnitSet or Unit status fields are added.

The API server can continue to derive state from:

- Unit phase.
- Unit `configSyncStatus`.
- Unit process state.
- UnitSet `units`.
- UnitSet `readyUnits`.
- UnitSet image, PVC, and resource sync status.

Config sync for both `clickhouse` and `clickhouse-keeper` uses the existing `SyncConfigService` contract and package ConfigMaps. Missing package ConfigMaps must fail in the same class of errors used by existing engines.

## Testing

Validation is split into focused layers:

1. Run `make pb-gen` after adding the ClickHouse proto and verify generated protobuf and gRPC bindings compile.
2. Add ClickHouse agent unit tests for connection environment defaults, request validation, S3 URL construction, SQL construction, command failure propagation, and `SetVariable` key validation.
3. Add or update GrpcCall controller tests for ClickHouse routing of `logical-backup`, `restore`, and `set-variable`, plus unsupported action behavior.
4. Regenerate CRDs and verify the GrpcCall type enum includes `clickhouse`.
5. Add or update examples for `clickhouse` and `clickhouse-keeper` UnitSets using version `26.3.9.8`.
6. Run the relevant Go tests and generated-code checks before implementation is considered complete.

## Acceptance Criteria

- A `clickhouse` UnitSet at version `26.3.9.8` reconciles into the requested two or three Units.
- A `clickhouse-keeper` UnitSet at version `26.3.9.8` reconciles into three Units when requested by the API server.
- Config sync succeeds for both types when package ConfigMaps are present.
- Config sync failure remains visible through existing Unit status behavior.
- ClickHouse `LogicalBackup`, `Restore`, and `SetVariable` can be invoked through the standard GrpcCall path.
- ClickHouse operation failures are surfaced as operation failures with useful messages.
- No ClickHouse topology CRD, switchover loop, replica repair loop, shard assignment controller, Keeper operation surface, or compose-operator dependency is added.
