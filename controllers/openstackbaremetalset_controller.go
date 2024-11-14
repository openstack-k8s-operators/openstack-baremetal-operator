/*
Copyright 2023.

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

package controllers

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	k8s_errors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/go-logr/logr"
	metal3v1 "github.com/metal3-io/baremetal-operator/apis/metal3.io/v1alpha1"
	"github.com/openstack-k8s-operators/lib-common/modules/common/condition"
	"github.com/openstack-k8s-operators/lib-common/modules/common/env"
	"github.com/openstack-k8s-operators/lib-common/modules/common/helper"
	"github.com/openstack-k8s-operators/lib-common/modules/common/labels"
	oko_secret "github.com/openstack-k8s-operators/lib-common/modules/common/secret"
	"github.com/openstack-k8s-operators/lib-common/modules/common/util"
	baremetalv1 "github.com/openstack-k8s-operators/openstack-baremetal-operator/api/v1beta1"
	"github.com/openstack-k8s-operators/openstack-baremetal-operator/pkg/openstackbaremetalset"
)

// OpenStackBaremetalSetReconciler reconciles a OpenStackBaremetalSet object
type OpenStackBaremetalSetReconciler struct {
	client.Client
	Kclient kubernetes.Interface
	Log     logr.Logger
	Scheme  *runtime.Scheme
}

// +kubebuilder:rbac:groups=baremetal.openstack.org,resources=openstackbaremetalsets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=baremetal.openstack.org,resources=openstackbaremetalsets/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=baremetal.openstack.org,resources=openstackbaremetalsets/finalizers,verbs=update;patch
// +kubebuilder:rbac:groups=baremetal.openstack.org,resources=openstackprovisionservers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=baremetal.openstack.org,resources=openstackprovisionservers/finalizers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=baremetal.openstack.org,resources=openstackprovisionservers/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=metal3.io,resources=baremetalhosts,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=metal3.io,resources=baremetalhosts/status,verbs=get
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=create;delete;get;list;patch;update;watch
// +kubebuilder:rbac:groups=core,resources=secrets/finalizers,verbs=create;delete;get;list;patch;update;watch

// Reconcile -
func (r *OpenStackBaremetalSetReconciler) Reconcile(ctx context.Context, req ctrl.Request) (result ctrl.Result, _err error) {
	_ = log.FromContext(ctx)

	// Fetch the OpenStackBaremetalSet instance
	instance := &baremetalv1.OpenStackBaremetalSet{}
	err := r.Client.Get(ctx, req.NamespacedName, instance)
	if err != nil {
		if k8s_errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected.
			// For additional cleanup logic use finalizers. Return and don't requeue.
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return ctrl.Result{}, err
	}

	helper, err := helper.NewHelper(
		instance,
		r.Client,
		r.Kclient,
		r.Scheme,
		r.Log,
	)
	if err != nil {
		return ctrl.Result{}, err
	}

	// initialize status if Conditions is nil, but do not reset if it already
	// exists
	isNewInstance := instance.Status.Conditions == nil
	if isNewInstance {
		instance.Status.Conditions = condition.Conditions{}
	}

	// Save a copy of the condtions so that we can restore the LastTransitionTime
	// when a condition's state doesn't change.
	savedConditions := instance.Status.Conditions.DeepCopy()

	// Always patch the instance status when exiting this function so we can
	// persist any changes.
	defer func() {
		condition.RestoreLastTransitionTimes(
			&instance.Status.Conditions, savedConditions)
		if instance.Status.Conditions.IsUnknown(condition.ReadyCondition) {
			instance.Status.Conditions.Set(
				instance.Status.Conditions.Mirror(condition.ReadyCondition))
		}
		err := helper.PatchInstance(ctx, instance)
		if err != nil {
			_err = err
			return
		}
	}()

	//
	// initialize status
	//
	// initialize conditions used later as Status=Unknown
	cl := condition.CreateList(
		condition.UnknownCondition(condition.ReadyCondition, condition.InitReason, condition.ReadyInitMessage),
		condition.UnknownCondition(condition.InputReadyCondition, condition.InitReason, condition.InputReadyInitMessage),
		condition.UnknownCondition(baremetalv1.OpenStackBaremetalSetProvServerReadyCondition, condition.InitReason, baremetalv1.OpenStackBaremetalSetProvServerReadyInitMessage),
		condition.UnknownCondition(baremetalv1.OpenStackBaremetalSetBmhProvisioningReadyCondition, condition.InitReason, baremetalv1.OpenStackBaremetalSetBmhProvisioningReadyInitMessage),
	)

	instance.Status.Conditions.Init(&cl)
	instance.Status.ObservedGeneration = instance.Generation

	// If we're not deleting this and the service object doesn't have our finalizer, add it.
	if instance.DeletionTimestamp.IsZero() && controllerutil.AddFinalizer(instance, helper.GetFinalizer()) || isNewInstance {
		return ctrl.Result{}, nil
	}

	if instance.Status.Hash == nil {
		instance.Status.Hash = map[string]string{}
	}
	if instance.Status.BaremetalHosts == nil {
		instance.Status.BaremetalHosts = map[string]baremetalv1.HostStatus{}
	}

	// Handle service delete
	if !instance.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, instance, helper)
	}

	// Handle non-deleted servers
	return r.reconcileNormal(ctx, instance, helper)
}

// SetupWithManager sets up the controller with the Manager.
func (r *OpenStackBaremetalSetReconciler) SetupWithManager(mgr ctrl.Manager) error {
	groupLabel := labels.GetGroupLabel(baremetalv1.ServiceName)

	openshiftMachineAPIBareMetalHostsFn := handler.EnqueueRequestsFromMapFunc(func(_ context.Context, o client.Object) []reconcile.Request {
		result := []reconcile.Request{}
		label := o.GetLabels()
		// verify object has ownerUIDLabelSelector
		if uid, ok := label[labels.GetOwnerUIDLabelSelector(groupLabel)]; ok {
			r.Log.Info(fmt.Sprintf("BareMetalHost object %s marked with OpenStackBaremetalSet owner ref: %s", o.GetName(), uid))

			// return namespace and Name of CR
			name := client.ObjectKey{
				Namespace: label[labels.GetOwnerNameSpaceLabelSelector(groupLabel)],
				Name:      label[labels.GetOwnerNameLabelSelector(groupLabel)],
			}
			result = append(result, reconcile.Request{NamespacedName: name})
		}
		if len(result) > 0 {
			return result
		}
		return nil
	})

	return ctrl.NewControllerManagedBy(mgr).
		For(&baremetalv1.OpenStackBaremetalSet{}).
		Owns(&baremetalv1.OpenStackProvisionServer{}).
		Watches(&metal3v1.BareMetalHost{}, openshiftMachineAPIBareMetalHostsFn).
		Complete(r)
}

func (r *OpenStackBaremetalSetReconciler) reconcileDelete(ctx context.Context, instance *baremetalv1.OpenStackBaremetalSet, helper *helper.Helper) (ctrl.Result, error) {
	r.Log.Info(fmt.Sprintf("Reconciling OpenStackBaremetalSet '%s' delete", instance.Name))

	// Clean up resources used by the operator
	// BareMetalHost resources in the namespace (don't delete, just deprovision)
	err := r.baremetalHostCleanup(ctx, helper, instance)
	if err != nil && !k8s_errors.IsNotFound(err) {
		// ignore not found errors if the object is already gone
		return ctrl.Result{}, err
	}

	controllerutil.RemoveFinalizer(instance, helper.GetFinalizer())
	r.Log.Info(fmt.Sprintf("Reconciled OpenStackBaremetalSet '%s' delete successfully", instance.Name))

	return ctrl.Result{}, nil
}

func (r *OpenStackBaremetalSetReconciler) reconcileInit(
	_ context.Context,
	instance *baremetalv1.OpenStackBaremetalSet,
	_ *helper.Helper,
) (ctrl.Result, error) {
	r.Log.Info(fmt.Sprintf("Reconciling OpenStackBaremetalSet '%s' init", instance.Name))

	r.Log.Info(fmt.Sprintf("Reconciled OpenStackBaremetalSet '%s' init successfully", instance.Name))
	return ctrl.Result{}, nil
}

func (r *OpenStackBaremetalSetReconciler) reconcileUpdate(_ context.Context, instance *baremetalv1.OpenStackBaremetalSet, _ *helper.Helper) (ctrl.Result, error) {
	r.Log.Info(fmt.Sprintf("Reconciling OpenStackBaremetalSet '%s' update", instance.Name))

	// TODO: should have minor update tasks if required
	// - delete dbsync hash from status to rerun it?

	r.Log.Info(fmt.Sprintf("Reconciled OpenStackBaremetalSet '%s' update successfully", instance.Name))
	return ctrl.Result{}, nil
}

func (r *OpenStackBaremetalSetReconciler) reconcileUpgrade(_ context.Context, instance *baremetalv1.OpenStackBaremetalSet, _ *helper.Helper) (ctrl.Result, error) {
	r.Log.Info(fmt.Sprintf("Reconciling OpenStackBaremetalSet '%s' upgrade", instance.Name))

	// TODO: should have major version upgrade tasks
	// -delete dbsync hash from status to rerun it?

	r.Log.Info(fmt.Sprintf("Reconciled OpenStackBaremetalSet '%s' upgrade successfully", instance.Name))
	return ctrl.Result{}, nil
}

func (r *OpenStackBaremetalSetReconciler) reconcileNormal(ctx context.Context, instance *baremetalv1.OpenStackBaremetalSet, helper *helper.Helper) (ctrl.Result, error) {
	r.Log.Info(fmt.Sprintf("Reconciling OpenStackBaremetalSet '%s'", instance.Name))

	l := log.FromContext(ctx)

	// ConfigMap
	configMapVars := make(map[string]env.Setter)

	//
	// check if the required deployment SSH secret is available and add hash to the vars map
	//
	sshSecret, hash, err := oko_secret.GetSecret(ctx, helper, instance.Spec.DeploymentSSHSecret, instance.Namespace)
	if err != nil {
		if k8s_errors.IsNotFound(err) {
			l.Info("Deployment SSH secret not found", "Deployment SSH secret", instance.Spec.DeploymentSSHSecret)
			instance.Status.Conditions.Set(condition.FalseCondition(
				condition.InputReadyCondition,
				condition.RequestedReason,
				condition.SeverityInfo,
				condition.InputReadyWaitingMessage))
			return ctrl.Result{RequeueAfter: time.Second * 10}, nil
		}
		instance.Status.Conditions.Set(condition.FalseCondition(
			condition.InputReadyCondition,
			condition.ErrorReason,
			condition.SeverityWarning,
			condition.InputReadyErrorMessage,
			err.Error()))
		return ctrl.Result{}, err
	}
	configMapVars[sshSecret.Name] = env.SetValue(hash)
	instance.Status.Conditions.MarkTrue(condition.InputReadyCondition, condition.InputReadyMessage)
	// run check deployment SSH secret - end

	//
	// check if a root password secret was provide and add hash to the vars map if so
	//
	var passwordSecret *corev1.Secret

	if instance.Spec.PasswordSecret != nil {
		passwordSecret, hash, err = oko_secret.GetSecret(
			ctx, helper, instance.Spec.PasswordSecret.Name,
			instance.Spec.PasswordSecret.Namespace)
		if err != nil {
			if k8s_errors.IsNotFound(err) {
				l.Info("Root password secret not found", "Root password secret", instance.Spec.PasswordSecret)
				instance.Status.Conditions.Set(condition.FalseCondition(
					condition.InputReadyCondition,
					condition.RequestedReason,
					condition.SeverityInfo,
					condition.InputReadyWaitingMessage))
				return ctrl.Result{RequeueAfter: time.Second * 10}, nil
			}
			instance.Status.Conditions.Set(condition.FalseCondition(
				condition.InputReadyCondition,
				condition.ErrorReason,
				condition.SeverityWarning,
				condition.InputReadyErrorMessage,
				err.Error()))
			return ctrl.Result{}, err
		}
		configMapVars[passwordSecret.Name] = env.SetValue(hash)
	}
	// run check OpenStack secret - end

	//
	// TODO check when/if Init, Update, or Upgrade should/could be skipped
	//

	// Handle service init
	ctrlResult, err := r.reconcileInit(ctx, instance, helper)
	if err != nil {
		return ctrlResult, err
	} else if (ctrlResult != ctrl.Result{}) {
		return ctrlResult, nil
	}

	// Handle service update
	ctrlResult, err = r.reconcileUpdate(ctx, instance, helper)
	if err != nil {
		return ctrlResult, err
	} else if (ctrlResult != ctrl.Result{}) {
		return ctrlResult, nil
	}

	// Handle service upgrade
	ctrlResult, err = r.reconcileUpgrade(ctx, instance, helper)
	if err != nil {
		return ctrlResult, err
	} else if (ctrlResult != ctrl.Result{}) {
		return ctrlResult, nil
	}

	//
	// normal reconcile tasks
	//

	//
	// either find the provided provision server or create a new one
	//
	provisionServer := &baremetalv1.OpenStackProvisionServer{}

	// TODO: webook should validate that either ProvisionServerName or OSImage is set in the instance spec
	if instance.Spec.ProvisionServerName == "" {
		provisionServer, err = r.provisionServerCreateOrUpdate(ctx, helper, instance)
	} else {
		// Clean-up any existing OsProvServer that we may have dynamically created for this OsBaremetalSet,
		// since we are instead relying on a pre-existing OsProvServer...but only do this if the user hasn't,
		// for some reason:
		// 1. Set "provisionServerName" to the previous auto-generated OpenStackProvisionServer
		// 2. Or happened to create an OsProvServer beforehand with a name that matches the one that would
		//    be auto-generated by this controller
		if instance.Spec.ProvisionServerName != fmt.Sprintf("%s-provisionserver", instance.Name) {
			err = r.provisionServerDelete(ctx, instance)
		}

		// If the above call hit an error, the error will drop through and be handled below
		if err == nil {
			// Get the pre-existing OsProvServer that we intend to use
			err = r.Client.Get(ctx, types.NamespacedName{Name: instance.Spec.ProvisionServerName, Namespace: instance.Namespace}, provisionServer)
		}
	}

	if err != nil {
		if k8s_errors.IsNotFound(err) {
			l.Info("OpenStackProvisionServer not found", "OpenStackProvisionServer", instance.Spec.ProvisionServerName)
			instance.Status.Conditions.Set(condition.FalseCondition(
				baremetalv1.OpenStackBaremetalSetProvServerReadyCondition,
				condition.RequestedReason,
				condition.SeverityInfo,
				baremetalv1.OpenStackBaremetalSetProvServerReadyWaitingMessage))
			return ctrl.Result{RequeueAfter: time.Second * 10}, nil
		}
		instance.Status.Conditions.Set(condition.FalseCondition(
			baremetalv1.OpenStackBaremetalSetProvServerReadyCondition,
			condition.ErrorReason,
			condition.SeverityWarning,
			baremetalv1.OpenStackBaremetalSetProvServerReadyErrorMessage,
			err.Error()))
		return ctrl.Result{}, err
	}

	if provisionServer.Status.LocalImageURL == "" {
		instance.Status.Conditions.Set(condition.FalseCondition(
			baremetalv1.OpenStackBaremetalSetProvServerReadyCondition,
			condition.RequestedReason,
			condition.SeverityInfo,
			baremetalv1.OpenStackBaremetalSetProvServerReadyRunningMessage))
		l.Info("OpenStackProvisionServer LocalImageURL not yet available", "OpenStackProvisionServer", provisionServer.Name)
		return ctrl.Result{RequeueAfter: time.Second * 30}, nil
	}

	if provisionServer.Status.LocalImageChecksumURL == "" {
		instance.Status.Conditions.Set(condition.FalseCondition(
			baremetalv1.OpenStackBaremetalSetProvServerReadyCondition,
			condition.RequestedReason,
			condition.SeverityInfo,
			baremetalv1.OpenStackBaremetalSetProvServerReadyRunningMessage))
		l.Info("OpenStackProvisionServer LocalImageChecksumURL not yet available", "OpenStackProvisionServer", provisionServer.Name)
		return ctrl.Result{RequeueAfter: time.Duration(5) * time.Second}, nil
	}

	instance.Status.Conditions.MarkTrue(baremetalv1.OpenStackBaremetalSetProvServerReadyCondition, baremetalv1.OpenStackBaremetalSetProvServerReadyMessage)
	// handle provision server - end

	// Check if any BMHs that this CR is using (i.e. that is present as a bmhRef in
	// the CR's Status.BaremetalHosts map) were inappropriately (manually) deleted.
	// If so, we cannot proceed further as we will risk placing the CR into an
	// inconsistent state and/or introducing unbounded reconciliation thrashing.
	//
	err = baremetalv1.VerifyBaremetalStatusBmhRefs(ctx, helper.GetClient(), instance)

	if err != nil {
		instance.Status.Conditions.Set(condition.FalseCondition(
			baremetalv1.OpenStackBaremetalSetBmhProvisioningReadyCondition,
			condition.ErrorReason,
			condition.SeverityWarning,
			baremetalv1.OpenStackBaremetalSetBmhProvisioningReadyErrorMessage,
			err.Error()))
		return ctrl.Result{}, err
	}
	// check for erroneous BMH deletion - end

	bmhLabels := labels.GetLabels(instance, labels.GetGroupLabel(baremetalv1.ServiceName), map[string]string{})

	//
	// handle BMH removal from BMSet
	//
	err = r.deleteBmh(
		ctx,
		helper,
		instance,
		bmhLabels,
	)
	if err != nil {
		instance.Status.Conditions.Set(condition.FalseCondition(
			baremetalv1.OpenStackBaremetalSetBmhProvisioningReadyCondition,
			condition.ErrorReason,
			condition.SeverityWarning,
			baremetalv1.OpenStackBaremetalSetBmhProvisioningReadyErrorMessage,
			err.Error()))
		return ctrl.Result{}, err
	}
	// handle BMH removal - end

	//
	// provision requested BMH replicas
	//
	if err := r.ensureBaremetalHosts(
		ctx,
		helper,
		instance,
		provisionServer,
		sshSecret,
		passwordSecret,
		bmhLabels,
		&configMapVars,
	); err != nil {
		instance.Status.Conditions.Set(condition.FalseCondition(
			baremetalv1.OpenStackBaremetalSetBmhProvisioningReadyCondition,
			condition.ErrorReason,
			condition.SeverityWarning,
			baremetalv1.OpenStackBaremetalSetBmhProvisioningReadyErrorMessage,
			err.Error()))
		return ctrl.Result{}, err
	}

	// Now calculate overall provisioning status for all requested BaremetalHosts
	for _, bmhStatus := range instance.Status.BaremetalHosts {
		if bmhStatus.ProvisioningState != baremetalv1.ProvisioningState(metal3v1.StateProvisioned) {
			instance.Status.Conditions.Set(condition.FalseCondition(
				baremetalv1.OpenStackBaremetalSetBmhProvisioningReadyCondition,
				condition.RequestedReason,
				condition.SeverityInfo,
				baremetalv1.OpenStackBaremetalSetBmhProvisioningReadyRunningMessage))
			return ctrl.Result{RequeueAfter: time.Second * 20}, nil
		}
	}
	instance.Status.Conditions.MarkTrue(baremetalv1.OpenStackBaremetalSetBmhProvisioningReadyCondition, baremetalv1.OpenStackBaremetalSetBmhProvisioningReadyMessage)
	// provision BMHs - end

	// We reached the end of the Reconcile, update the Ready condition based on
	// the sub conditions
	if instance.Status.Conditions.AllSubConditionIsTrue() {
		instance.Status.Conditions.MarkTrue(
			condition.ReadyCondition, condition.ReadyMessage)
	}
	r.Log.Info(fmt.Sprintf("Reconciled OpenStackBaremetalSet '%s' successfully", instance.Name))
	return ctrl.Result{}, nil
}

func (r *OpenStackBaremetalSetReconciler) provisionServerCreateOrUpdate(
	ctx context.Context,
	helper *helper.Helper,
	instance *baremetalv1.OpenStackBaremetalSet,
) (*baremetalv1.OpenStackProvisionServer, error) {
	l := log.FromContext(ctx)

	// Next deploy the provisioning image (Apache) server
	provisionServer := &baremetalv1.OpenStackProvisionServer{
		ObjectMeta: metav1.ObjectMeta{
			Name:      instance.ObjectMeta.Name + "-provisionserver",
			Namespace: instance.ObjectMeta.Namespace,
		},
	}

	op, err := controllerutil.CreateOrPatch(ctx, helper.GetClient(), provisionServer, func() error {
		// Leave the prov server's existing port as-is if this is an update, otherwise pick a new one
		// based on what is available
		if provisionServer.Spec.Port == 0 {
			err := baremetalv1.AssignProvisionServerPort(
				ctx,
				helper.GetClient(),
				provisionServer,
				baremetalv1.DefaultProvisionPort,
			)
			if err != nil {
				return err
			}
		}
		provisionServer.Spec.OSImage = instance.Spec.OSImage
		provisionServer.Spec.OSContainerImageURL = instance.Spec.OSContainerImageURL
		provisionServer.Spec.ApacheImageURL = instance.Spec.ApacheImageURL
		provisionServer.Spec.AgentImageURL = instance.Spec.AgentImageURL
		provisionServer.Spec.Interface = instance.Spec.ProvisioningInterface

		err := controllerutil.SetControllerReference(instance, provisionServer, helper.GetScheme())
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return provisionServer, err
	}
	if op != controllerutil.OperationResultNone {
		l.Info("OpenStackProvisionServer successfully reconciled", "OpenStackProvisionServer", provisionServer.Name, "operation", string(op))
	}

	return provisionServer, nil
}

func (r *OpenStackBaremetalSetReconciler) provisionServerDelete(
	ctx context.Context,
	instance *baremetalv1.OpenStackBaremetalSet,
) error {
	l := log.FromContext(ctx)

	provisionServer := &baremetalv1.OpenStackProvisionServer{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-provisionserver", instance.Name),
			Namespace: instance.ObjectMeta.Namespace,
		},
	}

	err := r.Client.Delete(ctx, provisionServer)

	if err == nil {
		l.Info("OpenStackProvisionServer successfully deleted", "OpenStackProvisionServer", provisionServer.Name)
	} else if k8s_errors.IsNotFound(err) {
		return nil
	}

	return err
}

// deleteBmh - Deprovision BaremetalHost resources based on spec's BaremetalHost map contrasted with status' BaremetalHost map
func (r *OpenStackBaremetalSetReconciler) deleteBmh(
	ctx context.Context,
	helper *helper.Helper,
	instance *baremetalv1.OpenStackBaremetalSet,
	labels map[string]string,
) error {
	// Get BaremetalHosts that this instance is currently using
	existingBaremetalHosts, err := baremetalv1.GetBaremetalHosts(ctx, helper.GetClient(), instance.Spec.BmhNamespace, labels)
	if err != nil {
		return err
	}

	// Figure out what BaremetalHost de-allocations we need, if any
	instanceBmhOwnershipLabelKey := fmt.Sprintf("%s%s", instance.Name, openstackbaremetalset.HostnameLabelSelectorSuffix)
	hostNamesToDeprovision := []string{}

	for _, existingBmh := range existingBaremetalHosts.Items {
		// Does the instance.Spec.BaremetalHosts map still contain this BMH?
		found := false

		for hostName := range instance.Spec.BaremetalHosts {
			if existingBmh.Labels[instanceBmhOwnershipLabelKey] == hostName {
				found = true
				break
			}
		}

		if !found {
			hostNamesToDeprovision = append(hostNamesToDeprovision, existingBmh.Labels[instanceBmhOwnershipLabelKey])
		}
	}

	// Deallocate all BaremetalHosts we no longer require
	for _, hostName := range hostNamesToDeprovision {
		bmhStatus := instance.Status.BaremetalHosts[hostName]

		err = openstackbaremetalset.BaremetalHostDeprovision(
			ctx,
			helper,
			instance,
			bmhStatus,
		)

		if err != nil {
			return err
		}
	}

	return nil
}

func (r *OpenStackBaremetalSetReconciler) buildExistingHostBMHMap(instance *baremetalv1.OpenStackBaremetalSet,
	existingBMHs *metal3v1.BareMetalHostList) map[string]metal3v1.BareMetalHost {
	existingHostBMHMap := make(map[string]metal3v1.BareMetalHost)
	instanceBmhOwnershipLabelKey := fmt.Sprintf("%s%s", instance.Name, openstackbaremetalset.HostnameLabelSelectorSuffix)
	for _, bmh := range existingBMHs.Items {
		hostName, ok := bmh.Labels[instanceBmhOwnershipLabelKey]
		if ok {
			existingHostBMHMap[hostName] = bmh
		}
	}
	return existingHostBMHMap
}

// Provision BaremetalHost resources based on replica count
func (r *OpenStackBaremetalSetReconciler) ensureBaremetalHosts(
	ctx context.Context,
	helper *helper.Helper,
	instance *baremetalv1.OpenStackBaremetalSet,
	provisionServer *baremetalv1.OpenStackProvisionServer,
	sshSecret *corev1.Secret,
	passwordSecret *corev1.Secret,
	bmhLabels map[string]string,
	envVars *map[string]env.Setter,
) error {

	// Get all BaremetalHosts (and, optionally, only those that match instance.Spec.BmhLabelSelector if there is one)
	baremetalHostsList, err := baremetalv1.GetBaremetalHosts(
		ctx,
		helper.GetClient(),
		instance.Spec.BmhNamespace,
		instance.Spec.BmhLabelSelector,
	)
	if err != nil {
		return err
	}

	// Get all existing BaremetalHosts of this CR
	existingBaremetalHosts, err := baremetalv1.GetBaremetalHosts(ctx, helper.GetClient(), instance.Spec.BmhNamespace, bmhLabels)
	if err != nil {
		return err
	}

	// Verify that we have enough hosts with the right hardware reqs available for scaling-up
	selectedHostBMHMap, err := baremetalv1.VerifyBaremetalSetScaleUp(log.FromContext(ctx), instance, baremetalHostsList, existingBaremetalHosts)
	if err != nil {
		return err
	}

	existingHostBMHMap := r.buildExistingHostBMHMap(instance, existingBaremetalHosts)
	selectedHostBMHMap = util.MergeMaps(selectedHostBMHMap, existingHostBMHMap)

	for desiredHostName, bmh := range selectedHostBMHMap {
		err := openstackbaremetalset.BaremetalHostProvision(
			ctx,
			helper,
			instance,
			bmh.Name,
			desiredHostName,
			instance.Spec.BaremetalHosts[desiredHostName].CtlPlaneIP, // ctlPlaneIP
			provisionServer,
			sshSecret,
			passwordSecret,
			envVars,
		)
		if err != nil {
			return err
		}
	}

	return nil
}

// Deprovision all associated BaremetalHosts for this OpenStackBaremetalSet via Metal3
func (r *OpenStackBaremetalSetReconciler) baremetalHostCleanup(
	ctx context.Context,
	helper *helper.Helper,
	instance *baremetalv1.OpenStackBaremetalSet,
) error {
	if instance.Status.BaremetalHosts != nil {
		for _, bmh := range instance.Status.BaremetalHosts {
			if err := openstackbaremetalset.BaremetalHostDeprovision(ctx, helper, instance, bmh); err != nil {
				return err
			}
		}
	}

	return nil
}
