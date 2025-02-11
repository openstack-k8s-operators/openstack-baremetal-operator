package v1beta1

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/go-logr/logr"
	metal3v1 "github.com/metal3-io/baremetal-operator/apis/metal3.io/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	goClient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// ServiceName -
	ServiceName                    = "openstackbaremetalset"
	IndividualComputeLabelMismatch = "one or more computes did not match the available Baremetalhosts due to their bmhLabelSelector(s)"
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

// VerifyAndSyncBaremetalStatusBmhRefs - Verify that BMHs haven't been improperly deleted
// outside of OpenStackBaremetalSet.  If deletions have occurred, we sync the state
// of instance.Status.BaremetalHosts.
func VerifyAndSyncBaremetalStatusBmhRefs(
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

	for computeName, bmhStatus := range instance.Status.BaremetalHosts {
		found := false
		for _, bmh := range allBaremetalHosts.Items {
			if bmh.Name == bmhStatus.BmhRef {
				found = true
				break
			}
		}

		if !found {
			// bmh could be deleted without us knowing about it
			delete(instance.Status.BaremetalHosts, computeName)
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

	// Figure out which compute hosts are new
	newComputes := map[string]InstanceSpec{}

	for hostName, compute := range instance.Spec.BaremetalHosts {
		// Any host name not found in the instance's status' BaremetalHosts map
		// is a new compute
		if _, found := instance.Status.BaremetalHosts[hostName]; !found {
			newComputes[hostName] = compute
		}
	}

	// How many new BaremetalHost allocations do we need (if any)?
	newBmhsNeededCount := len(newComputes)
	availableBaremetalHosts := []metal3v1.BareMetalHost{}
	var selectedBaremetalHosts map[string]metal3v1.BareMetalHost

	labelStr := ""
	errIndividualLabelsStr := ""

	if newBmhsNeededCount > 0 {
		if len(instance.Spec.BmhLabelSelector) > 0 {
			labelStr = fmt.Sprintf("%v", instance.Spec.BmhLabelSelector)
			labelStr = strings.Replace(labelStr, "map[", "[", 1)
		}

		l.Info("Attempting to find BaremetalHosts for scale-up of OpenStackBaremetalSet", "OpenStackBaremetalSet",
			instance.Name, "namespace", instance.Spec.BmhNamespace, "quantity", newBmhsNeededCount, "labels", labelStr)

		// First find BMHs that match everything WITHOUT considering individual compute host labels
		for _, baremetalHost := range allBmhs.Items {
			mismatch := false

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

			l.Info("Available BaremetalHost (compute labels not yet processed)", "BMH", baremetalHost.ObjectMeta.Name)

			availableBaremetalHosts = append(availableBaremetalHosts, baremetalHost)
		}

		// We only want to continue to individual compute label matching if we actually have
		// enough BMHs remaining given the filtering above
		if len(availableBaremetalHosts) >= newBmhsNeededCount {

			// Now try to fit all the requested new compute hosts into the set of available BMHs
			// given bmhLabelSelectors on each compute (if any) and the labels on the BMHs
			//
			// The problem we are trying to solve can be demonstrated through an example...
			// Imagine we have 3 available BMHs at this point with the following simplified labels:
			//
			// BMH1 (A, B, C)
			// BMH2 (A, C)
			// BMH3 (A, B)
			//
			// Imagine we have 3 new compute hosts with the following simplified labels:
			//
			// COMP1 (A, B)
			// COMP2 (A, C)
			// COMP3 (A, B)
			//
			// We want to make compute-to-BMH selections that allow all computes to find a BMH,
			// but imagine if the following valid matches were chosen for the first two computes:
			//
			// COMP1 -> BMH3 ((A, B) is a subset of (A, B))
			// COMP2 -> BMH1 ((A, C) is a subset of (A, B, C))
			//
			// Now trying to match COMP3, nothing remaining fits, because the potential satisfactory
			// matches, BMH1 and BMH3, were consumed by the first two computes already.  We would
			// have preferred instead that our algorithm made either of these sets of selections:
			//
			// COMP1 -> BMH1 ((A, B) is a subset of (A, B, C))
			// COMP2 -> BMH2 ((A, C) is a subset of (A, C))
			// COMP3 -> BMH3 ((A, B) is a subset of (A, B))
			// OR
			// COMP1 -> BMH3 ((A, B) is a subset of (A, B))
			// COMP2 -> BMH2 ((A, C) is a subset of (A, C))
			// COMP3 -> BMH1 ((A, B) is a subset of (A, B, C))
			//
			// The function called here accomplishes this (see its definition below for details)...

			selectedBaremetalHosts = findValidBaremetalSetInstanceLabelAssignments(newComputes, availableBaremetalHosts)

			if len(selectedBaremetalHosts) < 1 {
				l.Info("Unable to match requested new computes to satisfactory set of BaremetalHosts due to labeling")
				errIndividualLabelsStr = fmt.Sprintf(": %s", IndividualComputeLabelMismatch)
			}
		}
	}

	// If we can't satisfy the new requested BaremetalHost count, explicitly state so
	if newBmhsNeededCount > len(selectedBaremetalHosts) {
		errLabelStr := ""

		if labelStr != "" {
			errLabelStr = fmt.Sprintf(" with labels %s", labelStr)
		}

		return nil, fmt.Errorf("unable to find %d requested BaremetalHosts%s in namespace %s for scale-up (%d in use, %d available)%s",
			len(instance.Spec.BaremetalHosts),
			errLabelStr,
			instance.Spec.BmhNamespace,
			len(existingBmhs.Items),
			len(availableBaremetalHosts),
			errIndividualLabelsStr)
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

// Function to find valid assignments for computes-to-BMHs, given their labels, using backtracking
func findValidBaremetalSetInstanceLabelAssignments(computes map[string]InstanceSpec, bmhs []metal3v1.BareMetalHost) map[string]metal3v1.BareMetalHost {
	// First create map of valid computes-to-BMHs possibilities
	computesToBmhs := map[string][]metal3v1.BareMetalHost{}
	for compName, comp := range computes {
		for _, bmh := range bmhs {
			if IsMapSubset(bmh.GetLabels(), comp.BmhLabelSelector) {
				computesToBmhs[compName] = append(computesToBmhs[compName], bmh)
			}
		}
	}

	// Now sort computes by the number of valid matches, as sorting helps in
	// cases with large numbers of computes and/or BMHs, for a more optimal
	// backtracking

	// Make array of compute host names for use in "backtrack" func
	computeArray := make([]string, len(computes))

	i := 0
	// The keys of the "computes" map will be traversed in this for loo[] in random
	// order, but that does not matter, as we are defining the order of keys in
	// this array that we are creating and we will adhere to that from here on out
	for k := range computes {
		computeArray[i] = k
		i++
	}

	// Now we can do the actual sorting by number of preliminary matches
	sort.Slice(computeArray[:], func(i, j int) bool {
		return len(computesToBmhs[computeArray[i]]) < len(computesToBmhs[computeArray[j]])
	})

	// Finally create and use a backtracking func to find valid assignments
	assignedBMHs := map[string]bool{}                 // Keep track of assigned BMHs
	assignment := map[string]metal3v1.BareMetalHost{} // Store the final assignments

	// The backtracking function that we will use to crawl all potential
	// compute-to-BMH assignments, using the initial matching map that we
	// created earlier
	var backtrack func(index int) bool

	backtrack = func(index int) bool {
		if index == len(computes) {
			return true // All computes are assigned
		}

		comp := computeArray[index]
		for _, bmh := range computesToBmhs[comp] {
			if !assignedBMHs[bmh.Name] {
				// Assign this BMH to the compute host
				assignment[comp] = bmh
				assignedBMHs[bmh.Name] = true

				// Recur to assign the next compute
				if backtrack(index + 1) {
					return true
				}

				// If this assignment didn't work, backtrack
				delete(assignment, comp)
				assignedBMHs[bmh.Name] = false
			}
		}
		return false
	}

	// Look for matching BMHs starting with the first compute, and for each matching BMH found, recurse
	// for the next compute with the consumed BMHs removed from consideration -- backtracking down the
	// stack whenever matches are needed but exhausted for that particular path
	if backtrack(0) {
		return assignment // Return the valid assignments we have chosen
	} else {
		return nil // No valid assignments found to satisify all computes' bmhLabelSelectors
	}
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
