package openstackbaremetalset

const (
	// BmhRefInitState - (legacy, currently unused)
	BmhRefInitState = "unassigned"

	// CloudInitUserDataSecretName - Naming template used for generating BaremetalHost cloudinit userdata secrets
	CloudInitUserDataSecretName = "%s-cloudinit-userdata-%s"
	// CloudInitNetworkDataSecretName - Naming template used for generating BaremetalHost cloudinit networkdata secrets
	CloudInitNetworkDataSecretName = "%s-cloudinit-networkdata-%s"

	// HostnameLabelSelectorSuffix = Suffix used with OSBMS instance name to label BMH as belonging to an entry in OSBMS.Spec.BaremetalHosts
	HostnameLabelSelectorSuffix = "-osbms-hostname"

	// HostRemovalAnnotation - (legacy, currently unused) Annotation key placed BMH resources to target them for scale-down
	HostRemovalAnnotation = "baremetal.openstack.org/delete-host"

	// MustGatherSecret - Label placed on secrets that are safe to collect with must-gater
	MustGatherSecret = "baremetal.openstack.org/must-gather-secret"
)
