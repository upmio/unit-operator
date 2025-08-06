package unitset

import (
	upmiov1alpha2 "github.com/upmio/unit-operator/api/v1alpha2"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
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

	ports := upmiov1alpha2.Ports{
		{Name: "http", ContainerPort: "80"},
	}

	// Call the function to test
	fillEnvs(unit, unitset, mountEnvs, ports)

	// Verify the results
	if len(unit.Spec.Template.Spec.InitContainers[0].Env) != 4 {
		t.Errorf("Expected 4 environment variables for init container, got %d", len(unit.Spec.Template.Spec.InitContainers[0].Env))
	}

	if len(unit.Spec.Template.Spec.Containers[0].Env) != 4 {
		t.Errorf("Expected 4 environment variables for container, got %d", len(unit.Spec.Template.Spec.Containers[0].Env))
	}

	// Check the environment variables are in the correct order
	expectedEnvOrder := []string{"UNITSET_ENV", "MOUNT_ENV", "HTTP_PORT", "INIT_CONTAINER_ENV"}
	for i, env := range unit.Spec.Template.Spec.InitContainers[0].Env {
		if env.Name != expectedEnvOrder[i] {
			t.Errorf("Expected environment variable %s, got %s", expectedEnvOrder[i], env.Name)
		}
	}

	expectedEnvOrder = []string{"UNITSET_ENV", "MOUNT_ENV", "HTTP_PORT", "CONTAINER_ENV"}
	for i, env := range unit.Spec.Template.Spec.Containers[0].Env {
		if env.Name != expectedEnvOrder[i] {
			t.Errorf("Expected environment variable %s, got %s", expectedEnvOrder[i], env.Name)
		}
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
			name: "添加新的环境变量",
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
			name: "不添加已经存在的环境变量",
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
			name: "添加多个新的环境变量",
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
