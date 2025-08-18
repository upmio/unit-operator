package vars

import (
	"k8s.io/klog/v2"
	"os"
)

var (
	UnitAgentName     = "unit-agent"
	UnitAgentImage    string
	UnitAgentHostType = "domain"

	ManagerNamespace = "upm-system"
	ProjectName      = "unit-operator"
	IpFamily         = "IPv4"
)

func init() {
	managerNamespace := os.Getenv("NAMESPACE")
	if managerNamespace == "" {
		klog.Fatalf("not found env: [NAMESPACE], can't start service...")
	} else {
		ManagerNamespace = managerNamespace
	}

	ipFamily := os.Getenv("IP_FAMILY")
	//if ipFamily == "" {
	//	klog.Infof("not found env: [IP_FAMILY], only support [SingleStack:IPv4]...")
	//} else {
	//	klog.Infof("found env: [IP_FAMILY], only support [%s]...", ipFamily)
	//	IpFamily = ipFamily
	//}

	if ipFamily != "" {
		klog.Infof("found env: [IP_FAMILY], only support [%s]...", ipFamily)
		IpFamily = ipFamily
	}
}

const (
	ServiceMonitorCrdName           = "servicemonitors.monitoring.coreos.com"
	PodMonitorCrdName               = "podmonitors.monitoring.coreos.com"
	MonitorServiceMonitorNameSuffix = "-exporter-svcmon"
	EnvNameUnitName                 = "UNIT_NAME"
	LastUnitBelongNodeAnnotation    = "last.unit.belong.node"
)
