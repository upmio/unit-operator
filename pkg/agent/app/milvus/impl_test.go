package milvus

import (
	"encoding/json"
	"net"
	"testing"
)

func TestServiceName(t *testing.T) {
	svc := &service{}
	if svc.Name() != appName {
		t.Fatalf("expected %s, got %s", appName, svc.Name())
	}
}

func TestGetEtcdMemberList(t *testing.T) {
	etcdMemberList := []byte(`["jvcbhiiw-etcd-t62-0.jvcbhiiw-etcd-t62-headless-svc.test.svc.cluster.local","jvcbhiiw-etcd-t62-1.jvcbhiiw-etcd-t62-headless-svc.test.svc.cluster.local","jvcbhiiw-etcd-t62-2.jvcbhiiw-etcd-t62-headless-svc.test.svc.cluster.local"]`)

	memList := make([]string, 0)

	err := json.Unmarshal(etcdMemberList, &memList)
	if err != nil {
		t.Error(err)
	}

	uri := make([]string, 0)
	for _, member := range memList {
		endpoint := net.JoinHostPort(member, "2379")
		uri = append(uri, endpoint)
	}

	t.Log(uri)
}
