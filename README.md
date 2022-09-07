[![Build](https://github.com/epam/edp-gerrit-operator/.github/workflows/build.yaml/badge.svg)](https://github.com/epam/edp-gerrit-operator/.github/workflows/build.yaml)
[![codecov](https://codecov.io/gh/epam/edp-gerrit-operator/branch/master/graph/badge.svg?token=8JOEVZL3VL)](https://codecov.io/gh/epam/edp-gerrit-operator)
[![Go Report Card](https://goreportcard.com/badge/github.com/epam/edp-gerrit-operator/v2)](https://goreportcard.com/report/github.com/epam/edp-gerrit-operator/v2)
[![GitHub license](https://img.shields.io/github/license/epam/edp-gerrit-operator)](https://github.com/epam/edp-gerrit-operator/blob/master/LICENSE-2.0)

# Gerrit Operator

| :heavy_exclamation_mark: Please refer to [EDP documentation](https://epam.github.io/edp-install/) to get the notion of the main concepts and guidelines. |
| --- |

Get acquainted with the Gerrit Operator and the installation process as well as the local development, and architecture scheme.

## Overview

Gerrit Operator is an EDP operator that is responsible for installing and configuring Gerrit. Operator installation can be applied on OpenShift container orchestration platform.

_**NOTE:** Operator is platform-independent, that is why there is a unified instruction for deploying._

## Prerequisites

1. Linux machine or Windows Subsystem for Linux instance with [Helm 3](https://helm.sh/docs/intro/install/) installed;
2. Cluster admin access to the cluster;
3. EDP project/namespace is deployed by following the [Install EDP](https://epam.github.io/edp-install/operator-guide/install-edp/) instruction.
4. Make sure Git [`FSMonitor`](https://www.git-scm.com/docs/git-fsmonitor--daemon) feature is turned off. This is due to [limitations](https://github.com/go-git/go-git/issues/299) of `go-git`.

## Installation

In order to install the EDP Gerrit Operator, follow the steps below:

1. To add the Helm EPAMEDP Charts for local client, run "helm repo add":
     ```bash
     helm repo add epamedp https://epam.github.io/edp-helm-charts/stable
     ```
2. Choose available Helm chart version:
     ```bash
     helm search repo epamedp/gerrit-operator -l
     NAME                     CHART VERSION   APP VERSION     DESCRIPTION
     epamedp/gerrit-operator  2.11.0          2.11.0          A Helm chart for EDP Gerrit Operator
     epamedp/gerrit-operator  2.10.0          2.10.0          A Helm chart for EDP Gerrit Operator
     ```

    _**NOTE:** It is highly recommended to use the latest released version._

3. Full chart parameters available in [deploy-templates/README.md](deploy-templates/README.md).

4. Install operator in the <edp-project> namespace with the helm command; find below the installation command example:
    ```bash
    helm install gerrit-operator epamedp/gerrit-operator --version <chart_version> --namespace <edp-project> --set name=gerrit-operator --set global.edpName=<edp-project> --set global.platform=<platform_type> --set global.dnsWildCard=<cluster_DNS_wildcard>
    ```
5. Check the <edp-project> namespace that should contain Deployment with your operator in a running status.

## Local Development

In order to develop the operator, first set up a local environment. For details, please refer to the [Local Development](https://epam.github.io/edp-install/developer-guide/local-development/) page.

Development versions are also available, please refer to the [snapshot helm chart repository](https://epam.github.io/edp-helm-charts/snapshot/) page.

### Related Articles

- [Architecture Scheme of Gerrit Operator](documentation/arch.md)
- [Install EDP](https://epam.github.io/edp-install/operator-guide/install-edp/)
