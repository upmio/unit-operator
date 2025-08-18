package v1alpha2

import "fmt"

// MainContainerName returns the name of the main container
func (u *Unit) MainContainerName() string {
	return u.Spec.Template.Annotations[AnnotationMainContainerName]
}

func PersistentVolumeClaimName(unit *Unit, volume string) string {
	return fmt.Sprintf("%s-%s", unit.Name, volume)
}
func GetPersistentVolumeName(unit *Unit, volume string) string {
	return fmt.Sprintf("%s-%s", unit.Name, volume)
}

func UnitsetHeadlessSvcName(unit *Unit) string {
	return fmt.Sprintf("%s-headless-svc", unit.Labels[UnitsetName])
}
