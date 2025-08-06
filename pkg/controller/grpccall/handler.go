package grpccall

import (
	"context"
	"encoding/json"
	"fmt"
	upmv1alpha1 "github.com/upmio/unit-operator/api/v1alpha1"
	"github.com/upmio/unit-operator/pkg/agent/app/mysql"
	"github.com/upmio/unit-operator/pkg/agent/app/postgresql"
	"github.com/upmio/unit-operator/pkg/agent/app/proxysql"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

type messageGetter interface {
	GetMessage() string
}

// unmarshalParams serializes the raw Parameters map to JSON
// and unmarshals into the provided proto message.
func unmarshalParams(params map[string]apiextensionsv1.JSON, msg proto.Message) error {
	data, err := json.Marshal(params)
	if err != nil {
		return err
	}
	if err := protojson.Unmarshal(data, msg); err != nil {
		return err
	}
	return nil
}

// handleGrpcCall processes a GrpcCall CR by routing to the proper client
// stub based on unit type and action, handling request construction, execution,
// and status updates in a DRY manner.
func (r *ReconcileGrpcCall) handleGrpcCall(
	ctx context.Context,
	instance *upmv1alpha1.GrpcCall,
	c *Client,
) error {
	var (
		newReq func() proto.Message
		callFn func(ctx context.Context, msg proto.Message) (messageGetter, error)
	)

	switch instance.Spec.Type {
	case upmv1alpha1.MysqlType:
		mc := c.Mysql()
		switch instance.Spec.Action {
		case upmv1alpha1.PhysicalBackupAction:
			newReq = func() proto.Message { return &mysql.PhysicalBackupRequest{} }
			callFn = func(ctx context.Context, msg proto.Message) (messageGetter, error) {
				return mc.PhysicalBackup(ctx, msg.(*mysql.PhysicalBackupRequest))
			}
		case upmv1alpha1.LogicalBackupAction:
			newReq = func() proto.Message { return &mysql.LogicalBackupRequest{} }
			callFn = func(ctx context.Context, msg proto.Message) (messageGetter, error) {
				return mc.LogicalBackup(ctx, msg.(*mysql.LogicalBackupRequest))
			}
		case upmv1alpha1.CloneAction:
			newReq = func() proto.Message { return &mysql.CloneRequest{} }
			callFn = func(ctx context.Context, msg proto.Message) (messageGetter, error) {
				return mc.Clone(ctx, msg.(*mysql.CloneRequest))
			}
		case upmv1alpha1.GtidPurgeAction:
			newReq = func() proto.Message { return &mysql.GtidPurgeRequest{} }
			callFn = func(ctx context.Context, msg proto.Message) (messageGetter, error) {
				return mc.GtidPurge(ctx, msg.(*mysql.GtidPurgeRequest))
			}
		case upmv1alpha1.SetVariableAction:
			newReq = func() proto.Message { return &mysql.SetVariableRequest{} }
			callFn = func(ctx context.Context, msg proto.Message) (messageGetter, error) {
				return mc.SetVariable(ctx, msg.(*mysql.SetVariableRequest))
			}
		case upmv1alpha1.RestoreAction:
			newReq = func() proto.Message { return &mysql.RestoreRequest{} }
			callFn = func(ctx context.Context, msg proto.Message) (messageGetter, error) {
				return mc.Restore(ctx, msg.(*mysql.RestoreRequest))
			}
		default:
			return fmt.Errorf("unsupported action %q for type %q", instance.Spec.Action, instance.Spec.Type)
		}
	case upmv1alpha1.PostgresqlType:
		pc := c.Postgresql()
		switch instance.Spec.Action {
		case upmv1alpha1.PhysicalBackupAction:
			newReq = func() proto.Message { return &postgresql.PhysicalBackupRequest{} }
			callFn = func(ctx context.Context, msg proto.Message) (messageGetter, error) {
				return pc.PhysicalBackup(ctx, msg.(*postgresql.PhysicalBackupRequest))
			}
		case upmv1alpha1.LogicalBackupAction:
			newReq = func() proto.Message { return &postgresql.LogicalBackupRequest{} }
			callFn = func(ctx context.Context, msg proto.Message) (messageGetter, error) {
				return pc.LogicalBackup(ctx, msg.(*postgresql.LogicalBackupRequest))
			}
		case upmv1alpha1.RestoreAction:
			newReq = func() proto.Message { return &postgresql.RestoreRequest{} }
			callFn = func(ctx context.Context, msg proto.Message) (messageGetter, error) {
				return pc.Restore(ctx, msg.(*postgresql.RestoreRequest))
			}
		default:
			return fmt.Errorf("unsupported action %q for type %q", instance.Spec.Action, instance.Spec.Type)
		}
	case upmv1alpha1.ProxysqlType:
		pc := c.Proxysql()
		switch instance.Spec.Action {
		case upmv1alpha1.SetVariableAction:
			newReq = func() proto.Message { return &proxysql.SetVariableRequest{} }
			callFn = func(ctx context.Context, msg proto.Message) (messageGetter, error) {
				return pc.SetVariable(ctx, msg.(*proxysql.SetVariableRequest))
			}
		default:
			return fmt.Errorf("unsupported action %q for type %q", instance.Spec.Action, instance.Spec.Type)
		}
	default:
		return fmt.Errorf("unsupported unit type %q", instance.Spec.Type)
	}

	// Construct request, execute call, update status
	req := newReq()
	if err := unmarshalParams(instance.Spec.Parameters, req); err != nil {
		return fmt.Errorf("failed to unmarshal parameters: %v", err)
	}

	resp, err := callFn(ctx, req)
	if err != nil {
		return err
	}

	instance.Status.Message = resp.GetMessage()
	instance.Status.Result = upmv1alpha1.SuccessResult

	return nil
}
