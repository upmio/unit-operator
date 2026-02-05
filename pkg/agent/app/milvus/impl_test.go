package milvus

import "testing"

func TestServiceName(t *testing.T) {
	svc := &service{}
	if svc.Name() != appName {
		t.Fatalf("expected %s, got %s", appName, svc.Name())
	}
}
