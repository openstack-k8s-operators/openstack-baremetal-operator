package openstackbaremetalset

import (
	"context"
	"fmt"
	"strings"

	metal3v1alpha1 "github.com/metal3-io/baremetal-operator/apis/metal3.io/v1alpha1"
	"github.com/openstack-k8s-operators/lib-common/modules/common/env"
	"github.com/openstack-k8s-operators/lib-common/modules/common/helper"
	"github.com/openstack-k8s-operators/lib-common/modules/common/labels"
	oko_secret "github.com/openstack-k8s-operators/lib-common/modules/common/secret"
	"github.com/openstack-k8s-operators/lib-common/modules/common/util"
	baremetalv1 "github.com/openstack-k8s-operators/openstack-baremetal-operator/api/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// Provision a BaremetalHost via Metal3
func BaremetalHostProvision(
	ctx context.Context,
	helper *helper.Helper,
	instance *baremetalv1.OpenStackBaremetalSet,
	bmh string,
	hostName string,
	ctlPlaneIP string,
	localImageURL string,
	sshSecret *corev1.Secret,
	passwordSecret *corev1.Secret,
	envVars *map[string]env.Setter,
) error {
	l := log.FromContext(ctx)

	//
	// Get the associated BaremetalHost status entry for this hostname
	//
	// TODO: To be reworked when IPAM integrated
	var ok bool
	var bmhStatus baremetalv1.HostStatus

	if bmhStatus, ok = instance.Status.BaremetalHosts[hostName]; !ok {
		bmhStatus = baremetalv1.HostStatus{

			IPStatus: baremetalv1.IPStatus{
				Hostname:    hostName,
				BmhRef:      bmh,
				IPAddresses: map[string]string{},
			},
		}
		bmhStatus.IPAddresses["ctlplane"] = ctlPlaneIP
	}

	// User data cloud-init secret
	templateParameters := make(map[string]interface{})
	templateParameters["AuthorizedKeys"] = strings.TrimSuffix(string(sshSecret.Data["authorized_keys"]), "\n")
	templateParameters["Hostname"] = bmhStatus.Hostname
	templateParameters["DomainName"] = instance.Spec.DomainName

	// Prepare cloudinit (create secret)
	sts := []util.Template{}
	secretLabels := labels.GetLabels(instance, labels.GetGroupLabel(baremetalv1.ServiceName), map[string]string{})

	if passwordSecret != nil && len(passwordSecret.Data["NodeRootPassword"]) > 0 {
		templateParameters["NodeRootPassword"] = string(passwordSecret.Data["NodeRootPassword"])
	}

	userDataSecretName := fmt.Sprintf(CloudInitUserDataSecretName, instance.Name, bmh)

	userDataSt := util.Template{
		Name:               userDataSecretName,
		Namespace:          "openshift-machine-api",
		Type:               util.TemplateTypeConfig,
		InstanceType:       instance.Kind,
		AdditionalTemplate: map[string]string{"userData": "/openstackbaremetalset/cloudinit/userdata"},
		Labels:             secretLabels,
		ConfigOptions:      templateParameters,
	}

	sts = append(sts, userDataSt)

	// TODO: Get network information from somewhere else?

	// Network data cloud-init secret
	templateParameters = make(map[string]interface{})
	templateParameters["CtlplaneIp"] = ctlPlaneIP
	templateParameters["CtlplaneInterface"] = instance.Spec.CtlplaneInterface
	templateParameters["CtlplaneGateway"] = instance.Spec.CtlplaneGateway
	templateParameters["CtlplaneNetmask"] = instance.Spec.CtlplaneNetmask
	if len(instance.Spec.BootstrapDNS) > 0 {
		templateParameters["CtlplaneDns"] = instance.Spec.BootstrapDNS
	} else {
		templateParameters["CtlplaneDns"] = []string{}
	}

	if len(instance.Spec.DNSSearchDomains) > 0 {
		templateParameters["CtlplaneDnsSearch"] = instance.Spec.DNSSearchDomains
		// } else {
		// 	templateParameters["CtlplaneDnsSearch"] = osNetCfg.Spec.DNSSearchDomains
	}

	networkDataSecretName := fmt.Sprintf(CloudInitNetworkDataSecretName, instance.Name, bmh)

	// Flag the network data secret as safe to collect with must-gather
	secretLabelsWithMustGather := labels.GetLabels(instance, labels.GetGroupLabel(baremetalv1.ServiceName), map[string]string{
		MustGatherSecret: "yes",
	})

	networkDataSt := util.Template{
		Name:               networkDataSecretName,
		Namespace:          "openshift-machine-api",
		Type:               util.TemplateTypeConfig,
		InstanceType:       instance.Kind,
		AdditionalTemplate: map[string]string{"networkData": "/openstackbaremetalset/cloudinit/networkdata"},
		Labels:             secretLabelsWithMustGather,
		ConfigOptions:      templateParameters,
	}

	sts = append(sts, networkDataSt)

	err := oko_secret.EnsureSecrets(ctx, helper, instance, sts, envVars)
	if err != nil {
		return err
	}

	//
	// Provision the BaremetalHost
	//
	foundBaremetalHost := &metal3v1alpha1.BareMetalHost{}
	err = helper.GetClient().Get(ctx, types.NamespacedName{Name: bmh, Namespace: "openshift-machine-api"}, foundBaremetalHost)
	if err != nil {
		return err
	}

	op, err := controllerutil.CreateOrPatch(ctx, helper.GetClient(), foundBaremetalHost, func() error {
		// Set our ownership labels so we can watch this resource and also indicate that this BMH
		// belongs to the particular OSBMS.Spec.BaremetalHosts entry we have passed to this function.
		// Set ownership labels that can be found by the respective controller kind
		labelSelector := labels.GetLabels(instance, labels.GetGroupLabel(baremetalv1.ServiceName), map[string]string{
			fmt.Sprintf("%s%s", instance.Name, HostnameLabelSelectorSuffix): bmhStatus.Hostname,
		})
		foundBaremetalHost.Labels = util.MergeStringMaps(
			foundBaremetalHost.GetLabels(),
			labelSelector,
		)

		//
		// Update the BMH spec once when ConsumerRef is nil to only perform one time provision.
		//
		if foundBaremetalHost.Spec.ConsumerRef == nil {
			foundBaremetalHost.Spec.Online = true
			foundBaremetalHost.Spec.ConsumerRef = &corev1.ObjectReference{Name: instance.Name, Kind: instance.Kind, Namespace: instance.Namespace}
			foundBaremetalHost.Spec.Image = &metal3v1alpha1.Image{
				URL:      localImageURL,
				Checksum: fmt.Sprintf("%s.md5sum", localImageURL),
			}
			foundBaremetalHost.Spec.UserData = &corev1.SecretReference{
				Name:      userDataSecretName,
				Namespace: "openshift-machine-api",
			}
			foundBaremetalHost.Spec.NetworkData = &corev1.SecretReference{
				Name:      networkDataSecretName,
				Namespace: "openshift-machine-api",
			}
		}

		return nil
	})

	if err != nil {
		return err
	}

	if op != controllerutil.OperationResultNone {
		l.Info("BaremetalHost successfully reconciled", "BMH", foundBaremetalHost.Name, "operation", string(op))
	}

	//
	// Update status with BMH provisioning details
	//
	bmhStatus.UserDataSecretName = userDataSecretName
	bmhStatus.NetworkDataSecretName = networkDataSecretName
	bmhStatus.ProvisioningState = baremetalv1.ProvisioningState(foundBaremetalHost.Status.Provisioning.State)
	instance.Status.BaremetalHosts[hostName] = bmhStatus

	return nil
}

// BaremetalHostDeprovision - Deprovision a BaremetalHost via Metal3 and return the OSP compute hostname that was deleted
func BaremetalHostDeprovision(
	ctx context.Context,
	helper *helper.Helper,
	instance *baremetalv1.OpenStackBaremetalSet,
	bmhStatus baremetalv1.HostStatus,
) error {
	l := log.FromContext(ctx)

	baremetalHost := &metal3v1alpha1.BareMetalHost{}
	err := helper.GetClient().Get(ctx, types.NamespacedName{Name: bmhStatus.BmhRef, Namespace: "openshift-machine-api"}, baremetalHost)
	if err != nil {
		return err
	}

	l.Info("Deallocating BaremetalHost", "BMH", baremetalHost.Name)

	// Remove our ownership labels
	baremetalHostLabels := baremetalHost.GetObjectMeta().GetLabels()
	labelSelector := labels.GetLabels(instance, labels.GetGroupLabel(baremetalv1.ServiceName), map[string]string{
		fmt.Sprintf("%s%s", instance.Name, HostnameLabelSelectorSuffix): bmhStatus.Hostname,
	})
	for key := range labelSelector {
		delete(baremetalHostLabels, key)
	}
	baremetalHost.GetObjectMeta().SetLabels(baremetalHostLabels)

	// Remove deletion annotation (if any)
	annotations := baremetalHost.GetObjectMeta().GetAnnotations()
	delete(annotations, HostRemovalAnnotation)
	baremetalHost.GetObjectMeta().SetAnnotations(annotations)

	baremetalHost.Spec.Online = false
	baremetalHost.Spec.ConsumerRef = nil
	baremetalHost.Spec.Image = nil
	baremetalHost.Spec.UserData = nil
	baremetalHost.Spec.NetworkData = nil
	err = helper.GetClient().Update(ctx, baremetalHost)
	if err != nil {
		return err
	}

	l.Info("BaremetalHost deleted", "BMH", baremetalHost.Name, "Hostname", bmhStatus.Hostname)

	// Also remove userdata and networkdata secrets
	for _, secret := range []string{
		fmt.Sprintf(CloudInitUserDataSecretName, instance.Name, bmhStatus.BmhRef),
		fmt.Sprintf(CloudInitNetworkDataSecretName, instance.Name, bmhStatus.BmhRef),
	} {
		err = oko_secret.DeleteSecretsWithName(
			ctx,
			helper,
			secret,
			"openshift-machine-api",
		)
		if err != nil {
			return err
		}

		// It seems the lib-common DeleteSecretsWithName log this already
		//l.Info("BMH data secret deleted", "BMH", bmhStatus.BmhRef, "Secret", secret)
	}

	// Set status (remove this BaremetalHost entry)
	delete(instance.Status.BaremetalHosts, bmhStatus.Hostname)

	return nil
}
