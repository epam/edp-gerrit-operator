<a name="unreleased"></a>
## [Unreleased]


<a name="v2.21.0"></a>
## [v2.21.0] - 2024-06-12
### Features

- Enable gerrit restart flow ([#47](https://github.com/epam/edp-gerrit-operator/issues/47))
- Add the ability to remove the files with GerritMergeRequest CR ([#30](https://github.com/epam/edp-gerrit-operator/issues/30))

### Routine

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


<a name="v2.20.1"></a>
## [v2.20.1] - 2024-03-12
### Routine

- Bump alpine version ([#27](https://github.com/epam/edp-gerrit-operator/issues/27))


<a name="v2.20.0"></a>
## [v2.20.0] - 2024-03-11
### Features

- Align to the Tekton EventListener endpoint ([#24](https://github.com/epam/edp-gerrit-operator/issues/24))
- Add QuickLink Custom Resource ([#22](https://github.com/epam/edp-gerrit-operator/issues/22))

### Bug Fixes

- Update webhook url ([#21](https://github.com/epam/edp-gerrit-operator/issues/21))

### Code Refactoring

- QuickLink is managed by edp-tekton helm chart ([#24](https://github.com/epam/edp-gerrit-operator/issues/24))

### Routine

- Remove gitserver CR ([#168](https://github.com/epam/edp-gerrit-operator/issues/168))
- Remove edpcomponent creation from operator logic ([#23](https://github.com/epam/edp-gerrit-operator/issues/23))
- Bump github.com/go-git/go-git/v5 from 5.5.1 to 5.11.0 ([#19](https://github.com/epam/edp-gerrit-operator/issues/19))
- Bump github.com/cloudflare/circl from 1.3.3 to 1.3.7 ([#20](https://github.com/epam/edp-gerrit-operator/issues/20))
- Bump golang.org/x/crypto from 0.14.0 to 0.17.0 ([#18](https://github.com/epam/edp-gerrit-operator/issues/18))
- Update current development version ([#17](https://github.com/epam/edp-gerrit-operator/issues/17))

### Documentation

- Define name convention for ingress objects ([#23](https://github.com/epam/edp-gerrit-operator/issues/23))
- Update README md and Dockerfile ([#132](https://github.com/epam/edp-gerrit-operator/issues/132))


<a name="v2.19.0"></a>
## [v2.19.0] - 2023-12-18
### Bug Fixes

- Generate ChangeIDs using UUID ([#16](https://github.com/epam/edp-gerrit-operator/issues/16))

### Routine

- Update openssl package for operator container ([#16](https://github.com/epam/edp-gerrit-operator/issues/16))
- Update release flow for GH Actions ([#16](https://github.com/epam/edp-gerrit-operator/issues/16))
- Update current development version ([#15](https://github.com/epam/edp-gerrit-operator/issues/15))


<a name="v2.18.0"></a>
## [v2.18.0] - 2023-11-03
### Features

- Add label to the secret gerrit-ciuser-sshkey ([#14](https://github.com/epam/edp-gerrit-operator/issues/14))

### Routine

- Fix branch for GH Actions ([#8](https://github.com/epam/edp-gerrit-operator/issues/8))
- Bump golang.org/x/net from 0.8.0 to 0.17.0 ([#13](https://github.com/epam/edp-gerrit-operator/issues/13))
- Update changelog ([#11](https://github.com/epam/edp-gerrit-operator/issues/11))
- Remove jenkins admin-console perf operator logic ([#10](https://github.com/epam/edp-gerrit-operator/issues/10))
- Upgrade Go to 1.20 ([#8](https://github.com/epam/edp-gerrit-operator/issues/8))
- Update current development version ([#7](https://github.com/epam/edp-gerrit-operator/issues/7))

### Documentation

- Fix bage path for the build GH Action ([#8](https://github.com/epam/edp-gerrit-operator/issues/8))


<a name="v2.17.1"></a>
## [v2.17.1] - 2023-09-25
### Routine

- Upgrade Go to 1.20 ([#8](https://github.com/epam/edp-gerrit-operator/issues/8))
- Update CHANGELOG.md ([#85](https://github.com/epam/edp-gerrit-operator/issues/85))


<a name="v2.17.0"></a>
## [v2.17.0] - 2023-09-20
### Bug Fixes

- Use crypto rand to generate secure ED25519 private key check fields ([#5](https://github.com/epam/edp-gerrit-operator/issues/5))

### Code Refactoring

- Remove deprecated edpName parameter ([#6](https://github.com/epam/edp-gerrit-operator/issues/6))

### Routine

- Update current development version ([#4](https://github.com/epam/edp-gerrit-operator/issues/4))


<a name="v2.16.0"></a>
## [v2.16.0] - 2023-08-17

[Unreleased]: https://github.com/epam/edp-gerrit-operator/compare/v2.21.0...HEAD
[v2.21.0]: https://github.com/epam/edp-gerrit-operator/compare/v2.20.1...v2.21.0
[v2.20.1]: https://github.com/epam/edp-gerrit-operator/compare/v2.20.0...v2.20.1
[v2.20.0]: https://github.com/epam/edp-gerrit-operator/compare/v2.19.0...v2.20.0
[v2.19.0]: https://github.com/epam/edp-gerrit-operator/compare/v2.18.0...v2.19.0
[v2.18.0]: https://github.com/epam/edp-gerrit-operator/compare/v2.17.1...v2.18.0
[v2.17.1]: https://github.com/epam/edp-gerrit-operator/compare/v2.17.0...v2.17.1
[v2.17.0]: https://github.com/epam/edp-gerrit-operator/compare/v2.16.0...v2.17.0
[v2.16.0]: https://github.com/epam/edp-gerrit-operator/compare/v2.15.0...v2.16.0
