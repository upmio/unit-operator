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

	InPlacePodVerticalScalingEnabled = false
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
	if ipFamily != "" {
		klog.Infof("found env: [IP_FAMILY], only support [%s]...", ipFamily)
		IpFamily = ipFamily
	}
}
