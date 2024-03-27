package v1beta1

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-logr/logr"
	metal3v1 "github.com/metal3-io/baremetal-operator/apis/metal3.io/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	goClient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// ServiceName -
	ServiceName = "openstackbaremetalset"
)

// GetBaremetalHosts - Get all BaremetalHosts in the chosen namespace with (optional) labels
func GetBaremetalHosts(
	ctx context.Context,
	c goClient.Client,
	namespace string,
	labelSelector map[string]string,
) (*metal3v1.BareMetalHostList, error) {

	bmhHostsList := &metal3v1.BareMetalHostList{}

	listOpts := []client.ListOption{
		client.InNamespace(namespace),
	}

	if len(labelSelector) > 0 {
		labels := client.MatchingLabels(labelSelector)
		listOpts = append(listOpts, labels)
	}

	err := c.List(ctx, bmhHostsList, listOpts...)
	if err != nil {
		return nil, err
	}
	return bmhHostsList, nil

}

// VerifyBaremetalStatusBmhRefs - Verify that BMHs haven't been improperly deleted
// outside of our prescribed annotate-and-scale-count-down method.  If bad deletions
// have occurred, we return an error to halt further reconciliation that could lead
// to an inconsistent state for instance.Status.BaremetalHosts.
func VerifyBaremetalStatusBmhRefs(
	ctx context.Context,
	c goClient.Client,
	instance *OpenStackBaremetalSet,
) error {
	// Get all BaremetalHosts
	allBaremetalHosts, err := GetBaremetalHosts(
		ctx,
		c,
		instance.Spec.BmhNamespace,
		map[string]string{},
	)
	if err != nil {
		return err
	}

	for _, bmhStatus := range instance.Status.BaremetalHosts {
		found := false

		for _, bmh := range allBaremetalHosts.Items {
			if bmh.Name == bmhStatus.BmhRef {
				found = true
				break
			}
		}

		if !found {
			err := fmt.Errorf("existing BaremetalHost \"%s\" not found for OpenStackBaremetalSet %s.  "+
				"Please check BaremetalHost resources and re-add \"%s\" to continue",
				bmhStatus.BmhRef, instance.Name, bmhStatus.BmhRef)

			return err
		}
	}

	return nil
}

// VerifyBaremetalSetScaleUp -
func VerifyBaremetalSetScaleUp(
	l logr.Logger,
	instance *OpenStackBaremetalSet,
	allBmhs *metal3v1.BareMetalHostList,
	existingBmhs *metal3v1.BareMetalHostList) (map[string]metal3v1.BareMetalHost, error) {
	// How many new BaremetalHost allocations do we need (if any)?
	newBmhsNeededCount := len(instance.Spec.BaremetalHosts) - len(existingBmhs.Items)
	selectedBaremetalHosts := map[string]metal3v1.BareMetalHost{}

	labelStr := ""

	if newBmhsNeededCount > 0 {
		if len(instance.Spec.BmhLabelSelector) > 0 {
			labelStr = fmt.Sprintf("%v", instance.Spec.BmhLabelSelector)
			labelStr = strings.Replace(labelStr, "map[", "[", 1)
		}

		l.Info("Attempting to find BaremetalHosts for scale-up of OpenStackBaremetalSet", "OpenStackBaremetalSet",
			instance.Name, "namespace", instance.Spec.BmhNamespace, "quantity", newBmhsNeededCount, "labels", labelStr)

		selectedCount := 0
		for _, baremetalHost := range allBmhs.Items {

			if selectedCount == newBmhsNeededCount {
				break
			}
			mismatch := false
			hostName, matched := verifyBaremetalSetInstanceLabelMatch(l, instance, &baremetalHost)
			if !matched {
				l.Info("BaremetalHost cannot be used as it does not match node labels for", "BMH", baremetalHost.ObjectMeta.Name)
				mismatch = true
			}

			if !verifyBaremetalSetHardwareMatch(l, instance, &baremetalHost) {
				l.Info("BaremetalHost cannot be used because it does not match hardware requirements", "BMH", baremetalHost.ObjectMeta.Name)
				mismatch = true
			}

			if baremetalHost.Status.Provisioning.State != metal3v1.StateAvailable {
				l.Info("BaremetalHost ProvisioningState is not 'Available'")
				mismatch = true
			}

			if baremetalHost.Spec.Online {
				l.Info("BaremetalHost cannot be used because it is already online", "BMH", baremetalHost.ObjectMeta.Name)
				mismatch = true
			}

			if baremetalHost.Spec.ConsumerRef != nil {
				l.Info("BaremetalHost cannot be used because it already has a consumerRef", "BMH", baremetalHost.ObjectMeta.Name)
				mismatch = true
			}

			// If for any reason we can't use this BMH, do not add to the list of available BMHs
			if mismatch {
				continue
			}

			l.Info("Available BaremetalHost", "BMH", baremetalHost.ObjectMeta.Name)

			selectedBaremetalHosts[hostName] = baremetalHost
			selectedCount++
		}
	}
	// If we can't satisfy the new requested BaremetalHost count, explicitly state so
	if newBmhsNeededCount > len(selectedBaremetalHosts) {
		errLabelStr := ""

		if labelStr != "" {
			errLabelStr = fmt.Sprintf(" with labels %s", labelStr)
		}

		return nil, fmt.Errorf("unable to find %d requested BaremetalHosts%s in namespace %s for scale-up (%d in use, %d available)",
			len(instance.Spec.BaremetalHosts),
			errLabelStr,
			instance.Spec.BmhNamespace,
			len(existingBmhs.Items),
			len(selectedBaremetalHosts))
	}

	l.Info("Found sufficient quantity of BaremetalHosts for scale-up of OpenStackBaremetalSet",
		"OpenStackBaremetalSet", instance.Name, "namespace", instance.Spec.BmhNamespace, "BMHs",
		selectedBaremetalHosts, "labels", labelStr)

	return selectedBaremetalHosts, nil
}

