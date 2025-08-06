package v1alpha2

import (
	"fmt"
	"strconv"
)

func (u *UnitSet) UnitNames() ([]string, map[string]string) {
	//unitNames := make([]string, u.Spec.Units)
	unitNames := []string{}
	unitNameWithIndex := make(map[string]string)

	for i := 0; i < u.Spec.Units; i++ {
		unitName := fmt.Sprintf("%s-%d", u.Name, i)
		unitNames = append(unitNames, unitName)
		unitNameWithIndex[unitName] = strconv.Itoa(i)
	}

	return unitNames, unitNameWithIndex
}

// TemplateConfigTemplateName configtemplate Name
func (u *UnitSet) TemplateConfigTemplateName() string {
	if u.Spec.Edition == "" {
		return fmt.Sprintf("%s-%s-config-template", u.Spec.Type, u.Spec.Version)
	}

	return fmt.Sprintf("%s-%s-%s-config-template", u.Spec.Type, u.Spec.Edition, u.Spec.Version)
}

// ConfigTemplateName unitset configtemplate Name
func (u *UnitSet) ConfigTemplateName() string {
	return fmt.Sprintf("%s-config-template", u.Name)
}

// ConfigValueName unitset configvalue Name
func (u *UnitSet) ConfigValueName(unitName string) string {
	return fmt.Sprintf("%s-config-value", unitName)
}

// TemplateConfigValueName configvalue Name
func (u *UnitSet) TemplateConfigValueName() string {
	if u.Spec.Edition == "" {
		return fmt.Sprintf("%s-%s-config-value", u.Spec.Type, u.Spec.Version)
	}

	return fmt.Sprintf("%s-%s-%s-config-value", u.Spec.Type, u.Spec.Edition, u.Spec.Version)
}

func (u *UnitSet) HeadlessServiceName() string {
	return fmt.Sprintf("%s-headless-svc", u.Name)
}

func (u *UnitSet) ExternalServiceName() string {
	return fmt.Sprintf("%s-svc", u.Name)
}

func (u *UnitSet) TemplatePodTemplateName() string {
	if u.Spec.Edition == "" {
		return fmt.Sprintf("%s-%s", u.Spec.Type, u.Spec.Version)
	}

	return fmt.Sprintf("%s-%s-%s", u.Spec.Type, u.Spec.Edition, u.Spec.Version)
}

func (u *UnitSet) PodTemplateName() string {
	return fmt.Sprintf("%s-podtemplate", u.Name)
}
