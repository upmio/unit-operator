/*
 * UPM for Enterprise
 *
 * Copyright (c) 2009-2025 SYNTROPY Pte. Ltd.
 * All rights reserved.
 *
 * This software is the confidential and proprietary information of
 * SYNTROPY Pte. Ltd. ("Confidential Information"). You shall not
 * disclose such Confidential Information and shall use it only in
 * accordance with the terms of the license agreement you entered
 * into with SYNTROPY.
 */

package v1alpha1

import (
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// UnitType defines the type of unit this GrpcCall will interact with.
// Currently supported types are "mysql", "proxysql" and "postgresql".
// +kubebuilder:validation:Enum=mysql;postgresql;proxysql
type UnitType string

const (
	// MysqlType represents a MySQL unit.
	MysqlType UnitType = "mysql"

	// PostgresqlType represents a PostgreSQL unit.
	PostgresqlType UnitType = "postgresql"

	// ProxysqlType represents a ProxySQL unit.
	ProxysqlType UnitType = "proxysql"

	// RedisType represents a Redis unit.
	RedisType UnitType = "redis"

	// SentinelType represents a Redis Sentinel unit.
	SentinelType UnitType = "redis-sentinel"
)

// Action defines the specific operation to be sent to the unit-agent.
// Each action corresponds to a gRPC method exposed by the unit-agent.
// +kubebuilder:validation:Enum=logical-backup;physical-backup;restore;gtid-purge;set-variable;clone
type Action string

const (
	// LogicalBackupAction instructs the agent to perform a logical backup.
	LogicalBackupAction Action = "logical-backup"

	// PhysicalBackupAction instructs the agent to perform a physical backup.
	PhysicalBackupAction Action = "physical-backup"

	// RestoreAction instructs the agent to restore from a backup.
	RestoreAction Action = "restore"

	// GtidPurgeAction instructs the agent to purge GTID information (specific to MySQL).
	GtidPurgeAction Action = "gtid-purge"

	// SetVariableAction instructs the agent to set runtime configuration parameters.
	SetVariableAction Action = "set-variable"

	// CloneAction instructs the agent to perform a clone operation from another instance.
	CloneAction Action = "clone"
)

// GrpcCallSpec defines the desired behavior of a GrpcCall custom resource.
// Each GrpcCall instance represents a single request to a unit-agent running in a unit pod.
type GrpcCallSpec struct {
	// TargetUnit is the name of the target Unit custom resource.
	// This identifies which unit's agent the request should be sent to.
	TargetUnit string `json:"targetUnit"`

	// Type specifies the type of the target unit (e.g., mysql, proxysql, postgresql).
	// This helps the operator determine how to format and route the request.
	Type UnitType `json:"type"`

	// Action specifies which gRPC method should be called on the unit-agent.
	Action Action `json:"action"`

	// ttlSecondsAfterFinished limits the lifetime of a Grpc Call that has finished
	// execution (either Complete or Failed). If this field is set,
	// ttlSecondsAfterFinished after the Grpc Call finishes, it is eligible to be
	// automatically deleted.
	TTLSecondsAfterFinished *int32 `json:"ttlSecondsAfterFinished"`

	// Parameters provides a flexible map of key-value pairs used as arguments
	// to the gRPC call. The exact keys depend on the action type.
	// For example: {"BACKUP_MODE": "physical", "S3_BUCKET": "my-bucket"}
	// +kubebuilder:pruning:PreserveUnknownFields
	Parameters map[string]apiextensionsv1.JSON `json:"parameters"`
}

// Result defines the outcome status of a GrpcCall execution.
// It represents the final state of the gRPC request sent to the unit-agent.
// +kubebuilder:validation:Enum=Success;Failed
type Result string

const (
	// SuccessResult indicates that the gRPC call was completed successfully.
	SuccessResult Result = "Success"

	// FailedResult indicates that the gRPC call failed due to an error during execution.
	FailedResult Result = "Failed"
)

// GrpcCallStatus defines the observed state of a GrpcCall.
// It records the execution result and related information returned
// by the unit-agent after invoking the specified gRPC action.
type GrpcCallStatus struct {
	// Result indicates the final outcome of the gRPC call.
	// Valid values: "Success", "Failed".
	Result Result `json:"result"`

	// Message contains additional context about the result,
	// such as error details, logs, or debug output.
	Message string `json:"message"`

	// CompletionTime is the timestamp when the gRPC call completed.
	// +optional
	CompletionTime *metav1.Time `json:"completionTime,omitempty"`

	// StartTime is the timestamp when the controller started processing the gRPC call.
	// +optional
	StartTime *metav1.Time `json:"startTime,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=gc
// +kubebuilder:printcolumn:name="RESULT",type=string,JSONPath=`.status.result`
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"

// GrpcCall is the Schema for the grpccalls API
type GrpcCall struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GrpcCallSpec   `json:"spec,omitempty"`
	Status GrpcCallStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// GrpcCallList contains a list of GrpcCall
type GrpcCallList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []GrpcCall `json:"items"`
}

func init() {
	SchemeBuilder.Register(&GrpcCall{}, &GrpcCallList{})
}
