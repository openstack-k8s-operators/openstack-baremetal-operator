package v1beta1

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/types"
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
		return nil, fmt.Errorf("failed to get list of all OpenStackProvisionServer(s): %s", err.Error())
	}

	for _, provServer := range provServerList.Items {
		namespacedName := types.NamespacedName{
			Namespace: provServer.Namespace,
			Name:      provServer.Name}
		found[namespacedName.String()] = provServer.Spec.Port
	}

	return found, nil
}

// AssignProvisionServerPort - Assigns an Apache listening port for a particular OpenStackProvisionServer.
func AssignProvisionServerPort(
	ctx context.Context,
	c goClient.Client,
	instance *OpenStackProvisionServer,
	portStart int32,
	portEnd int32,
) error {
	existingPorts, err := GetExistingProvServerPorts(ctx, c, instance)
	if err != nil {
		return err
	}

	namespacedName := types.NamespacedName{
		Namespace: instance.Namespace,
		Name:      instance.Name}
	// It's possible that this prov server already exists and we are just dealing with
	// a minimized version of it (only its ObjectMeta is set, etc)
	cur := existingPorts[namespacedName.String()]
	if cur == 0 {
		cur = portStart
	}

	for ; ; cur++ {
		if cur > portEnd {
			return fmt.Errorf("selected port is out of range %v-%v-%v", cur, portStart, portEnd)
		}
		found := false
		for _, port := range existingPorts {
			if port == cur {
				found = true
				break
			}
		}

		if found {
			if existingPorts[namespacedName.String()] != cur {
				// continue to use the next port in the port range.
				continue
			} else {
				break
			}
		}

		if !found {
			break
		}

	}
	instance.Spec.Port = cur
	return nil
}
