package unitset

import (
	"context"
	"fmt"
	"strconv"
	"sync"

	upmiov1alpha2 "github.com/upmio/unit-operator/api/v1alpha2"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *UnitSetReconciler) reconcileHeadlessService(
	ctx context.Context,
	req ctrl.Request,
	unitset *upmiov1alpha2.UnitSet,
	ports upmiov1alpha2.Ports) error {

	headlessService := v1.Service{}
	exceptedServiceNamespacedName := client.ObjectKey{Name: unitset.HeadlessServiceName(), Namespace: unitset.Namespace}
	err := r.Get(ctx, exceptedServiceNamespacedName, &headlessService)
	if apierrors.IsNotFound(err) {

		ref := metav1.NewControllerRef(unitset, controllerKind)
		service := &v1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:            unitset.HeadlessServiceName(),
				Namespace:       unitset.Namespace,
				Labels:          make(map[string]string),
				OwnerReferences: []metav1.OwnerReference{*ref},
			},
			Spec: v1.ServiceSpec{
				Type:                     v1.ServiceTypeClusterIP,
				PublishNotReadyAddresses: true,
				ClusterIP:                "None",
				Selector:                 make(map[string]string),
			},
		}

		for _, p := range ports {
			intPort, convErr := strconv.Atoi(p.ContainerPort)
			if convErr != nil || intPort <= 0 || intPort > 65535 {
				// Skip invalid ports to avoid API validation errors
				continue
			}
			service.Spec.Ports = append(service.Spec.Ports, v1.ServicePort{
				Name:     p.Name,
				Port:     int32(intPort),
				Protocol: v1.Protocol(p.Protocol),
			})
		}

		// Copy labels from UnitSet to avoid sharing the same map reference
		if service.Labels == nil {
			service.Labels = make(map[string]string)
		}
		for k, v := range unitset.Labels {
			service.Labels[k] = v
		}

		service.Labels[upmiov1alpha2.UnitsetName] = unitset.Name
		service.Spec.Selector[upmiov1alpha2.UnitsetName] = unitset.Name

		err = r.Create(ctx, service)
		if err != nil {
			return fmt.Errorf("create headless service:[%s] error:[%s]", unitset.HeadlessServiceName(), err.Error())
		}

	} else if err != nil {
		return fmt.Errorf("get headless service:[%s] error:[%s]", unitset.HeadlessServiceName(), err.Error())
	}

	return nil
}

func (r *UnitSetReconciler) reconcileExternalService(
	ctx context.Context,
	req ctrl.Request,
	unitset *upmiov1alpha2.UnitSet,
	ports upmiov1alpha2.Ports) error {

	if unitset.Spec.ExternalService.Type == "" {
		klog.V(4).Infof("reconcileExternalService: unitset: [%s] spec.externalService.type is null,"+
			"no need generate external service", req.String())
		return nil
	}

	if unitset.Spec.SharedConfigName == "" {
		klog.V(4).Infof("reconcileExternalService: unitset name: [%s], not found shared config!!!", unitset.Name)
		return nil
	}

	externalService := v1.Service{}
	exceptedServiceNamespacedName := client.ObjectKey{Name: unitset.ExternalServiceName(), Namespace: unitset.Namespace}
	err := r.Get(ctx, exceptedServiceNamespacedName, &externalService)
	if apierrors.IsNotFound(err) {

		ref := metav1.NewControllerRef(unitset, controllerKind)
		service := &v1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:            unitset.ExternalServiceName(),
				Namespace:       unitset.Namespace,
				Labels:          make(map[string]string),
				OwnerReferences: []metav1.OwnerReference{*ref},
			},
			Spec: v1.ServiceSpec{
				Type:                     v1.ServiceType(unitset.Spec.ExternalService.Type),
				PublishNotReadyAddresses: true,
				//ClusterIP:                "None",
				Selector: make(map[string]string),
			},
		}

		for _, p := range ports {
			intPort, convErr := strconv.Atoi(p.ContainerPort)
			if convErr != nil || intPort <= 0 || intPort > 65535 {
				continue
			}
			service.Spec.Ports = append(service.Spec.Ports, v1.ServicePort{
				Name:     p.Name,
				Port:     int32(intPort),
				Protocol: v1.Protocol(p.Protocol),
			})
		}

		if service.Labels == nil {
			service.Labels = make(map[string]string)
		}
		for k, v := range unitset.Labels {
			service.Labels[k] = v
		}

		service.Labels[upmiov1alpha2.UnitsetName] = unitset.Name
		service.Spec.Selector[upmiov1alpha2.UnitsetName] = unitset.Name

		err = r.Create(ctx, service)
		if err != nil {
			return fmt.Errorf("create external service:[%s] error:[%s]", unitset.ExternalServiceName(), err.Error())
		}

	} else if err != nil {
		return fmt.Errorf("get external service:[%s] error:[%s]", unitset.ExternalServiceName(), err.Error())
	}

	return nil
}

