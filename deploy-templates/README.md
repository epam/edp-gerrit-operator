# gerrit-operator

![Version: 2.12.0-SNAPSHOT](https://img.shields.io/badge/Version-2.12.0--SNAPSHOT-informational?style=flat-square) ![Type: application](https://img.shields.io/badge/Type-application-informational?style=flat-square) ![AppVersion: 2.12.0-SNAPSHOT](https://img.shields.io/badge/AppVersion-2.12.0--SNAPSHOT-informational?style=flat-square)

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
| gerrit.basePath | string | `""` |  |
| gerrit.deploy | bool | `true` |  |
| gerrit.image | string | `"openfrontier/gerrit"` | Define gerrit docker image name |
| gerrit.imagePullPolicy | string | `"IfNotPresent"` | If defined, a imagePullPolicy applied for gerrit deployment |
| gerrit.imagePullSecrets | string | `nil` |  |
| gerrit.ingress.annotations | object | `{}` |  |
| gerrit.ingress.pathType | string | `"Prefix"` |  |
| gerrit.ingress.tls | list | `[]` |  |
| gerrit.name | string | `"gerrit"` |  |
| gerrit.nodeSelector | object | `{}` |  |
| gerrit.port | string | `"8080"` |  |
| gerrit.resources.limits.memory | string | `"2Gi"` |  |
| gerrit.resources.requests.cpu | string | `"100m"` |  |
| gerrit.resources.requests.memory | string | `"512Mi"` |  |
| gerrit.sshPort | string | `"30022"` |  |
| gerrit.storage.class | string | `"gp2"` |  |
| gerrit.storage.size | string | `"1Gi"` |  |
| gerrit.tolerations | list | `[]` |  |
| gerrit.version | string | `"3.3.2"` | Define gerrit docker image tag |
| gitServer.httpsPort | int | `443` |  |
| gitServer.name | string | `"gerrit"` |  |
| gitServer.nameSshKeySecret | string | `"gerrit-ciuser-sshkey"` |  |
| gitServer.user | string | `"jenkins"` |  |
| global.admins[0] | string | `"stub_user_one@example.com"` |  |
| global.developers[0] | string | `"stub_user_one@example.com"` |  |
| global.developers[1] | string | `"stub_user_two@example.com"` |  |
| global.dnsWildCard | string | `"example.com"` |  |
| global.edpName | string | `""` |  |
| global.openshift.deploymentType | string | `"deploymentConfigs"` |  |
| global.platform | string | `"openshift"` |  |
| image.name | string | `"epamedp/gerrit-operator"` |  |
| image.version | string | `nil` | if not defined then .Chart.AppVersion is used |
| imagePullPolicy | string | `"IfNotPresent"` |  |
| name | string | `"gerrit-operator"` |  |
| nodeSelector | object | `{}` |  |
| projectSyncInterval | string | `"1h"` |  |
| resources.limits.memory | string | `"192Mi"` |  |
| resources.requests.cpu | string | `"50m"` |  |
| resources.requests.memory | string | `"64Mi"` |  |
| tolerations | list | `[]` |  |

