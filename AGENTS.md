# AGENTS.md - openstack-baremetal-operator

## Project overview

openstack-baremetal-operator is a Kubernetes operator that manages
bare metal provisioning infrastructure for
[OpenStack](https://docs.openstack.org/) on OpenShift/Kubernetes. It handles
provisioning physical servers (PXE boot, RHEL image serving) and matching
hardware requirements (CPU count/MHz, memory, disk size/SSD) for OpenStack
services. It is part of the
[openstack-k8s-operators](https://github.com/openstack-k8s-operators) project.

Key domain concepts: **bare metal sets** (groups of provisioned hosts),
**provision servers** (HTTP image serving), **hardware requirements**
(CPU count, CPU MHz, memory, disk size, SSD), **IPMI**, **BMC** (Baseboard
Management Controller), **Metal3** (bare metal host management).

## Tech stack

| Layer | Technology |
|-------|------------|
| Language | Go (modules, multi-module workspace via `go.work`) |
| Scaffolding | [Kubebuilder v4](https://book.kubebuilder.io/) + [Operator SDK](https://sdk.operatorframework.io/) |
| CRD generation | controller-gen (DeepCopy, CRDs, RBAC, webhooks) |
| Config management | Kustomize |
| Packaging | OLM bundle |
| Testing | Ginkgo/Gomega + envtest (functional) |
| Linting | golangci-lint (`.golangci.yaml`) |
| CI | Zuul (`zuul.d/`), Prow (`.ci-operator.yaml`), GitHub Actions |

## Custom Resources

| Kind | Purpose |
|------|---------|
| `OpenStackBaremetalSet` | Manages a set of bare metal hosts provisioned for OpenStack services. Matches hardware requirements and provisions via Metal3. |
| `OpenStackProvisionServer` | Manages an HTTP server for serving RHEL images to bare metal nodes during provisioning. |

Both CRs have defaulting and validating admission webhooks.

## Directory structure

| Directory | Contents |
|-----------|----------|
| `api/v1beta1/` | CRD types (`openstackbaremetalset_types.go`, `openstackprovisionserver_types.go`), conditions, webhook markers |
| `cmd/` | `main.go` entry point |
| `internal/controller/` | Reconcilers: `openstackbaremetalset_controller.go`, `openstackprovisionserver_controller.go` |
| `internal/openstackbaremetalset/` | BaremetalSet resource builders (bare metal host management, set helpers) |
| `internal/openstackprovisionserver/` | ProvisionServer resource builders (deployment, volumes, init containers, jobs) |
| `internal/webhook/` | Webhook implementation |
| `templates/` | Config files and scripts mounted into pods via `OPERATOR_TEMPLATES` env var. Subdirs: `openstackbaremetalset/`, `openstackprovisionserver/` |
| `config/crd,rbac,manager,webhook/` | Generated Kubernetes manifests (CRDs, RBAC, deployment, webhooks) |
| `config/samples/` | Example CRs |
| `containers/` | Container image build files (agent) |
| `tests/functional/` | envtest-based Ginkgo/Gomega tests |
| `hack/` | Helper scripts |

## Build commands

After modifying Go code, always run: `make generate manifests fmt vet`.

## Code style guidelines

- Follow standard openstack-k8s-operators conventions and lib-common patterns.
- Use `lib-common` modules for conditions, endpoints, TLS, storage, and other
  cross-cutting concerns rather than re-implementing them.
- CRD types go in `api/v1beta1/`. Controller logic goes in
  `internal/controller/`. Resource-building helpers go in
  `internal/openstackbaremetalset/` and `internal/openstackprovisionserver/`
  packages matching the CR they support.
- Config templates are plain files in `templates/` -- they are mounted at
  runtime via the `OPERATOR_TEMPLATES` environment variable.
- Webhook logic is split between the kubebuilder markers in `api/v1beta1/` and
  the implementation in `internal/webhook/`.

## Testing

- Functional tests use the envtest framework with Ginkgo/Gomega and live in
  `tests/functional/` (note: `tests/` not `test/`).
- There are no KUTTL integration tests in this operator.
- Run all functional tests: `make test`.
- When adding a new field or feature, add corresponding test cases in
  `tests/functional/`.

## Key dependencies

- [lib-common](https://github.com/openstack-k8s-operators/lib-common): shared modules for conditions, endpoints, TLS, secrets, etc.
- [Metal3 baremetal-operator](https://github.com/metal3-io/baremetal-operator): manages `BareMetalHost` CRDs for bare metal host lifecycle.
- [Cluster Baremetal Operator](https://github.com/openshift/cluster-baremetal-operator): manages the `Provisioning` CRD for Metal3 provisioning infrastructure on OpenShift.
