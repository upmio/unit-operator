/*
 * UPM for Enterprise
 *
 * Copyright (c) 2009-2025 SYNTROPY Pte. Ltd.
 * All rights reserved.
 *
 * This software is the confidential and proprietary information of
 * SYNTROPY Pte. Ltd. ("Confidential Information"). You shall not
 * disclose such Confidential Information and shall use it only in
 * accordance with the terms of the license agreement you entered
 * into with SYNTROPY.
 */

package controller

import (
	"github.com/upmio/unit-operator/pkg/controller/grpccall"

	ctrl "sigs.k8s.io/controller-runtime"
)

// Setup creates all controllers with the supplied logger and adds
// them to the supplied manager.
func Setup(mgr ctrl.Manager) error {
	for _, setup := range []func(ctrl.Manager) error{
		grpccall.Setup,
	} {
		if err := setup(mgr); err != nil {
			return err
		}
	}
	return nil
}
