package openstackprovisionserver

import "errors"

var (
	// ErrProvisioningAgent is returned when the provisioning agent reports an error
	ErrProvisioningAgent = errors.New("provisioning agent reported error")
)
