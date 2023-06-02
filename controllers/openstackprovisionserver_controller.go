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
	"strconv"
	"strings"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	k8s_errors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/go-logr/logr"
	"github.com/openstack-k8s-operators/lib-common/modules/common"
	"github.com/openstack-k8s-operators/lib-common/modules/common/condition"
	"github.com/openstack-k8s-operators/lib-common/modules/common/configmap"
	"github.com/openstack-k8s-operators/lib-common/modules/common/deployment"
	"github.com/openstack-k8s-operators/lib-common/modules/common/env"
	"github.com/openstack-k8s-operators/lib-common/modules/common/helper"
	"github.com/openstack-k8s-operators/lib-common/modules/common/labels"
	common_rbac "github.com/openstack-k8s-operators/lib-common/modules/common/rbac"
	"github.com/openstack-k8s-operators/lib-common/modules/common/util"
	baremetalv1 "github.com/openstack-k8s-operators/openstack-baremetal-operator/api/v1beta1"
	openstackprovisionserver "github.com/openstack-k8s-operators/openstack-baremetal-operator/pkg/openstackprovisionserver"
)

var (
	provisioningsGVR = schema.GroupVersionResource{
		Group:    "metal3.io",
		Version:  "v1alpha1",
		Resource: "provisionings",
	}
)

// OpenStackProvisionServerReconciler reconciles a OpenStackProvisionServer object
type OpenStackProvisionServerReconciler struct {
	client.Client
	Kclient kubernetes.Interface
	Log     logr.Logger
	Scheme  *runtime.Scheme
}

// service account, role, rolebinding
// +kubebuilder:rbac:groups="",resources=serviceaccounts,verbs=get;list;watch;create;update
// +kubebuilder:rbac:groups="rbac.authorization.k8s.io",resources=roles,verbs=get;list;watch;create;update
// +kubebuilder:rbac:groups="rbac.authorization.k8s.io",resources=rolebindings,verbs=get;list;watch;create;update
// service account permissions that are needed to grant permission to the above
// +kubebuilder:rbac:groups="security.openshift.io",resourceNames=privileged,resources=securitycontextconstraints,verbs=use
// +kubebuilder:rbac:groups="",resources=pods,verbs=create;delete;get;list;patch;update;watch

// +kubebuilder:rbac:groups=baremetal.openstack.org,resources=openstackprovisionservers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=baremetal.openstack.org,resources=openstackprovisionservers/status,verbs=get;list;update;patch
// +kubebuilder:rbac:groups=baremetal.openstack.org,resources=openstackprovisionservers/finalizers,verbs=update
// +kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;create;update;delete;watch;
// +kubebuilder:rbac:groups=core,resources=configmaps/finalizers,verbs=get;list;create;update;delete;watch;
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;create;update;delete;patch;watch;
// +kubebuilder:rbac:groups=core,resources=volumes,verbs=get;list;create;update;delete;watch;
// +kubebuilder:rbac:groups=core,resources=nodes,verbs=get;list;update;watch;
// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;update;watch;
// +kubebuilder:rbac:groups=metal3.io,resources=provisionings,verbs=get;list;watch
// +kubebuilder:rbac:groups=metal3.io,resources=baremetalhosts,verbs=get;list;update;patch;watch

