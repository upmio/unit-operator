package v1alpha2

const (
	UnitsetName = "unitset.name"
	UnitName    = "unit.name"
	UnitSn      = "unit.sn"

	NoneSetFlag = "noneSet"

	AnnotationMaintenance          = "maintenance"
	AnnotationMainContainerName    = "kubectl.kubernetes.io/default-container"
	AnnotationMainContainerVersion = "kubectl.kubernetes.io/default-container-version"
	AnnotationForceDelete          = "upm.io/force-delete"
	AnnotationLastUnitBelongNode   = "last.unit.belong.node"
	// AnnotationUnitsetNodeNameMap stores a JSON object mapping unit name -> node name (or "noneSet")
	// Example: {"mysql-cluster-0":"node-a","mysql-cluster-1":"noneSet"}
	AnnotationUnitsetNodeNameMap = "unit-operator.unit.node-name.map"
	// AnnotationUnitServiceType is the type of the unit service
	// Example: "ClusterIP"
	AnnotationUnitServiceType = "unit-operator.unit-service.type"

	// AnnotationUnitServiceNodeportMapPrefix
	// AnnotationUnitServiceNodeportMapSuffix
	// annotation real name: unit-operator.unit-service.<port name>.nodeport.map
	// Example: unit-operator.unit-service.http.nodeport.map
	// value is a JSON object mapping unit name -> nodePort
	// Example: {"mysql-cluster-0":30468,"mysql-cluster-1":30469}
	// if the annotation is not empty, when recreate the unit service, the nodePort will be filled from the annotation
	AnnotationUnitServiceNodeportMapPrefix = "unit-operator.unit-service."
	AnnotationUnitServiceNodeportMapSuffix = ".nodeport.map"

	FinalizerUnitDelete      = "upm.io/unit-delete"
	FinalizerConfigMapDelete = "upm.io/configmap-delete"
	FinalizerPodDelete       = "upm.io/pod-delete"
	FinalizerPvcDelete       = "upm.io/pvc-delete"

	CertmanagerIssuerSuffix      = "certmanager-issuer"
	CertmanagerCertificateSuffix = "certmanager-ca"
	CertmanagerSecretNameSuffix  = "secret"
)

// UnitPhase is a label for the condition of a pod at the current time.
// +enum
type UnitPhase string

// These are the valid statuses of pods.
const (
	// UnitPending means the pod has been accepted by the system, but one or more of the containers
	// has not been started. This includes time before being bound to a node, as well as time spent
	// pulling images onto the host.
	UnitPending UnitPhase = "Pending"
	// UnitRunning means the pod has been bound to a node and all of the containers have been started.
	// At least one container is still running or is in the process of being restarted.
	UnitRunning UnitPhase = "Running"
	// UnitReady means the pod Running and ready condition = true
	UnitReady UnitPhase = "Ready"
	// UnitSucceeded means that all containers in the pod have voluntarily terminated
	// with a container exit code of 0, and the system is not going to restart any of these containers.
	UnitSucceeded UnitPhase = "Succeeded"
	// UnitFailed means that all containers in the pod have terminated, and at least one container has
	// terminated in a failure (exited with a non-zero exit code or was stopped by the system).
	UnitFailed UnitPhase = "Failed"
	// UnitUnknown means that for some reason the state of the pod could not be obtained, typically due
	// to an error in communicating with the host of the pod.
	// Deprecated: It isn't being set since 2015 (74da3b14b0c0f658b3bb8d2def5094686d0e9095)
	UnitUnknown UnitPhase = "Unknown"
)