// VerifyBaremetalSetScaleDown - TODO: not needed at the current moment
func VerifyBaremetalSetScaleDown(
	instance *OpenStackBaremetalSet,
	existingBmhs *metal3v1.BareMetalHostList,
	removalAnnotatedBmhCount int) error {
	// How many new BaremetalHost de-allocations do we need (if any)?
	bmhsToRemoveCount := len(existingBmhs.Items) - len(instance.Spec.BaremetalHosts)

	if bmhsToRemoveCount > removalAnnotatedBmhCount {
		return fmt.Errorf("unable to find sufficient amount of BaremetalHosts annotated for scale-down (%d found, %d requested)",
			removalAnnotatedBmhCount,
			bmhsToRemoveCount)
	}

	return nil
}

func verifyBaremetalSetInstanceLabelMatch(
	l logr.Logger,
	instance *OpenStackBaremetalSet,
	bmh *metal3v1.BareMetalHost) (string, bool) {

	bmhLabels := bmh.GetLabels()
	for hostName, instanceSpec := range instance.Spec.BaremetalHosts {
		if IsMapSubset(bmhLabels, instanceSpec.BmhLabelSelector) {
			return hostName, true
		}
	}
	l.Info("BaremetalHost does not match any of the node labels as requested", "BMH", bmh.ObjectMeta.Name)
	return "", false
}

func IsMapSubset[K, V comparable](m map[K]V, sub map[K]V) bool {
	if sub == nil {
		return true
	}
	if len(sub) > len(m) {
		return false
	}
	for k, vsub := range sub {
		if vm, found := m[k]; !found || vm != vsub {
			return false
		}
	}
	return true
}

