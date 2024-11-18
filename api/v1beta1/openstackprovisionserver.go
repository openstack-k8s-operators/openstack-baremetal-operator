package v1beta1

import (
	"context"
	"fmt"

	goClient "sigs.k8s.io/controller-runtime/pkg/client"
)

// GetExistingProvServerPorts - Get all ports currently in use by all OpenStackProvisionServers
func GetExistingProvServerPorts(
	ctx context.Context,
	c goClient.Client,
	instance *OpenStackProvisionServer,
) (map[string]int32, error) {
	found := map[string]int32{}

	provServerList := &OpenStackProvisionServerList{}

	listOpts := []goClient.ListOption{}

	err := c.List(ctx, provServerList, listOpts...)
	if err != nil {
		return nil, fmt.Errorf("Failed to get list of all OpenStackProvisionServer(s): %s", err.Error())
	}

	for _, provServer := range provServerList.Items {
		found[provServer.Name] = provServer.Spec.Port
	}

	return found, nil
}

// AssignProvisionServerPort - Assigns an Apache listening port for a particular OpenStackProvisionServer.
func AssignProvisionServerPort(
	ctx context.Context,
	c goClient.Client,
	instance *OpenStackProvisionServer,
	portStart int32,
) error {
	existingPorts, err := GetExistingProvServerPorts(ctx, c, instance)
	if err != nil {
		return err
	}

	// It's possible that this prov server already exists and we are just dealing with
	// a minimized version of it (only its ObjectMeta is set, etc)
	instance.Spec.Port = existingPorts[instance.GetName()]

	// If we get this far, no port has been previously assigned, so we pick one
	if instance.Spec.Port == 0 {
		cur := portStart

		for {
			found := false

			for _, port := range existingPorts {
				if port == cur {
					found = true
					break
				}
			}

			if !found {
				break
			}

			cur++
		}

		instance.Spec.Port = cur
	}

	return nil
}
