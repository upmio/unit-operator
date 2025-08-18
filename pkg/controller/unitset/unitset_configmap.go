package unitset

import (
	"context"
	"fmt"
	upmiov1alpha2 "github.com/upmio/unit-operator/api/v1alpha2"
	"github.com/upmio/unit-operator/pkg/utils/config"
	"github.com/upmio/unit-operator/pkg/vars"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sync"
)

func (r *UnitSetReconciler) reconcileConfigmap(
	ctx context.Context,
	req ctrl.Request,
	unitset *upmiov1alpha2.UnitSet) error {
	//upm-system           mysql-community-8.0.41-config-template	One for each unitset
	configTemplateName := unitset.ConfigTemplateName()
	cm := v1.ConfigMap{}
	err := r.Get(ctx, client.ObjectKey{Name: configTemplateName, Namespace: req.Namespace}, &cm)
	if apierrors.IsNotFound(err) {
		templateConfigTemplateName := unitset.TemplateConfigTemplateName()
		templateCm := v1.ConfigMap{}
		err = r.Get(ctx, client.ObjectKey{Name: templateConfigTemplateName, Namespace: vars.ManagerNamespace}, &templateCm)
		if err != nil {
			return err
		}

		ref := metav1.NewControllerRef(unitset, controllerKind)
		cm = v1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:            configTemplateName,
				Namespace:       req.Namespace,
				Labels:          make(map[string]string),
				Annotations:     make(map[string]string),
				OwnerReferences: []metav1.OwnerReference{*ref},
			},
		}

		cm.Data = templateCm.Data
		cm.Labels = unitset.Labels
		cm.Annotations[upmiov1alpha2.AnnotationMainContainerVersion] = unitset.Spec.Version

		err = r.Create(ctx, &cm)
		if err != nil {
			return fmt.Errorf("[reconcileConfigmap] [CREATE] create config template cm:[%s] error:[%s]", configTemplateName, err.Error())
		}

	} else if err != nil {
		return fmt.Errorf("[reconcileConfigmap] [CREATE] get config template cm:[%s] error:[%s]", configTemplateName, err.Error())
	}

	// when update image, compare cm.Annotations[upmiov1alpha2.AnnotationMainContainerVersion]
	// Initialize annotations map when nil to avoid panic
	if cm.Annotations == nil {
		cm.Annotations = make(map[string]string)
	}
	if cm.Annotations[upmiov1alpha2.AnnotationMainContainerVersion] != unitset.Spec.Version {
		klog.Infof("[reconcileConfigmap] unitset:[%s] version update [old:%s, new:%s], trigger update [template configmap]",
			req.String(), cm.Annotations[upmiov1alpha2.AnnotationMainContainerVersion], unitset.Spec.Version)

		templateConfigTemplateName := unitset.TemplateConfigTemplateName()
		templateCm := v1.ConfigMap{}
		err = r.Get(ctx, client.ObjectKey{Name: templateConfigTemplateName, Namespace: vars.ManagerNamespace}, &templateCm)
		if err != nil {
			return fmt.Errorf("[reconcileConfigmap] [UPDATE] get template config template cm:[%s] error:[%s]", templateConfigTemplateName, err.Error())
		}

		cm.Data = templateCm.Data
		// Ensure annotations map is initialized
		if cm.Annotations == nil {
			cm.Annotations = make(map[string]string)
		}
		cm.Annotations[upmiov1alpha2.AnnotationMainContainerVersion] = unitset.Spec.Version

		err = r.Update(ctx, &cm)
		if err != nil {
			return fmt.Errorf("[reconcileConfigmap] [UPDATE] update config template cm:[%s] error:[%s]", configTemplateName, err.Error())
		}
	}

	//upm-system           mysql-community-8.0.41-config-value		One for each unit
	unitNames, _ := unitset.UnitNames()
	errs := []error{}
	var wg sync.WaitGroup
	for _, unitName := range unitNames {
		wg.Add(1)
		go func(unitName string) {
			defer wg.Done()

			if unitName != "" {
				err = r.reconcileConfigTemplateValue(ctx, req, unitset, unitName)
				if err != nil {
					errs = append(errs, err)
					return
				}
			}

		}(unitName)

	}
	wg.Wait()

	err = utilerrors.NewAggregate(errs)
	if err != nil {
		return err
	}

	return nil
}