func verifyBaremetalSetHardwareMatch(
	l logr.Logger,
	instance *OpenStackBaremetalSet,
	bmh *metal3v1.BareMetalHost,
) bool {
	// If no requested hardware requirements, we're all set
	if instance.Spec.HardwareReqs == (HardwareReqs{}) {
		return true
	}

	// Can't make comparisons if the BMH lacks hardware details
	if bmh.Status.HardwareDetails == nil {
		l.Info("WARNING: BaremetalHost lacks hardware details in status; cannot verify against hardware requests!", "BMH", bmh.Name)
		return false
	}

	cpuReqs := instance.Spec.HardwareReqs.CPUReqs

	// CPU architecture is always exact-match only
	if cpuReqs.Arch != "" && bmh.Status.HardwareDetails.CPU.Arch != cpuReqs.Arch {
		l.Info("BaremetalHost CPU arch does not match request",
			"BMH",
			bmh.Name,
			"CPU arch",
			bmh.Status.HardwareDetails.CPU.Arch,
			"CPU arch request",
			cpuReqs.Arch)

		return false
	}

	// CPU count can be exact-match or (default) greater
	if cpuReqs.CountReq.Count != 0 && bmh.Status.HardwareDetails.CPU.Count != cpuReqs.CountReq.Count {
		if cpuReqs.CountReq.ExactMatch || cpuReqs.CountReq.Count > bmh.Status.HardwareDetails.CPU.Count {
			l.Info("BaremetalHost CPU count does not match request",
				"BMH",
				bmh.Name,
				"CPU count",
				bmh.Status.HardwareDetails.CPU.Count,
				"CPU count request",
				cpuReqs.CountReq.Count)

			return false
		}
	}

	// CPU clock speed can be exact-match or (default) greater
	if cpuReqs.MhzReq.Mhz != 0 {
		clockSpeed := int(bmh.Status.HardwareDetails.CPU.ClockMegahertz)
		if cpuReqs.MhzReq.Mhz != clockSpeed && (cpuReqs.MhzReq.ExactMatch || cpuReqs.MhzReq.Mhz > clockSpeed) {
			l.Info("BaremetalHost CPU mhz does not match request",
				"BMH",
				bmh.Name,
				"CPU mhz",
				clockSpeed,
				"CPU mhz request",
				cpuReqs.MhzReq.Mhz)

			return false
		}
	}

	memReqs := instance.Spec.HardwareReqs.MemReqs

	// Memory GBs can be exact-match or (default) greater
	if memReqs.GbReq.Gb != 0 {
		memGbBms := float64(memReqs.GbReq.Gb)
		memGbBmh := float64(bmh.Status.HardwareDetails.RAMMebibytes) / float64(1024)

		if memGbBmh != memGbBms && (memReqs.GbReq.ExactMatch || memGbBms > memGbBmh) {
			l.Info("BaremetalHost memory size does not match request",
				"BMH",
				bmh.Name,
				"Memory size",
				memGbBmh,
				"Memory size request",
				memGbBms)

			return false
		}
	}

	diskReqs := instance.Spec.HardwareReqs.DiskReqs

	var foundDisk *metal3v1.Storage

	if diskReqs.GbReq.Gb != 0 {
		diskGbBms := float64(diskReqs.GbReq.Gb)
		// TODO: Make sure there's at least one disk of this size?
		for _, disk := range bmh.Status.HardwareDetails.Storage {
			diskGbBmh := float64(disk.SizeBytes) / float64(1073741824)

			if diskGbBmh == diskGbBms || (!diskReqs.GbReq.ExactMatch && diskGbBmh > diskGbBms) {
				foundDisk = &disk
				break
			}
		}

		if foundDisk == nil {
			l.Info("BaremetalHost does not contain a disk of proper size that matches request",
				"BMH",
				bmh.Name,
				"Disk size request",
				diskGbBms)

			return false
		}
	}

	// We only care about the SSD flag if the user requested an exact match for it or if SSD is true
	if diskReqs.SSDReq.ExactMatch || diskReqs.SSDReq.SSD {
		found := false

		// If we matched on a disk for a GbReqs above, we need to match on the same disk
		if foundDisk != nil {
			if foundDisk.Rotational != diskReqs.SSDReq.SSD {
				found = true
			}
		} else {
			// TODO: Just need to match on any disk?
			for _, disk := range bmh.Status.HardwareDetails.Storage {
				if disk.Rotational != diskReqs.SSDReq.SSD {
					found = true
				}
			}
		}

		if !found {
			l.Info("BaremetalHost does not contain a disk that matches 'is rotational' request",
				"BMH",
				bmh.Name,
				"Rotational disk wanted",
				diskReqs.SSDReq.SSD)

			return false
		}
	}

	l.Info("BaremetalHost satisfies hardware requirements", "BMH", bmh.Name)

	return true
}
