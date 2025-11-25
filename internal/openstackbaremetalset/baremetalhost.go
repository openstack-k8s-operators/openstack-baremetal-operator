// Package openstackbaremetalset provides functionality for managing OpenStack baremetal host sets
package openstackbaremetalset

import (
	"context"
	"fmt"
	"net"
	"regexp"
	"strings"

	metal3v1 "github.com/metal3-io/baremetal-operator/apis/metal3.io/v1alpha1"
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

// BaremetalHostProvision - Provision a BaremetalHost via Metal3
func BaremetalHostProvision(
	ctx context.Context,
	helper *helper.Helper,
	instance *baremetalv1.OpenStackBaremetalSet,
	bmh string,
	hostName string,
	ctlPlaneIP string,
	provServer *baremetalv1.OpenStackProvisionServer,
	sshSecret *corev1.Secret,
	passwordSecret *corev1.Secret,
	envVars *map[string]env.Setter,
) error {
	l := log.FromContext(ctx)
	//
	// Update status with BMH provisioning details
	//
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
	// Instance UserData/NetworkData
	userDataSecret := instance.Spec.BaremetalHosts[hostName].UserData
	networkDataSecret := instance.Spec.BaremetalHosts[hostName].NetworkData

	sts := []util.Template{}
	// User data cloud-init secret
	if userDataSecret == nil {
		templateParameters := make(map[string]any)
		templateParameters["AuthorizedKeys"] = strings.TrimSuffix(string(sshSecret.Data["authorized_keys"]), "\n")
		templateParameters["HostName"] = hostName
		//If Hostname is fqdn, use it
		if !hostNameIsFQDN(hostName) && instance.Spec.DomainName != "" {
			templateParameters["FQDN"] = strings.Join([]string{hostName, instance.Spec.DomainName}, ".")
		} else {
			templateParameters["FQDN"] = hostName
		}
		templateParameters["CloudUserName"] = instance.Spec.CloudUserName

		// Prepare cloudinit (create secret)
		secretLabels := labels.GetLabels(instance, labels.GetGroupLabel(baremetalv1.ServiceName), map[string]string{})
		if passwordSecret != nil && len(passwordSecret.Data["NodeRootPassword"]) > 0 {
			templateParameters["NodeRootPassword"] = string(passwordSecret.Data["NodeRootPassword"])
		}

		userDataSecretName := fmt.Sprintf(CloudInitUserDataSecretName, instance.Name, hostName)

		userDataSt := util.Template{
			Name:               userDataSecretName,
			Namespace:          instance.Spec.BmhNamespace,
			Type:               util.TemplateTypeConfig,
			InstanceType:       instance.Kind,
			AdditionalTemplate: map[string]string{"userData": "/openstackbaremetalset/cloudinit/userdata"},
			Labels:             secretLabels,
			ConfigOptions:      templateParameters,
		}
		sts = append(sts, userDataSt)
		userDataSecret = &corev1.SecretReference{
			Name:      userDataSecretName,
			Namespace: instance.Spec.BmhNamespace,
		}

	}

	if networkDataSecret == nil {

		// Check IP version and set template variables accordingly
		ipAddr, ipNet, err := net.ParseCIDR(ctlPlaneIP)
		if err != nil {
			return err
		}

		CtlplaneIPVersion := "ipv6"
		if ipAddr.To4() != nil {
			CtlplaneIPVersion = "ipv4"
		}

		templateParameters := make(map[string]any)
		templateParameters["CtlplaneIpVersion"] = CtlplaneIPVersion
		templateParameters["CtlplaneIp"] = ipAddr
		if instance.Spec.BaremetalHosts[hostName].CtlplaneVlan != nil {
			templateParameters["CtlplaneVlan"] = *instance.Spec.BaremetalHosts[hostName].CtlplaneVlan
		} else if instance.Spec.CtlplaneVlan != nil {
			templateParameters["CtlplaneVlan"] = *instance.Spec.CtlplaneVlan
		}
		if instance.Spec.BaremetalHosts[hostName].CtlplaneInterface != "" {
			templateParameters["CtlplaneInterface"] = instance.Spec.BaremetalHosts[hostName].CtlplaneInterface
		} else {
			templateParameters["CtlplaneInterface"] = instance.Spec.CtlplaneInterface
		}
		if instance.Spec.BaremetalHosts[hostName].CtlplaneGateway != "" {
			templateParameters["CtlplaneGateway"] = instance.Spec.BaremetalHosts[hostName].CtlplaneGateway
		} else {
			templateParameters["CtlplaneGateway"] = instance.Spec.CtlplaneGateway
		}
		// Handle bonding configuration from template spec
		if instance.Spec.CtlplaneBond != nil {
			templateParameters["CtlplaneBondInterfaces"] = instance.Spec.CtlplaneBond.BondInterfaces
			if instance.Spec.CtlplaneBond.BondMode != "" {
				templateParameters["CtlplaneBondMode"] = instance.Spec.CtlplaneBond.BondMode
			}
			if len(instance.Spec.CtlplaneBond.BondOptions) > 0 {
				templateParameters["CtlplaneBondOptions"] = instance.Spec.CtlplaneBond.BondOptions
			}
		}
		templateParameters["CtlplaneNetmask"] = net.IP(ipNet.Mask)
		if len(instance.Spec.BootstrapDNS) > 0 {
			templateParameters["CtlplaneDns"] = instance.Spec.BootstrapDNS
		} else {
			templateParameters["CtlplaneDns"] = []string{}
		}

		if len(instance.Spec.DNSSearchDomains) > 0 {
			templateParameters["CtlplaneDnsSearch"] = instance.Spec.DNSSearchDomains
		} else {
			templateParameters["CtlplaneDnsSearch"] = []string{}
		}

		networkDataSecretName := fmt.Sprintf(CloudInitNetworkDataSecretName, instance.Name, hostName)

		// Flag the network data secret as safe to collect with must-gather
		secretLabelsWithMustGather := labels.GetLabels(instance, labels.GetGroupLabel(baremetalv1.ServiceName), map[string]string{
			MustGatherSecret: "yes",
		})

		networkDataSt := util.Template{
			Name:               networkDataSecretName,
			Namespace:          instance.Spec.BmhNamespace,
			Type:               util.TemplateTypeConfig,
			InstanceType:       instance.Kind,
			AdditionalTemplate: map[string]string{"networkData": "/openstackbaremetalset/cloudinit/networkdata"},
			Labels:             secretLabelsWithMustGather,
			ConfigOptions:      templateParameters,
		}
		sts = append(sts, networkDataSt)
		networkDataSecret = &corev1.SecretReference{
			Name:      networkDataSecretName,
			Namespace: instance.Spec.BmhNamespace,
		}
	}

	if len(sts) > 0 {
		err := oko_secret.EnsureSecrets(ctx, helper, instance, sts, envVars)
		if err != nil {
			return err
		}
	}

	//
	// Provision the BaremetalHost
	//
	foundBaremetalHost := &metal3v1.BareMetalHost{}
	err := helper.GetClient().Get(ctx, types.NamespacedName{Name: bmh, Namespace: instance.Spec.BmhNamespace}, foundBaremetalHost)
	if err != nil {
		return err
	}
	op, err := controllerutil.CreateOrPatch(ctx, helper.GetClient(), foundBaremetalHost, func() error {
		// Set our ownership labels so we can watch this resource and also indicate that this BMH
		// belongs to the particular OSBMS.Spec.BaremetalHosts entry we have passed to this function.
		// Set ownership labels that can be found by the respective controller kind
		labelSelector := labels.GetLabels(instance, labels.GetGroupLabel(baremetalv1.ServiceName), map[string]string{
			fmt.Sprintf("%s%s", instance.Name, HostnameLabelSelectorSuffix): hostName,
		})
		foundBaremetalHost.Labels = util.MergeStringMaps(
			foundBaremetalHost.GetLabels(),
			labelSelector,
		)

		// Ensure AutomatedCleaningMode is set as per spec
		foundBaremetalHost.Spec.AutomatedCleaningMode = metal3v1.AutomatedCleaningMode(instance.Spec.AutomatedCleaningMode)

		//
		// Ensure the image url is up to date unless already provisioned
		//
		if foundBaremetalHost.Status.Provisioning.State != metal3v1.StateProvisioned {
			if instance.Spec.OSImageDeploymentType == baremetalv1.OSImageDeploymentTypePassThrough {
				// PassThrough mode: use container URL directly
				foundBaremetalHost.Spec.Image = &metal3v1.Image{
					URL: instance.Spec.OSContainerImageURL,
				}
			} else {
				// SelfExtracting mode: use provision server
				foundBaremetalHost.Spec.Image = &metal3v1.Image{
					URL:          provServer.Status.LocalImageURL,
					Checksum:     provServer.Status.LocalImageChecksumURL,
					ChecksumType: provServer.Status.OSImageChecksumType,
				}
			}
		}

		//
		// Update the BMH spec once when ConsumerRef is nil to only perform one time provision.
		//
		if foundBaremetalHost.Spec.ConsumerRef == nil {
			foundBaremetalHost.Spec.Online = true
			foundBaremetalHost.Spec.ConsumerRef = &corev1.ObjectReference{Name: instance.Name, Kind: instance.Kind, Namespace: instance.Namespace}
			if instance.Spec.OSImageDeploymentType == baremetalv1.OSImageDeploymentTypePassThrough {
				// PassThrough mode: use container URL directly
				foundBaremetalHost.Spec.Image = &metal3v1.Image{
					URL: instance.Spec.OSContainerImageURL,
				}
			} else {
				// SelfExtracting mode: use provision server
				foundBaremetalHost.Spec.Image = &metal3v1.Image{
					URL:          provServer.Status.LocalImageURL,
					Checksum:     provServer.Status.LocalImageChecksumURL,
					ChecksumType: provServer.Status.OSImageChecksumType,
				}
			}
			foundBaremetalHost.Spec.UserData = userDataSecret
			foundBaremetalHost.Spec.NetworkData = networkDataSecret
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
	bmhStatus.UserDataSecretName = userDataSecret.Name
	bmhStatus.NetworkDataSecretName = networkDataSecret.Name
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

	baremetalHost := &metal3v1.BareMetalHost{}
	err := helper.GetClient().Get(ctx, types.NamespacedName{Name: bmhStatus.BmhRef, Namespace: instance.Spec.BmhNamespace}, baremetalHost)
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
			instance.Spec.BmhNamespace,
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

// NodeHostNameIsFQDN Helper to check if a hostname is fqdn
func hostNameIsFQDN(hostname string) bool {
	// Regular expression to match a valid FQDN
	// This regex assumes that the hostname and domain name segments only contain letters, digits, hyphens, and periods.
	regex := `^([a-zA-Z0-9-]+\.)*[a-zA-Z0-9-]+\.[a-zA-Z]{2,}$`

	match, _ := regexp.MatchString(regex, hostname)
	return match
}
