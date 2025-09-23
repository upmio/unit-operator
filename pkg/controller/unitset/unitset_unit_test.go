package unitset

import (
	"encoding/json"
	"testing"

	upmiov1alpha2 "github.com/upmio/unit-operator/api/v1alpha2"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestFillEnvs(t *testing.T) {
	// Create test data
	unit := &upmiov1alpha2.Unit{
		ObjectMeta: metav1.ObjectMeta{
			Name: "testUnit",
		},
		Spec: upmiov1alpha2.UnitSpec{
			Template: v1.PodTemplateSpec{
				Spec: v1.PodSpec{
					InitContainers: []v1.Container{
						{
							Env: []v1.EnvVar{
								{Name: "INIT_CONTAINER_ENV", Value: "init-container-env-value"},
							},
						},
					},
					Containers: []v1.Container{
						{
							Env: []v1.EnvVar{
								{Name: "CONTAINER_ENV", Value: "container-env-value"},
							},
						},
					},
				},
			},
		},
	}

	unitset := &upmiov1alpha2.UnitSet{
		Spec: upmiov1alpha2.UnitSetSpec{
			Env: []v1.EnvVar{
				{Name: "UNITSET_ENV", Value: "unitset-env-value"},
			},
		},
	}

	mountEnvs := []v1.EnvVar{
		{Name: "MOUNT_ENV", Value: "mount-env-value"},
	}

	ports := []v1.ContainerPort{
		{Name: "http", ContainerPort: 80},
	}

	// Call the function to test
	fillEnvs(unit, unitset, mountEnvs, ports)

	// Verify the results
	if len(unit.Spec.Template.Spec.InitContainers[0].Env) != 3 {
		t.Errorf("Expected 3 environment variables for init container, got %d", len(unit.Spec.Template.Spec.InitContainers[0].Env))
	}

	if len(unit.Spec.Template.Spec.Containers[0].Env) != 3 {
		t.Errorf("Expected 3 environment variables for container, got %d", len(unit.Spec.Template.Spec.Containers[0].Env))
	}

	// Check the environment variables are in the correct order
	expectedEnvOrder := []string{"UNITSET_ENV", "MOUNT_ENV", "INIT_CONTAINER_ENV"}
	for i, env := range unit.Spec.Template.Spec.InitContainers[0].Env {
		if env.Name != expectedEnvOrder[i] {
			t.Errorf("Expected environment variable %s, got %s", expectedEnvOrder[i], env.Name)
		}
	}

	expectedEnvOrder = []string{"UNITSET_ENV", "MOUNT_ENV", "CONTAINER_ENV"}
	for i, env := range unit.Spec.Template.Spec.Containers[0].Env {
		if env.Name != expectedEnvOrder[i] {
			t.Errorf("Expected environment variable %s, got %s", expectedEnvOrder[i], env.Name)
		}
	}
}

func TestNodeNameMapAnnotationAppliedToAffinity(t *testing.T) {
	unitset := &upmiov1alpha2.UnitSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "mysql-cluster",
			Namespace:   "default",
			Annotations: map[string]string{},
		},
		Spec: upmiov1alpha2.UnitSetSpec{
			Type:  "mysql",
			Units: 1,
			Env:   []v1.EnvVar{},
		},
	}
	// node-name-map annotation
	m := map[string]string{"mysql-cluster-0": "node-a"}
	b, _ := json.Marshal(m)
	unitset.Annotations[upmiov1alpha2.AnnotationUnitsetNodeNameMap] = string(b)

	unitTemplate := upmiov1alpha2.Unit{
		ObjectMeta: metav1.ObjectMeta{},
		Spec: upmiov1alpha2.UnitSpec{
			Template: v1.PodTemplateSpec{Spec: v1.PodSpec{Containers: []v1.Container{{Name: "mysql"}}}},
		},
	}

	// personalize for unit-0
	unit := fillUnitPersonalizedInfo(unitTemplate, unitset, map[string]string{"mysql-cluster-0": "0"}, "mysql-cluster-0")

	if unit.Spec.Template.Spec.Affinity == nil || unit.Spec.Template.Spec.Affinity.NodeAffinity == nil || unit.Spec.Template.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution == nil {
		t.Fatalf("expected node affinity to be set from annotation")
	}
	terms := unit.Spec.Template.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms
	found := false
	for _, term := range terms {
		for _, me := range term.MatchExpressions {
			if me.Key == "kubernetes.io/hostname" && len(me.Values) == 1 && me.Values[0] == "node-a" {
				found = true
			}
		}
	}
	if !found {
		t.Errorf("expected hostname match expression for node-a")
	}
}

func TestAddEnvVar(t *testing.T) {
	tests := []struct {
		name     string
		envs     []v1.EnvVar
		newEnv   v1.EnvVar
		expected []v1.EnvVar
	}{
		{
			name: "Add new environment variable",
			envs: []v1.EnvVar{
				{Name: "EXISTING_VAR", Value: "existing_value"},
			},
			newEnv: v1.EnvVar{
				Name:  "NEW_VAR",
				Value: "new_value",
			},
			expected: []v1.EnvVar{
				{Name: "EXISTING_VAR", Value: "existing_value"},
				{Name: "NEW_VAR", Value: "new_value"},
			},
		},
		{
			name: "Don't add existing environment variable",
			envs: []v1.EnvVar{
				{Name: "EXISTING_VAR", Value: "existing_value"},
			},
			newEnv: v1.EnvVar{
				Name:  "EXISTING_VAR",
				Value: "new_value",
			},
			expected: []v1.EnvVar{
				{Name: "EXISTING_VAR", Value: "existing_value"},
			},
		},
		{
			name: "Add multiple new environment variables",
			envs: []v1.EnvVar{
				{Name: "EXISTING_VAR", Value: "existing_value"},
			},
			newEnv: v1.EnvVar{
				Name:  "NEW_VAR_1",
				Value: "new_value_1",
			},
			expected: []v1.EnvVar{
				{Name: "EXISTING_VAR", Value: "existing_value"},
				{Name: "NEW_VAR_1", Value: "new_value_1"},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := addEnvVar(test.envs, test.newEnv)
			if !envVarsEqual(actual, test.expected) {
				t.Errorf("expected %v, got %v", test.expected, actual)
			}
		})
	}
}

func envVarsEqual(a, b []v1.EnvVar) bool {
	if len(a) != len(b) {
		return false
	}
	for i, env := range a {
		if env.Name != b[i].Name || env.Value != b[i].Value {
			return false
		}
	}
	return true
}
