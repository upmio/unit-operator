# Unit Operator Helm Chart

This Helm chart installs the [Unit Operator](https://github.com/upmio/unit-operator) into your Kubernetes (v1.27+) or Openshift cluster (v4.6+).

## TL;DR


```sh
# Add the repo to helm (typically use a tag rather than main):
helm repo add upm-charts https://upmio.github.io/helm-charts

# Install the operator
helm install unit-operator --namespace upm-system --create-namespace \
  --set crd.enabled=true \
  upm-charts/unit-operator
```

## Introduction

The Unit operator is installed into the `upm-system` namespace for Kubernetes clusters.

The Unit and UnitSet Custom Resource Definitions (CRDs) can either be installed manually (the recommended approach, part of the Helm chart (`crd.enabled=true`).
Installing the CRDs as part of the Helm chart is not recommended for production setups, since uninstalling the Helm chart will also uninstall the CRDs and subsequently delete any remaining CRs.
The CRDs allow you to configure individual parts of your Unit setup:

* [`Unit`](https://github.com/upmio/unit-operator/blob/dev/doc/unit-operator-api.md#unit)
* [`UnitSet`](https://github.com/upmio/unit-operator/blob/dev/doc/unit-operator-api.md#unitset)

After the installation of the Unit-operator chart, you can start inject the Custom Resources (CRs) into your cluster.
The Unit operator will then automatically start installing the components.
Please see the documentation of each CR for details.

## Uninstalling

Before removing the Unit operator from your cluster, you should first make sure that there are no instances of resources managed by the operator left:

```sh
kubectl get Unit,UnitSet --all-namespaces
```

Now you can use Helm to uninstall the Unit operator:

```sh
# for Kubernetes:
helm uninstall --namespace upm-system unit-operator --wait

# optionally remove repository from helm:
helm repo remove upm-charts
```

**Important:** if you installed the CRDs with the Helm chart (by setting `crd.enabled=true`), the CRDs will be removed as well: this means any remaining Unit resources (e.g. Unit Pipelines) in the cluster will be deleted!

If you installed the CRDs manually, you can use the following command to remove them (*this will remove all Unit resources from your cluster*):
```
kubectl delete crd Unit,UnitSet --ignore-not-found
```