// Reconcile -
func (r *OpenStackProvisionServerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (result ctrl.Result, _err error) {
	_ = log.FromContext(ctx)

	// Fetch the OpenStackProvisionServer instance
	instance := &baremetalv1.OpenStackProvisionServer{}
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

	// Always patch the instance status when exiting this function so we can persist any changes.
	defer func() {
		// update the overall status condition if service is ready
		if instance.IsReady() {
			instance.Status.Conditions.MarkTrue(condition.ReadyCondition, condition.ReadyMessage)
		}

		err := helper.PatchInstance(ctx, instance)
		if err != nil {
			_err = err
			return
		}
	}()

	// If we're not deleting this and the service object doesn't have our finalizer, add it.
	if instance.DeletionTimestamp.IsZero() && controllerutil.AddFinalizer(instance, helper.GetFinalizer()) {
		return ctrl.Result{}, nil
	}

	//
	// initialize status
	//
	if instance.Status.Conditions == nil {
		instance.Status.Conditions = condition.Conditions{}
		// initialize conditions used later as Status=Unknown
		cl := condition.CreateList(
			condition.UnknownCondition(
				condition.DeploymentReadyCondition,
				condition.InitReason,
				condition.DeploymentReadyInitMessage,
			),
			condition.UnknownCondition(
				condition.ServiceConfigReadyCondition,
				condition.InitReason,
				condition.ServiceConfigReadyInitMessage,
			),
			condition.UnknownCondition(
				baremetalv1.OpenStackProvisionServerProvIntfReadyCondition,
				condition.InitReason,
				baremetalv1.OpenStackProvisionServerProvIntfReadyInitMessage,
			),
			condition.UnknownCondition(
				baremetalv1.OpenStackProvisionServerLocalImageURLReadyCondition,
				condition.InitReason,
				baremetalv1.OpenStackProvisionServerLocalImageURLReadyInitMessage,
			),

			// service account, role, rolebinding conditions
			condition.UnknownCondition(
				condition.ServiceAccountReadyCondition,
				condition.InitReason,
				condition.ServiceAccountReadyInitMessage,
			),
			condition.UnknownCondition(
				condition.RoleReadyCondition,
				condition.InitReason,
				condition.RoleReadyInitMessage,
			),
			condition.UnknownCondition(
				condition.RoleBindingReadyCondition,
				condition.InitReason,
				condition.RoleBindingReadyInitMessage,
			),
		)
		instance.Status.Conditions.Init(&cl)

		// Register overall status immediately to have an early feedback e.g. in the cli
		return ctrl.Result{}, nil
	}
	if instance.Status.Hash == nil {
		instance.Status.Hash = map[string]string{}
	}

	// Handle service delete
	if !instance.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, instance, helper)
	}

	// Service account, role, binding
	rbacRules := []rbacv1.PolicyRule{
		{
			APIGroups:     []string{"security.openshift.io"},
			ResourceNames: []string{"privileged"},
			Resources:     []string{"securitycontextconstraints"},
			Verbs:         []string{"use"},
		},
		{
			APIGroups: []string{""},
			Resources: []string{"pods"},
			Verbs:     []string{"create", "get", "list", "watch", "update", "patch", "delete"},
		},
		{
			APIGroups: []string{"baremetal.openstack.org"},
			Resources: []string{"openstackprovisionservers"},
			Verbs:     []string{"get", "list"},
		},
		{
			APIGroups: []string{"baremetal.openstack.org"},
			Resources: []string{"openstackprovisionservers/status"},
			Verbs:     []string{"get", "list", "update"},
		},
	}
	rbacResult, err := common_rbac.ReconcileRbac(ctx, helper, instance, rbacRules)
	if err != nil {
		return rbacResult, err
	} else if (rbacResult != ctrl.Result{}) {
		return rbacResult, nil
	}
	// Handle non-deleted servers
	return r.reconcileNormal(ctx, instance, helper)
}

// SetupWithManager sets up the controller with the Manager.
func (r *OpenStackProvisionServerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&baremetalv1.OpenStackProvisionServer{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.ConfigMap{}).
		Complete(r)
}

func (r *OpenStackProvisionServerReconciler) reconcileDelete(ctx context.Context, instance *baremetalv1.OpenStackProvisionServer, helper *helper.Helper) (ctrl.Result, error) {
	r.Log.Info(fmt.Sprintf("Reconciling OpenStackProvisionServer '%s' delete", instance.Name))

	controllerutil.RemoveFinalizer(instance, helper.GetFinalizer())
	r.Log.Info(fmt.Sprintf("Reconciled OpenStackProvisionServer '%s' delete successfully", instance.Name))

	return ctrl.Result{}, nil
}

func (r *OpenStackProvisionServerReconciler) reconcileInit(
	ctx context.Context,
	instance *baremetalv1.OpenStackProvisionServer,
	helper *helper.Helper,
	serviceLabels map[string]string,
) (ctrl.Result, error) {
	r.Log.Info(fmt.Sprintf("Reconciling OpenStackProvisionServer '%s' init", instance.Name))

	r.Log.Info(fmt.Sprintf("Reconciled OpenStackProvisionServer '%s' init successfully", instance.Name))
	return ctrl.Result{}, nil
}

