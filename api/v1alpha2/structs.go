package v1alpha2

type PortInfo struct {
	Name          string `json:"name,omitempty"`
	ContainerPort string `json:"containerPort,omitempty"`
	Protocol      string `json:"protocol,omitempty"`
}

type Ports []PortInfo