func (r *UnitSetReconciler) reconcileUnitService(
	ctx context.Context,
	req ctrl.Request,
	unitset *upmiov1alpha2.UnitSet,
	ports upmiov1alpha2.Ports) error {

	if unitset.Spec.UnitService.Type == "" {
		klog.Infof("reconcileUnitService: unitset: [%s] spec.unitService.type is null,"+
			"no need generate unit service", req.String())
		return nil
	}

	if unitset.Spec.SharedConfigName == "" {
		klog.V(4).Infof("reconcileUnitService: unitset name: [%s], not found shared config!!!", unitset.Name)
		return nil
	}

	unitNames, _ := unitset.UnitNames()
	ref := metav1.NewControllerRef(unitset, controllerKind)

	errs := []error{}
	var wg sync.WaitGroup
	var mu sync.Mutex
	createdAny := false
	for _, unitName := range unitNames {
		wg.Add(1)
		go func(unitName string) {
			defer wg.Done()

			unitService := v1.Service{}
			unitServiceName := fmt.Sprintf("%s-svc", unitName)
			unitServiceNamespacedName := client.ObjectKey{Name: unitServiceName, Namespace: unitset.Namespace}
			err := r.Get(ctx, unitServiceNamespacedName, &unitService)
			if apierrors.IsNotFound(err) {

				service := &v1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:            unitServiceName,
						Namespace:       unitset.Namespace,
						Labels:          make(map[string]string),
						OwnerReferences: []metav1.OwnerReference{*ref},
					},
					Spec: v1.ServiceSpec{
						Type:                     v1.ServiceType(unitset.Spec.UnitService.Type),
						PublishNotReadyAddresses: true,
						Selector:                 make(map[string]string),
					},
				}

				for _, p := range ports {
					intPort, convErr := strconv.Atoi(p.ContainerPort)
					if convErr != nil || intPort <= 0 || intPort > 65535 {
						continue
					}
					service.Spec.Ports = append(service.Spec.Ports, v1.ServicePort{
						Name:     p.Name,
						Port:     int32(intPort),
						Protocol: v1.Protocol(p.Protocol),
					})
				}

				if service.Labels == nil {
					service.Labels = make(map[string]string)
				}
				for k, v := range unitset.Labels {
					service.Labels[k] = v
				}

				service.Labels[upmiov1alpha2.UnitName] = unitName
				service.Spec.Selector[upmiov1alpha2.UnitName] = unitName

				err = r.Create(ctx, service)
				if err != nil {
					errs = append(errs, fmt.Errorf("create unit service:[%s] error:[%s]", unitServiceName, err.Error()))
					return
				}

				mu.Lock()
				createdAny = true
				mu.Unlock()

			} else if err != nil {
				errs = append(errs, fmt.Errorf("get unit service:[%s] error:[%s]", unitServiceName, err.Error()))
				return
			}

		}(unitName)
	}
	wg.Wait()

	// If any unit service was created in this reconcile, annotate unitset with service type
	if createdAny {
		original := unitset.DeepCopy()
		if unitset.Annotations == nil {
			unitset.Annotations = map[string]string{}
		}
		unitset.Annotations[upmiov1alpha2.AnnotationUnitServiceType] = unitset.Spec.UnitService.Type
		if _, pErr := r.patchUnitset(ctx, original, unitset); pErr != nil {
			mu.Lock()
			errs = append(errs, fmt.Errorf("annotate unitset with unit service type error:[%s]", pErr.Error()))
			mu.Unlock()
		}
	}

	err := utilerrors.NewAggregate(errs)
	if err != nil {
		return err
	}

	return nil
}
