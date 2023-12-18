# gerrit-operator

![Version: 2.20.0-SNAPSHOT](https://img.shields.io/badge/Version-2.20.0--SNAPSHOT-informational?style=flat-square) ![Type: application](https://img.shields.io/badge/Type-application-informational?style=flat-square) ![AppVersion: 2.20.0-SNAPSHOT](https://img.shields.io/badge/AppVersion-2.20.0--SNAPSHOT-informational?style=flat-square)

A Helm chart for EDP Gerrit Operator

**Homepage:** <https://epam.github.io/edp-install/>

## Maintainers

| Name | Email | Url |
| ---- | ------ | --- |
| epmd-edp | <SupportEPMD-EDP@epam.com> | <https://solutionshub.epam.com/solution/epam-delivery-platform> |
| sergk |  | <https://github.com/SergK> |

## Source Code

* <https://github.com/epam/edp-gerrit-operator>

## Values

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| affinity | object | `{}` |  |
| annotations | object | `{}` |  |
| gerrit.affinity | object | `{}` |  |
| gerrit.annotations | object | `{}` |  |
| gerrit.basePath | string | `""` | Base path for Nexus URL |
| gerrit.caCerts.enabled | bool | `false` | Flag for enabling additional CA certificates |
| gerrit.caCerts.image | string | `"adoptopenjdk/openjdk11:alpine"` | Change init CA certificates container image |
| gerrit.caCerts.secret | string | `"secret-name"` | Name of the secret containing additional CA certificates |
| gerrit.deploy | bool | `true` | Flag to enable/disable Gerrit deploy |
| gerrit.image | string | `"epamedp/edp-gerrit"` | Define gerrit docker image name |
| gerrit.imagePullPolicy | string | `"IfNotPresent"` | If defined, a imagePullPolicy applied for gerrit deployment |
| gerrit.imagePullSecrets | string | `nil` | Secrets to pull from private Docker registry; |
| gerrit.ingress.annotations | object | `{}` |  |
| gerrit.ingress.pathType | string | `"Prefix"` | pathType is only for k8s >= 1.1= |
| gerrit.ingress.tls | list | `[]` | See https://kubernetes.io/blog/2020/04/02/improvements-to-the-ingress-api-in-kubernetes-1.18/#specifying-the-class-of-an-ingress ingressClassName: nginx |
| gerrit.javaOptions | string | `""` | Values to add to JAVA_OPTIONS |
| gerrit.name | string | `"gerrit"` | Gerrit name |
| gerrit.nodeSelector | object | `{}` |  |
| gerrit.port | string | `"8080"` | HTTP port |
| gerrit.resources.limits.memory | string | `"2Gi"` |  |
| gerrit.resources.requests.cpu | string | `"100m"` |  |
| gerrit.resources.requests.memory | string | `"512Mi"` |  |
| gerrit.storage.size | string | `"1Gi"` | Size for Gerrit data volume |
| gerrit.tolerations | list | `[]` |  |
| gerrit.version | string | `"3.6.2"` | Define gerrit docker image tag |
| gitServer.httpsPort | int | `443` | HTTPS port |
| gitServer.name | string | `"gerrit"` | GitServer CR name |
| gitServer.nameSshKeySecret | string | `"gerrit-ciuser-sshkey"` | Name of secret with credentials to Git server |
| gitServer.user | string | `"edp-ci"` | Git user to connect |
| global.admins | list | `["stub_user_one@example.com"]` | Administrators of your tenant |
| global.developers | list | `["stub_user_one@example.com","stub_user_two@example.com"]` | Developers of your tenant |
| global.dnsWildCard | string | `nil` | a cluster DNS wildcard name |
| global.gerritSSHPort | string | `"30022"` | Gerrit SSH node port |
| global.openshift.deploymentType | string | `"deployments"` | Which type of kind will be deployed to Openshift (values: deployments/deploymentConfigs) |
| global.platform | string | `"openshift"` | platform type that can be "kubernetes" or "openshift" |
| groupMemberSyncInterval | string | `"30m"` | If not defined the exponential formula with the max value of 1hr will be used |
| image.repository | string | `"epamedp/gerrit-operator"` | EDP gerrit-operator Docker image name. The released image can be found on [Dockerhub](https://hub.docker.com/r/epamedp/gerrit-operator) |
| image.tag | string | `nil` | EDP gerrit-operator Docker image tag. The released image can be found on [Dockerhub](https://hub.docker.com/r/epamedp/gerrit-operator/tags) |
| imagePullPolicy | string | `"IfNotPresent"` |  |
| name | string | `"gerrit-operator"` | component name |
| nodeSelector | object | `{}` |  |
| projectSyncInterval | string | `"1h"` | Format: golang time.Duration-formatted string |
| resources.limits.memory | string | `"192Mi"` |  |
| resources.requests.cpu | string | `"50m"` |  |
| resources.requests.memory | string | `"64Mi"` |  |
| tolerations | list | `[]` |  |

