<a name="unreleased"></a>
## [Unreleased]


<a name="v2.23.0"></a>
## v2.23.0 - 2025-01-23
### Features

- Update Dockerfile packages([#59](https://github.com/epam/edp-gerrit-operator/issues/59))
- Migrate sso resources from edp-install to gerrit-operator([#59](https://github.com/epam/edp-gerrit-operator/issues/59))
- Remove deprecated v1alpha1 versions from the operator ([#54](https://github.com/epam/edp-gerrit-operator/issues/54))
- Enable gerrit restart flow ([#47](https://github.com/epam/edp-gerrit-operator/issues/47))
- Add the ability to remove the files with GerritMergeRequest CR ([#30](https://github.com/epam/edp-gerrit-operator/issues/30))
- Align to the Tekton EventListener endpoint ([#24](https://github.com/epam/edp-gerrit-operator/issues/24))
- Add QuickLink Custom Resource ([#22](https://github.com/epam/edp-gerrit-operator/issues/22))
- Add label to the secret gerrit-ciuser-sshkey ([#14](https://github.com/epam/edp-gerrit-operator/issues/14))

### Bug Fixes

- Update webhook url ([#21](https://github.com/epam/edp-gerrit-operator/issues/21))
- Generate ChangeIDs using UUID ([#16](https://github.com/epam/edp-gerrit-operator/issues/16))
- Use crypto rand to generate secure ED25519 private key check fields ([#5](https://github.com/epam/edp-gerrit-operator/issues/5))

### Code Refactoring

- QuickLink is managed by edp-tekton helm chart ([#24](https://github.com/epam/edp-gerrit-operator/issues/24))
- Remove deprecated edpName parameter ([#6](https://github.com/epam/edp-gerrit-operator/issues/6))

### Routine

- Enable QuickLink resource by default ([#71](https://github.com/epam/edp-gerrit-operator/issues/71))
- Make QuickLink installation optional ([#71](https://github.com/epam/edp-gerrit-operator/issues/71))
- Update Dockerfile packages ([#66](https://github.com/epam/edp-gerrit-operator/issues/66))
- Update Pull Request Template ([#66](https://github.com/epam/edp-gerrit-operator/issues/66))
- Update current development version ([#64](https://github.com/epam/edp-gerrit-operator/issues/64))
- Update alpine base image to v3.18.9 ([#62](https://github.com/epam/edp-gerrit-operator/issues/62))
- Update KubeRocketCI names and documentation links ([#57](https://github.com/epam/edp-gerrit-operator/issues/57))
- Update container image ([#54](https://github.com/epam/edp-gerrit-operator/issues/54))
- Update current development version ([#52](https://github.com/epam/edp-gerrit-operator/issues/52))
- Bump alpine packages ([#49](https://github.com/epam/edp-gerrit-operator/issues/49))
- Update argocd diff ([#49](https://github.com/epam/edp-gerrit-operator/issues/49))
- Align argocd diff ([#42](https://github.com/epam/edp-gerrit-operator/issues/42))
- Switch to private Keycloak client ([#42](https://github.com/epam/edp-gerrit-operator/issues/42))
- Enable Keycloak client creation ([#42](https://github.com/epam/edp-gerrit-operator/issues/42))
- Disable keycloak configuration by default ([#42](https://github.com/epam/edp-gerrit-operator/issues/42))
- Update oauth gerrit plugin instead of gerrit version ([#39](https://github.com/epam/edp-gerrit-operator/issues/39))
- Bump gerrit image version to 3.7.9 ([#39](https://github.com/epam/edp-gerrit-operator/issues/39))
- Bump to Go 1.22 ([#37](https://github.com/epam/edp-gerrit-operator/issues/37))
- Add codeowners file to the repo ([#31](https://github.com/epam/edp-gerrit-operator/issues/31))
- Describe GerritMergeRequest CR spec ([#28](https://github.com/epam/edp-gerrit-operator/issues/28))
- Update current development version ([#26](https://github.com/epam/edp-gerrit-operator/issues/26))
- Bump alpine version ([#27](https://github.com/epam/edp-gerrit-operator/issues/27))
- Update current development version ([#26](https://github.com/epam/edp-gerrit-operator/issues/26))
- Remove gitserver CR ([#168](https://github.com/epam/edp-gerrit-operator/issues/168))
- Remove edpcomponent creation from operator logic ([#23](https://github.com/epam/edp-gerrit-operator/issues/23))
- Bump github.com/go-git/go-git/v5 from 5.5.1 to 5.11.0 ([#19](https://github.com/epam/edp-gerrit-operator/issues/19))
- Bump github.com/cloudflare/circl from 1.3.3 to 1.3.7 ([#20](https://github.com/epam/edp-gerrit-operator/issues/20))
- Bump golang.org/x/crypto from 0.14.0 to 0.17.0 ([#18](https://github.com/epam/edp-gerrit-operator/issues/18))
- Update current development version ([#17](https://github.com/epam/edp-gerrit-operator/issues/17))
- Update openssl package for operator container ([#16](https://github.com/epam/edp-gerrit-operator/issues/16))
- Update release flow for GH Actions ([#16](https://github.com/epam/edp-gerrit-operator/issues/16))
- Update current development version ([#15](https://github.com/epam/edp-gerrit-operator/issues/15))
- Fix branch for GH Actions ([#8](https://github.com/epam/edp-gerrit-operator/issues/8))
- Bump golang.org/x/net from 0.8.0 to 0.17.0 ([#13](https://github.com/epam/edp-gerrit-operator/issues/13))
- Update changelog ([#11](https://github.com/epam/edp-gerrit-operator/issues/11))
- Remove jenkins admin-console perf operator logic ([#10](https://github.com/epam/edp-gerrit-operator/issues/10))
- Upgrade Go to 1.20 ([#8](https://github.com/epam/edp-gerrit-operator/issues/8))
- Update current development version ([#7](https://github.com/epam/edp-gerrit-operator/issues/7))
- Update current development version ([#4](https://github.com/epam/edp-gerrit-operator/issues/4))

### Documentation

- Define name convention for ingress objects ([#23](https://github.com/epam/edp-gerrit-operator/issues/23))
- Update README md and Dockerfile ([#132](https://github.com/epam/edp-gerrit-operator/issues/132))
- Fix bage path for the build GH Action ([#8](https://github.com/epam/edp-gerrit-operator/issues/8))


[Unreleased]: https://github.com/epam/edp-gerrit-operator/compare/v2.23.0...HEAD
