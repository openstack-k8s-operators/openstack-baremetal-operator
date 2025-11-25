/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package v1beta1 contains webhook implementations for the v1beta1 API version.
package v1beta1

import (
	"context"
	"errors"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	baremetalv1beta1 "github.com/openstack-k8s-operators/openstack-baremetal-operator/api/v1beta1"
)

var (
	// ErrInvalidOpenStackBaremetalSetType is returned when the object is not an OpenStackBaremetalSet
	ErrInvalidOpenStackBaremetalSetType = errors.New("expected an OpenStackBaremetalSet object")
)

// nolint:unused
// log is for logging in this package.
var openstackbaremetalsetlog = logf.Log.WithName("openstackbaremetalset-resource")

// SetupOpenStackBaremetalSetWebhookWithManager registers the webhook for OpenStackBaremetalSet in the manager.
func SetupOpenStackBaremetalSetWebhookWithManager(mgr ctrl.Manager) error {
	// Set up webhookClient for API webhook functions
	baremetalv1beta1.SetupWebhookClient(mgr.GetClient())

	return ctrl.NewWebhookManagedBy(mgr).For(&baremetalv1beta1.OpenStackBaremetalSet{}).
		WithValidator(&OpenStackBaremetalSetCustomValidator{}).
		WithDefaulter(&OpenStackBaremetalSetCustomDefaulter{}).
		Complete()
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// +kubebuilder:webhook:path=/mutate-baremetal-openstack-org-v1beta1-openstackbaremetalset,mutating=true,failurePolicy=fail,sideEffects=None,groups=baremetal.openstack.org,resources=openstackbaremetalsets,verbs=create;update,versions=v1beta1,name=mopenstackbaremetalset-v1beta1.kb.io,admissionReviewVersions=v1

// OpenStackBaremetalSetCustomDefaulter struct is responsible for setting default values on the custom resource of the
// Kind OpenStackBaremetalSet when those are created or updated.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as it is used only for temporary operations and does not need to be deeply copied.
type OpenStackBaremetalSetCustomDefaulter struct {
	// TODO(user): Add more fields as needed for defaulting
}

var _ webhook.CustomDefaulter = &OpenStackBaremetalSetCustomDefaulter{}

// Default implements webhook.CustomDefaulter so a webhook will be registered for the Kind OpenStackBaremetalSet.
func (d *OpenStackBaremetalSetCustomDefaulter) Default(_ context.Context, obj runtime.Object) error {
	openstackbaremetalset, ok := obj.(*baremetalv1beta1.OpenStackBaremetalSet)

	if !ok {
		return fmt.Errorf("%w but got %T", ErrInvalidOpenStackBaremetalSetType, obj)
	}
	openstackbaremetalsetlog.Info("Defaulting for OpenStackBaremetalSet", "name", openstackbaremetalset.GetName())

	// No defaulting logic needed as of yet for OpenStackBaremetalSet

	return nil
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// NOTE: The 'path' attribute must follow a specific pattern and should not be modified directly here.
// Modifying the path for an invalid path can cause API server errors; failing to locate the webhook.
// +kubebuilder:webhook:path=/validate-baremetal-openstack-org-v1beta1-openstackbaremetalset,mutating=false,failurePolicy=fail,sideEffects=None,groups=baremetal.openstack.org,resources=openstackbaremetalsets,verbs=create;update,versions=v1beta1,name=vopenstackbaremetalset-v1beta1.kb.io,admissionReviewVersions=v1

// OpenStackBaremetalSetCustomValidator struct is responsible for validating the OpenStackBaremetalSet resource
// when it is created, updated, or deleted.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as this struct is used only for temporary operations and does not need to be deeply copied.
type OpenStackBaremetalSetCustomValidator struct {
	// TODO(user): Add more fields as needed for validation
}

var _ webhook.CustomValidator = &OpenStackBaremetalSetCustomValidator{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type OpenStackBaremetalSet.
func (v *OpenStackBaremetalSetCustomValidator) ValidateCreate(_ context.Context, obj runtime.Object) (admission.Warnings, error) {
	openstackbaremetalset, ok := obj.(*baremetalv1beta1.OpenStackBaremetalSet)
	if !ok {
		return nil, fmt.Errorf("%w but got %T", ErrInvalidOpenStackBaremetalSetType, obj)
	}
	openstackbaremetalsetlog.Info("Validation for OpenStackBaremetalSet upon creation", "name", openstackbaremetalset.GetName())

	// Call the validation function from api/v1beta1
	return openstackbaremetalset.ValidateCreate()
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type OpenStackBaremetalSet.
func (v *OpenStackBaremetalSetCustomValidator) ValidateUpdate(_ context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	openstackbaremetalset, ok := newObj.(*baremetalv1beta1.OpenStackBaremetalSet)
	if !ok {
		return nil, fmt.Errorf("%w but got %T", ErrInvalidOpenStackBaremetalSetType, newObj)
	}
	openstackbaremetalsetlog.Info("Validation for OpenStackBaremetalSet upon update", "name", openstackbaremetalset.GetName())

	// Call the validation function from api/v1beta1
	return openstackbaremetalset.ValidateUpdate(oldObj)
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type OpenStackBaremetalSet.
func (v *OpenStackBaremetalSetCustomValidator) ValidateDelete(_ context.Context, obj runtime.Object) (admission.Warnings, error) {
	openstackbaremetalset, ok := obj.(*baremetalv1beta1.OpenStackBaremetalSet)
	if !ok {
		return nil, fmt.Errorf("%w but got %T", ErrInvalidOpenStackBaremetalSetType, obj)
	}
	openstackbaremetalsetlog.Info("Validation for OpenStackBaremetalSet upon deletion", "name", openstackbaremetalset.GetName())

	// Call the validation function from api/v1beta1
	return openstackbaremetalset.ValidateDelete()
}