func (r *OpenStackProvisionServerReconciler) reconcileUpdate(ctx context.Context, instance *baremetalv1.OpenStackProvisionServer, helper *helper.Helper) (ctrl.Result, error) {
	r.Log.Info(fmt.Sprintf("Reconciling OpenStackProvisionServer '%s' update", instance.Name))

	// TODO: should have minor update tasks if required
	// - delete dbsync hash from status to rerun it?

	r.Log.Info(fmt.Sprintf("Reconciled OpenStackProvisionServer '%s' update successfully", instance.Name))
	return ctrl.Result{}, nil
}

func (r *OpenStackProvisionServerReconciler) reconcileUpgrade(ctx context.Context, instance *baremetalv1.OpenStackProvisionServer, helper *helper.Helper) (ctrl.Result, error) {
	r.Log.Info(fmt.Sprintf("Reconciling OpenStackProvisionServer '%s' upgrade", instance.Name))

	// TODO: should have major version upgrade tasks
	// -delete dbsync hash from status to rerun it?

	r.Log.Info(fmt.Sprintf("Reconciled OpenStackProvisionServer '%s' upgrade successfully", instance.Name))
	return ctrl.Result{}, nil
}

func (r *OpenStackProvisionServerReconciler) reconcileNormal(ctx context.Context, instance *baremetalv1.OpenStackProvisionServer, helper *helper.Helper) (ctrl.Result, error) {
	r.Log.Info(fmt.Sprintf("Reconciling OpenStackProvisionServer '%s'", instance.Name))

	//
	// Create ConfigMap required as input for the server and calculate an overall hash of hashes
	//

	configMapVars := make(map[string]env.Setter)

	//
	// create Configmap required for glance input
	// - %-scripts configmap holding scripts to e.g. bootstrap the service
	// - %-config configmap holding minimal glance config required to get the service up, user can add additional files to be added to the service
	// - parameters which has passwords gets added from the OpenStack secret via the init container
	//
	err := r.generateServiceConfigMaps(ctx, helper, instance, &configMapVars)
	if err != nil {
		instance.Status.Conditions.Set(condition.FalseCondition(
			condition.ServiceConfigReadyCondition,
			condition.ErrorReason,
			condition.SeverityWarning,
			condition.ServiceConfigReadyErrorMessage,
			err.Error()))
		return ctrl.Result{}, err
	}
	//
	// create hash over all the different input resources to identify if any those changed
	// and a restart/recreate is required.
	//
	inputHash, hashChanged, err := r.createHashOfInputHashes(ctx, instance, configMapVars)
	if err != nil {
		instance.Status.Conditions.Set(condition.FalseCondition(
			condition.ServiceConfigReadyCondition,
			condition.ErrorReason,
			condition.SeverityWarning,
			condition.ServiceConfigReadyErrorMessage,
			err.Error()))
		return ctrl.Result{}, err
	} else if hashChanged {
		// Hash changed and instance status should be updated (which will be done by main defer func),
		// so we need to return and reconcile again
		return ctrl.Result{}, nil
	}
	instance.Status.Conditions.MarkTrue(condition.ServiceConfigReadyCondition, condition.ServiceConfigReadyMessage)
	// Create ConfigMap - end

	// Get the provisioning interface of the cluster worker nodes from either Metal3
	// or from the instance spec itself if it was provided there
	provInterfaceName, err := r.getProvisioningInterfaceName(ctx, instance)
	if err != nil {
		instance.Status.Conditions.Set(condition.FalseCondition(
			baremetalv1.OpenStackProvisionServerProvIntfReadyCondition,
			condition.ErrorReason,
			condition.SeverityWarning,
			baremetalv1.OpenStackProvisionServerProvIntfReadyErrorMessage,
			err.Error()))
		return ctrl.Result{}, err
	}
	instance.Status.Conditions.MarkTrue(baremetalv1.OpenStackProvisionServerProvIntfReadyCondition, baremetalv1.OpenStackProvisionServerProvIntfReadyMessage)

	serviceLabels := labels.GetLabels(instance, openstackprovisionserver.AppLabel, map[string]string{
		common.AppSelector: instance.Name + "-deployment",
	})

	// Handle service init
	ctrlResult, err := r.reconcileInit(ctx, instance, helper, serviceLabels)
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

	oldLocalImageURL := instance.Status.LocalImageURL

	// If the deployment is not ready, we should not have anything set for the localImageURL,
	// but if it is ready we will set localImageURL properly below
	instance.Status.LocalImageURL = ""

	// Define a new Deployment object
	depl := deployment.NewDeployment(
		openstackprovisionserver.Deployment(instance, inputHash, serviceLabels, provInterfaceName),
		5,
	)

	ctrlResult, err = depl.CreateOrPatch(ctx, helper)
	if err != nil {
		instance.Status.Conditions.Set(condition.FalseCondition(
			condition.DeploymentReadyCondition,
			condition.ErrorReason,
			condition.SeverityWarning,
			condition.DeploymentReadyErrorMessage,
			err.Error()))
		return ctrlResult, err
	} else if (ctrlResult != ctrl.Result{}) {
		instance.Status.Conditions.Set(condition.FalseCondition(
			condition.DeploymentReadyCondition,
			condition.RequestedReason,
			condition.SeverityInfo,
			condition.DeploymentReadyRunningMessage))
		return ctrlResult, nil
	}
	instance.Status.ReadyCount = depl.GetDeployment().Status.ReadyReplicas
	if instance.Status.ReadyCount > 0 {
		instance.Status.Conditions.MarkTrue(condition.DeploymentReadyCondition, condition.DeploymentReadyMessage)
	} else {
		instance.Status.Conditions.Set(condition.FalseCondition(
			condition.DeploymentReadyCondition,
			condition.RequestedReason,
			condition.SeverityInfo,
			condition.DeploymentReadyRunningMessage))

		return ctrl.Result{RequeueAfter: time.Duration(10) * time.Second}, nil
	}
	// create Deployment - end

	//
	// Check whether instance.Status.ProvisionIp has been set by the side-car agent container
	// that is created with the deployment above and generate the LocalImageURL if so
	//
	// Provision IP Discovery Agent sets status' ProvisionIP
	if instance.Status.ProvisionIP == "" {
		instance.Status.Conditions.Set(condition.FalseCondition(
			baremetalv1.OpenStackProvisionServerLocalImageURLReadyCondition,
			condition.RequestedReason,
			condition.SeverityInfo,
			baremetalv1.OpenStackProvisionServerLocalImageURLReadyRunningMessage))
		return ctrlResult, nil
	}

	instance.Status.LocalImageURL = r.getLocalImageURL(instance)

	if oldLocalImageURL != instance.Status.LocalImageURL {
		r.Log.Info(fmt.Sprintf("OpenStackProvisionServer LocalImageURL changed: %s", instance.Status.LocalImageURL))
	}

	instance.Status.Conditions.MarkTrue(baremetalv1.OpenStackProvisionServerLocalImageURLReadyCondition, baremetalv1.OpenStackProvisionServerLocalImageURLReadyMessage)
	// check ProvisionIp/LocalImageURL - end

	r.Log.Info(fmt.Sprintf("Reconciled OpenStackProvisionServer '%s' successfully", instance.Name))
	return ctrl.Result{}, nil
}

