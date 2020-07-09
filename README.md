# How to Install Operator

EDP installation can be applied on two container orchestration platforms: OpenShift and Kubernetes.

_**NOTE:** Installation of operators is platform-independent, that is why there is a unified instruction for deploying._

## Prerequisites
1. Linux machine or Windows Subsystem for Linux instance with [Helm 3](https://helm.sh/docs/intro/install/) installed;
2. Cluster admin access to the cluster;
3. EDP project/namespace is deployed by following one of the instructions: [edp-install-openshift](https://github.com/epmd-edp/edp-install/blob/master/documentation/openshift_install_edp.md#edp-project) or [edp-install-kubernetes](https://github.com/epmd-edp/edp-install/blob/master/documentation/kubernetes_install_edp.md#edp-namespace).

## Installation
In order to install the EDP Gerrit-operator, follow the steps below:

1. To add the Helm EPAMEDP Charts for local client, run "helm repo add":
     ```bash
     helm repo add epamedp https://chartmuseum.demo.edp-epam.com/
     ```
2. Choose available Helm chart version:
    ```bash
     helm search repo epamedp/gerrit-operator
    ```
   Example response:
   ```
     NAME                     CHART VERSION   APP VERSION     DESCRIPTION
     epamedp/gerrit-operator  v2.4.0                          Helm chart for Golang application/service deplo...
     ```

    _**NOTE:** It is highly recommended to use the latest released version._
3. Deploy operator:

Full available chart parameters list:
```
    - <chart_version>                        # Helm chart version;
    - global.edpName                         # a namespace or a project name (in case of OpenShift);
    - global.platform                        # a platform type that can be "kubernetes" or "openshift";
    - global.dnsWildCard                     # a cluster DNS wildcard name;
    - global.admins                          # Administrators of your tenant separated by comma (,) (eg --set 'global.admins={test@example.com}');
    - image.name                             # EDP image. The released image can be found on [Dockerhub](https://hub.docker.com/repository/docker/epamedp/gerrit-operator);
    - image.version                          # EDP tag. The released image can be found on [Dockerhub](https://hub.docker.com/repository/docker/epamedp/gerrit-operator/tags);
    - gerrit.deploy                          # Flag to enable/disable Gerrit deploy;
    - gerrit.name                            # Gerrit name;
    - gerrit.image                           # Gerrit image, e.g. openfrontier/gerrit;
    - gerrit.version                         # Gerrit version, e.g. 3.1.4;
    - gerrit.sshPort                         # SSH port;
    - gitServer.name                         # GitServer CR name;
    - gitServer.user                         # Git user to connect;
    - gitServer.httpsPort                    # HTTPS port;
    - gitServer.nameSshKeySecret             # Name of secret with credentials to Git server;
    - gitServer.sshPort                      # SSH port;
```

_**NOTE:** Follow instruction to create namespace [edp-install-openshift](https://github.com/epmd-edp/edp-install/blob/master/documentation/openshift_install_edp.md#install-edp) or [edp-install-kubernetes](https://github.com/epmd-edp/edp-install/blob/master/documentation/kubernetes_install_edp.md#install-edp)._

Inspect the sample of launching a Helm template for Nexus operator installation:
```bash
helm install gerrit-operator epamedp/gerrit-operator --version <chart_version> --namespace <edp_cicd_project> --set name=gerrit-operator --set global.edpName=<edp_cicd_project> --set global.platform
```

* Check the <edp_cicd_project> namespace that should contain Deployment with your operator in a running status

## Local Development
In order to develop the operator, first set up a local environment. For details, please refer to the [Local Development](documentation/local-development.md) page.

### Related Articles
- [Gerrit Operator Architecture Scheme](documentation/arch.md)
- [Replicate Gerrit to GitLab](documentation/replicate_gerrit_to_gitlab.md)