package common

import (
	"context"
	"fmt"

	"github.com/upmio/unit-operator/pkg/agent/conf"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"

	unitv1alpha2 "github.com/upmio/unit-operator/api/v1alpha2"
)

type EventRecorder struct {
	unitClient client.Client
	record.EventRecorder
}

func NewEventRecorder() (*EventRecorder, error) {
	clientSet, err := conf.GetConf().GetClientSet()
	if err != nil {
		return nil, err
	}

	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: clientSet.CoreV1().Events("")})

	schema, err := unitv1alpha2.SchemeBuilder.Build()
	if err != nil {
		return nil, err
	}

	recorder := eventBroadcaster.NewRecorder(schema, v1.EventSource{Component: "unit-agent"})

	c, err := conf.GetConf().GetUnitClient()
	if err != nil {
		return nil, err
	}

	return &EventRecorder{
		unitClient:    c,
		EventRecorder: recorder,
	}, nil
}

func (s *EventRecorder) SendNormalEventToUnit(name, namespace, reason, msg string) error {
	return s.sendEventToUnit(context.Background(), name, namespace, reason, msg, v1.EventTypeNormal)
}

func (s *EventRecorder) SendWarningEventToUnit(name, namespace, reason, msg string) error {
	return s.sendEventToUnit(context.Background(), name, namespace, reason, msg, v1.EventTypeWarning)
}

// sendEventToUnit sends an event to a unit with the specified event type
func (s *EventRecorder) sendEventToUnit(ctx context.Context, name, namespace, reason, msg, eventType string) error {
	instance := &unitv1alpha2.Unit{}
	if err := s.unitClient.Get(ctx, types.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}, instance); err != nil {
		return fmt.Errorf("failed to fetch unit[%s/%s]: %w", namespace, name, err)
	}

	s.Event(instance, eventType, reason, msg)

	return nil
}