// generateServiceConfigMaps - create create configmaps which hold scripts and service configuration
func (r *OpenStackProvisionServerReconciler) generateServiceConfigMaps(
	ctx context.Context,
	h *helper.Helper,
	instance *baremetalv1.OpenStackProvisionServer,
	envVars *map[string]env.Setter,
) error {
	//
	// create Configmap/Secret required for glance input
	// - %-scripts configmap holding scripts to e.g. bootstrap the service
	// - %-config configmap holding minimal glance config required to get the service up, user can add additional files to be added to the service
	// - parameters which has passwords gets added from the ospSecret via the init container
	//

	cmLabels := labels.GetLabels(instance, openstackprovisionserver.AppLabel, map[string]string{})

	templateParameters := make(map[string]interface{})
	templateParameters["Port"] = strconv.FormatInt(int64(instance.Spec.Port), 10)

	cms := []util.Template{
		// Apache server config
		{
			Name:               fmt.Sprintf("%s-httpd-config", instance.Name),
			Namespace:          instance.Namespace,
			Type:               util.TemplateTypeConfig,
			InstanceType:       instance.Kind,
			AdditionalTemplate: map[string]string{},
			Labels:             cmLabels,
			ConfigOptions:      templateParameters,
		},
	}
	err := configmap.EnsureConfigMaps(ctx, h, instance, cms, envVars)

	if err != nil {
		return nil
	}

	return nil
}

