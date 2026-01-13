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

package v1alpha2

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	upmv1alpha2 "github.com/upmio/unit-operator/api/v1alpha2"
)

var _ = Describe("Unit Webhook", func() {

	Context("When creating Unit under Defaulting Webhook", func() {
		It("Should fill in the default value if a required field is empty", func() {

			// TODO(user): Add your logic here

		})
	})

	Context("When creating Unit under Validating Webhook", func() {
		It("Should deny if configTemplateName is empty", func() {
			wh := &unitAdmission{}
			unit := &upmv1alpha2.Unit{ObjectMeta: metav1.ObjectMeta{Name: "u", Namespace: "default"}}
			unit.Spec.ConfigTemplateName = ""
			unit.Spec.ConfigValueName = "cv"

			_, err := wh.ValidateCreate(context.Background(), unit)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("spec.configTemplateName is required"))
		})

		It("Should deny if configValueName is empty", func() {
			wh := &unitAdmission{}
			unit := &upmv1alpha2.Unit{ObjectMeta: metav1.ObjectMeta{Name: "u", Namespace: "default"}}
			unit.Spec.ConfigTemplateName = "ct"
			unit.Spec.ConfigValueName = ""

			_, err := wh.ValidateCreate(context.Background(), unit)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("spec.configValueName is required"))
		})

		It("Should admit if all required fields are provided", func() {
			wh := &unitAdmission{}
			unit := &upmv1alpha2.Unit{ObjectMeta: metav1.ObjectMeta{Name: "u", Namespace: "default"}}
			unit.Spec.ConfigTemplateName = "ct"
			unit.Spec.ConfigValueName = "cv"

			_, err := wh.ValidateCreate(context.Background(), unit)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("When updating Unit under Validating Webhook", func() {
		It("Should deny update if required fields become empty", func() {
			wh := &unitAdmission{}
			oldUnit := &upmv1alpha2.Unit{ObjectMeta: metav1.ObjectMeta{Name: "u", Namespace: "default"}}
			oldUnit.Spec.ConfigTemplateName = "ct"
			oldUnit.Spec.ConfigValueName = "cv"

			newUnit := oldUnit.DeepCopy()
			newUnit.Spec.ConfigValueName = ""

			_, err := wh.ValidateUpdate(context.Background(), oldUnit, newUnit)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("spec.configValueName is required"))
		})
	})

})
