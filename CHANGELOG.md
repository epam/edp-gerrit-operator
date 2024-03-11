<a name="unreleased"></a>
## [Unreleased]


<a name="v2.20.0"></a>
## v2.20.0 - 2024-02-29
### Features

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


[Unreleased]: https://github.com/epam/edp-gerrit-operator/compare/v2.20.0...HEAD
