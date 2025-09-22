package vars

import (
	"os"
	"strings"

	"k8s.io/klog/v2"
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
		if strings.HasSuffix(os.Args[0], ".test") {
			managerNamespace = ManagerNamespace
			_ = os.Setenv("NAMESPACE", managerNamespace)
		} else {
			klog.Fatalf("not found env: [NAMESPACE], can't start service...")
		}
	}
	ManagerNamespace = managerNamespace

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