func (r *UnitSetReconciler) reconcileConfigTemplateValue(
	ctx context.Context,
	req ctrl.Request,
	unitset *upmiov1alpha2.UnitSet,
	unitName string) error {

	// Ignore empty unit names to avoid invalid resource names
	if unitName == "" {
		return nil
	}
	klog.V(4).Infof("reconcileConfigTemplateValue unitset:[%s], unitName:[%s]", req.String(), unitName)

	//upm-system           mysql-community-8.0.41-config-value		One for each unit
	configValueName := unitset.ConfigValueName(unitName)
	cm := v1.ConfigMap{}
	err := r.Get(ctx, client.ObjectKey{Name: configValueName, Namespace: req.Namespace}, &cm)
	if apierrors.IsNotFound(err) {
		templateConfigValueName := unitset.TemplateConfigValueName()
		templateConfigValueCm := v1.ConfigMap{}
		err = r.Get(ctx, client.ObjectKey{Name: templateConfigValueName, Namespace: vars.ManagerNamespace}, &templateConfigValueCm)
		if err != nil {
			return fmt.Errorf("[reconcileConfigTemplateValue] [CREATE] get template config value cm:[%s] error:[%s]", templateConfigValueName, err.Error())
		}

		ref := metav1.NewControllerRef(unitset, controllerKind)
		cm = v1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:            configValueName,
				Namespace:       req.Namespace,
				Labels:          make(map[string]string),
				Annotations:     make(map[string]string),
				OwnerReferences: []metav1.OwnerReference{*ref},
			},
		}

		cm.Data = templateConfigValueCm.Data
		cm.Labels = unitset.Labels
		cm.Annotations[upmiov1alpha2.AnnotationMainContainerVersion] = unitset.Spec.Version
		err = r.Create(ctx, &cm)
		if err != nil {
			return fmt.Errorf("[reconcileConfigTemplateValue] [CREATE] create config value cm:[%s] error:[%s]", configValueName, err.Error())
		}

	} else if err != nil {
		return fmt.Errorf("[reconcileConfigTemplateValue] [CREATE] get config value cm:[%s] error:[%s]", configValueName, err.Error())
	}

	// Ensure annotations map is initialized before reading/updating
	if cm.Annotations == nil {
		cm.Annotations = make(map[string]string)
	}
	// when update image, compare cm.Annotations[upmiov1alpha2.AnnotationMainContainerVersion]
	if cm.Annotations[upmiov1alpha2.AnnotationMainContainerVersion] != unitset.Spec.Version {
		klog.Infof("[reconcileConfigTemplateValue] [UPDATE] unitset:[%s] version update [old:%s, new:%s], trigger update [value configmap]",
			req.String(), cm.Annotations[upmiov1alpha2.AnnotationMainContainerVersion], unitset.Spec.Version)

		templateConfigValueName := unitset.TemplateConfigValueName()
		templateConfigValueCm := v1.ConfigMap{}
		err = r.Get(ctx, client.ObjectKey{Name: templateConfigValueName, Namespace: vars.ManagerNamespace}, &templateConfigValueCm)
		if err != nil {
			return fmt.Errorf("[reconcileConfigTemplateValue] [UPDATE] get template config value cm:[%s] error:[%s]", templateConfigValueName, err.Error())
		}

		configTemplateValueContent, ok := templateConfigValueCm.Data[unitset.Spec.Type]
		if !ok {
			return nil
		}

		// Ensure data map is initialized
		if cm.Data == nil {
			cm.Data = make(map[string]string)
		}
		// If original key missing, seed with template directly
		originalValueContent, hasOriginal := cm.Data[unitset.Spec.Type]
		if !hasOriginal {
			cm.Data[unitset.Spec.Type] = configTemplateValueContent
		} else {
			// Determine if the original equals the old template (i.e., not customized)
			oldVersion := cm.Annotations[upmiov1alpha2.AnnotationMainContainerVersion]
			var oldTemplateValueName string
			if unitset.Spec.Edition == "" {
				oldTemplateValueName = fmt.Sprintf("%s-%s-config-value", unitset.Spec.Type, oldVersion)
			} else {
				oldTemplateValueName = fmt.Sprintf("%s-%s-%s-config-value", unitset.Spec.Type, unitset.Spec.Edition, oldVersion)
			}

			oldTemplateValueCm := v1.ConfigMap{}
			oldTemplateValueContent := ""
			if getErr := r.Get(ctx, client.ObjectKey{Name: oldTemplateValueName, Namespace: vars.ManagerNamespace}, &oldTemplateValueCm); getErr == nil {
				if v, ok := oldTemplateValueCm.Data[unitset.Spec.Type]; ok {
					oldTemplateValueContent = v
				}
			}

			if oldTemplateValueContent != "" && oldTemplateValueContent == originalValueContent {
				// Not customized: adopt the new template as-is
				cm.Data[unitset.Spec.Type] = configTemplateValueContent
			} else {
				// Customized: overlay original keys onto new template
				originalValueConfiger, err := config.NewViper(originalValueContent, "yaml")
				if err != nil {
					return fmt.Errorf("[reconcileConfigTemplateValue] [UPDATE] configmap:[%s], get new viper failed: [%s]", cm.Name, err.Error())
				}

				configTemplateValueConfiger, err := config.NewViper(configTemplateValueContent, "yaml")
				if err != nil {
					return fmt.Errorf("[reconcileConfigTemplateValue] [UPDATE] configmap:[%s], get new viper failed: [%s]", templateConfigValueCm.Name, err.Error())
				}

				for _, one := range originalValueConfiger.AllKeys() {
					configTemplateValueConfiger.Set(one, originalValueConfiger.Get(one))
				}

				newConfigStr, err := config.Viper2String(configTemplateValueConfiger)
				if err != nil {
					return fmt.Errorf("[reconcileConfigTemplateValue] [UPDATE] configmap:[%s], configer to string failed: [%s]", cm.Name, err.Error())
				}

				cm.Data[unitset.Spec.Type] = newConfigStr
			}
		}
		cm.Annotations[upmiov1alpha2.AnnotationMainContainerVersion] = unitset.Spec.Version

		err = r.Update(ctx, &cm)
		if err != nil {
			return fmt.Errorf("[reconcileConfigTemplateValue] [UPDATE] update config value cm:[%s] error:[%s]", configValueName, err.Error())
		}
	}

	return nil
}
