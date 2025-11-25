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
	// ErrInvalidOpenStackProvisionServerType is returned when the object is not an OpenStackProvisionServer
	ErrInvalidOpenStackProvisionServerType = errors.New("expected an OpenStackProvisionServer object")
)

// nolint:unused
// log is for logging in this package.
var openstackprovisionserverlog = logf.Log.WithName("openstackprovisionserver-resource")

// SetupOpenStackProvisionServerWebhookWithManager registers the webhook for OpenStackProvisionServer in the manager.
func SetupOpenStackProvisionServerWebhookWithManager(mgr ctrl.Manager) error {
	// Set up webhookClient for API webhook functions
	baremetalv1beta1.SetupWebhookClient(mgr.GetClient())

	return ctrl.NewWebhookManagedBy(mgr).For(&baremetalv1beta1.OpenStackProvisionServer{}).
		WithValidator(&OpenStackProvisionServerCustomValidator{}).
		WithDefaulter(&OpenStackProvisionServerCustomDefaulter{}).
		Complete()
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// +kubebuilder:webhook:path=/mutate-baremetal-openstack-org-v1beta1-openstackprovisionserver,mutating=true,failurePolicy=fail,sideEffects=None,groups=baremetal.openstack.org,resources=openstackprovisionservers,verbs=create;update,versions=v1beta1,name=mopenstackprovisionserver-v1beta1.kb.io,admissionReviewVersions=v1

// OpenStackProvisionServerCustomDefaulter struct is responsible for setting default values on the custom resource of the
// Kind OpenStackProvisionServer when those are created or updated.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as it is used only for temporary operations and does not need to be deeply copied.
type OpenStackProvisionServerCustomDefaulter struct {
	// TODO(user): Add more fields as needed for defaulting
}

var _ webhook.CustomDefaulter = &OpenStackProvisionServerCustomDefaulter{}

// Default implements webhook.CustomDefaulter so a webhook will be registered for the Kind OpenStackProvisionServer.
func (d *OpenStackProvisionServerCustomDefaulter) Default(_ context.Context, obj runtime.Object) error {
	openstackprovisionserver, ok := obj.(*baremetalv1beta1.OpenStackProvisionServer)

	if !ok {
		return fmt.Errorf("%w but got %T", ErrInvalidOpenStackProvisionServerType, obj)
	}
	openstackprovisionserverlog.Info("Defaulting for OpenStackProvisionServer", "name", openstackprovisionserver.GetName())

	// Call the defaulting function from api/v1beta1
	openstackprovisionserver.Default()

	return nil
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// NOTE: The 'path' attribute must follow a specific pattern and should not be modified directly here.
// Modifying the path for an invalid path can cause API server errors; failing to locate the webhook.
// +kubebuilder:webhook:path=/validate-baremetal-openstack-org-v1beta1-openstackprovisionserver,mutating=false,failurePolicy=fail,sideEffects=None,groups=baremetal.openstack.org,resources=openstackprovisionservers,verbs=create;update,versions=v1beta1,name=vopenstackprovisionserver-v1beta1.kb.io,admissionReviewVersions=v1

// OpenStackProvisionServerCustomValidator struct is responsible for validating the OpenStackProvisionServer resource
// when it is created, updated, or deleted.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as this struct is used only for temporary operations and does not need to be deeply copied.
type OpenStackProvisionServerCustomValidator struct {
	// TODO(user): Add more fields as needed for validation
}

var _ webhook.CustomValidator = &OpenStackProvisionServerCustomValidator{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type OpenStackProvisionServer.
func (v *OpenStackProvisionServerCustomValidator) ValidateCreate(_ context.Context, obj runtime.Object) (admission.Warnings, error) {
	openstackprovisionserver, ok := obj.(*baremetalv1beta1.OpenStackProvisionServer)
	if !ok {
		return nil, fmt.Errorf("%w but got %T", ErrInvalidOpenStackProvisionServerType, obj)
	}
	openstackprovisionserverlog.Info("Validation for OpenStackProvisionServer upon creation", "name", openstackprovisionserver.GetName())

	// Call the validation function from api/v1beta1
	return openstackprovisionserver.ValidateCreate()
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type OpenStackProvisionServer.
func (v *OpenStackProvisionServerCustomValidator) ValidateUpdate(_ context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	openstackprovisionserver, ok := newObj.(*baremetalv1beta1.OpenStackProvisionServer)
	if !ok {
		return nil, fmt.Errorf("%w but got %T", ErrInvalidOpenStackProvisionServerType, newObj)
	}
	openstackprovisionserverlog.Info("Validation for OpenStackProvisionServer upon update", "name", openstackprovisionserver.GetName())

	// Call the validation function from api/v1beta1
	return openstackprovisionserver.ValidateUpdate(oldObj)
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type OpenStackProvisionServer.
func (v *OpenStackProvisionServerCustomValidator) ValidateDelete(_ context.Context, obj runtime.Object) (admission.Warnings, error) {
	openstackprovisionserver, ok := obj.(*baremetalv1beta1.OpenStackProvisionServer)
	if !ok {
		return nil, fmt.Errorf("%w but got %T", ErrInvalidOpenStackProvisionServerType, obj)
	}
	openstackprovisionserverlog.Info("Validation for OpenStackProvisionServer upon deletion", "name", openstackprovisionserver.GetName())

	// Call the validation function from api/v1beta1
	return openstackprovisionserver.ValidateDelete()
}