// createHashOfInputHashes - creates a hash of hashes which gets added to the resources which requires a restart
// if any of the input resources change, like configs, passwords, ...
//
// returns the hash, whether the hash changed (as a bool) and any error
func (r *OpenStackProvisionServerReconciler) createHashOfInputHashes(
	ctx context.Context,
	instance *baremetalv1.OpenStackProvisionServer,
	envVars map[string]env.Setter,
) (string, bool, error) {
	var hashMap map[string]string
	changed := false
	mergedMapVars := env.MergeEnvs([]corev1.EnvVar{}, envVars)
	hash, err := util.ObjectHash(mergedMapVars)
	if err != nil {
		return hash, changed, err
	}
	if hashMap, changed = util.SetHash(instance.Status.Hash, common.InputHashName, hash); changed {
		instance.Status.Hash = hashMap
		r.Log.Info(fmt.Sprintf("Input maps hash %s - %s", common.InputHashName, hash))
	}
	return hash, changed, nil
}

func (r *OpenStackProvisionServerReconciler) getProvisioningInterfaceName(
	ctx context.Context,
	instance *baremetalv1.OpenStackProvisionServer,
) (string, error) {
	// Get the provisioning interface of the cluster worker nodes from either Metal3
	// or from the instance spec itself if it was provided there
	var err error
	provInterfaceName := instance.Spec.Interface

	if provInterfaceName != "" {
		r.Log.Info(fmt.Sprintf("Provisioning interface supplied by %s spec", instance.Name))
	} else {
		r.Log.Info("Provisioning interface name not yet discovered, checking Metal3...")

		provInterfaceName, err = r.getProvisioningInterface(ctx, instance)

		if err != nil {
			return "", err
		}

		if provInterfaceName == "" {
			return "", fmt.Errorf("metal3 provisioning interface configuration not found")
		}
	}

	return provInterfaceName, nil
}

func (r *OpenStackProvisionServerReconciler) getProvisioningInterface(
	ctx context.Context,
	instance *baremetalv1.OpenStackProvisionServer,
) (string, error) {
	cfg, err := config.GetConfig()

	if err != nil {
		return "", err
	}

	dynClient, err := dynamic.NewForConfig(cfg)

	if err != nil {
		return "", err
	}

	provisioningsClient := dynClient.Resource(provisioningsGVR)

	provisioning, err := provisioningsClient.Get(ctx, "provisioning-configuration", metav1.GetOptions{})

	if err != nil {
		return "", err
	}

	provisioningSpecIntf := provisioning.Object["spec"]

	if provisioningSpec, ok := provisioningSpecIntf.(map[string]interface{}); ok {
		interfaceIntf := provisioningSpec["provisioningInterface"]

		if provInterfaceName, ok := interfaceIntf.(string); ok {
			r.Log.Info(fmt.Sprintf("Found provisioning interface %s in Metal3 config", provInterfaceName))
			return provInterfaceName, nil
		}
	}

	return "", nil
}

func (r *OpenStackProvisionServerReconciler) getLocalImageURL(instance *baremetalv1.OpenStackProvisionServer) string {
	baseFilename := instance.Spec.RhelImageURL[strings.LastIndex(instance.Spec.RhelImageURL, "/")+1 : len(instance.Spec.RhelImageURL)]
	baseFilenameEnd := baseFilename[len(baseFilename)-3:]

	if baseFilenameEnd == ".gz" || baseFilenameEnd == ".xz" {
		baseFilename = baseFilename[0 : len(baseFilename)-3]
	}

	return fmt.Sprintf("http://%s:%d/images/%s/compressed-%s", instance.Status.ProvisionIP, instance.Spec.Port, baseFilename, baseFilename)
}
