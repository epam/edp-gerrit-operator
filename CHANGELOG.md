<a name="unreleased"></a>
## [Unreleased]

### Features

- Implement Developers group creation in go code [EPMDEDP-7502](https://jiraeu.epam.com/browse/EPMDEDP-7502)
- Manage Gerrit access [EPMDEDP-7502](https://jiraeu.epam.com/browse/EPMDEDP-7502)
- add git executable to docker image [EPMDEDP-8162](https://jiraeu.epam.com/browse/EPMDEDP-8162)
- implement gerrit merge request CR [EPMDEDP-8162](https://jiraeu.epam.com/browse/EPMDEDP-8162)
- Update Makefile changelog target [EPMDEDP-8218](https://jiraeu.epam.com/browse/EPMDEDP-8218)
- add status tracking to gerrit merge request [EPMDEDP-8222](https://jiraeu.epam.com/browse/EPMDEDP-8222)
- make time interval configurable. [EPMDEDP-8244](https://jiraeu.epam.com/browse/EPMDEDP-8244)
- Allow to re-define project sync time [EPMDEDP-8244](https://jiraeu.epam.com/browse/EPMDEDP-8244)
- cmd args for git merge [EPMDEDP-8305](https://jiraeu.epam.com/browse/EPMDEDP-8305)
- add .golangci-lint config [EPMDEDP-8343](https://jiraeu.epam.com/browse/EPMDEDP-8343)
- Add ingress tls certificate option when using ingress controller [EPMDEDP-8377](https://jiraeu.epam.com/browse/EPMDEDP-8377)
- Generate CRDs and helm docs automatically [EPMDEDP-8385](https://jiraeu.epam.com/browse/EPMDEDP-8385)

### Bug Fixes

- add author name email to merge request CRD [EPMDEDP-8162](https://jiraeu.epam.com/browse/EPMDEDP-8162)
- Fix changelog generation in GH Release Action [EPMDEDP-8468](https://jiraeu.epam.com/browse/EPMDEDP-8468)

### Code Refactoring

- remove legacy fields in Gerrit CR spec. [EPMDEDP-7536](https://jiraeu.epam.com/browse/EPMDEDP-7536)

### Testing

- Add tests and mocks [EPMDEDP-7992](https://jiraeu.epam.com/browse/EPMDEDP-7992)
- Add tests [EPMDEDP-7992](https://jiraeu.epam.com/browse/EPMDEDP-7992)
- Add tests and mocks [EPMDEDP-7992](https://jiraeu.epam.com/browse/EPMDEDP-7992)
- Fix unit tests [EPMDEDP-7992](https://jiraeu.epam.com/browse/EPMDEDP-7992)
- add tests for git and gerrit client [EPMDEDP-8162](https://jiraeu.epam.com/browse/EPMDEDP-8162)
- fix running in cluster function [EPMDEDP-8222](https://jiraeu.epam.com/browse/EPMDEDP-8222)
- fix gerrit service tests [EPMDEDP-8222](https://jiraeu.epam.com/browse/EPMDEDP-8222)

### Routine

- Update release CI pipelines [EPMDEDP-7847](https://jiraeu.epam.com/browse/EPMDEDP-7847)
- Fix CI for codecov report [EPMDEDP-7992](https://jiraeu.epam.com/browse/EPMDEDP-7992)
- Populate chart with Artifacthub annotations [EPMDEDP-8049](https://jiraeu.epam.com/browse/EPMDEDP-8049)
- Update gerrit URL baseline link [EPMDEDP-8204](https://jiraeu.epam.com/browse/EPMDEDP-8204)
- Update changelog [EPMDEDP-8227](https://jiraeu.epam.com/browse/EPMDEDP-8227)
- Update base docker image to alpine 3.15.4 [EPMDEDP-8853](https://jiraeu.epam.com/browse/EPMDEDP-8853)


<a name="v2.10.0"></a>
## [v2.10.0] - 2021-12-06
### Features

- Provide operator's build information [EPMDEDP-7847](https://jiraeu.epam.com/browse/EPMDEDP-7847)

### Bug Fixes

- Changelog links [EPMDEDP-7847](https://jiraeu.epam.com/browse/EPMDEDP-7847)

### Code Refactoring

- Expand gerrit-operator role [EPMDEDP-7279](https://jiraeu.epam.com/browse/EPMDEDP-7279)
- Add namespace field in roleRef in OKD RB, aling CRB name [EPMDEDP-7279](https://jiraeu.epam.com/browse/EPMDEDP-7279)
- Replace cluster-wide role/rolebinding to namespaced [EPMDEDP-7279](https://jiraeu.epam.com/browse/EPMDEDP-7279)
- Remove Sonar-Verified label [EPMDEDP-7799](https://jiraeu.epam.com/browse/EPMDEDP-7799)
- Address golangci-lint issues [EPMDEDP-7945](https://jiraeu.epam.com/browse/EPMDEDP-7945)

### Formatting

- go fmt. Remove unnecessary spaces [EPMDEDP-7943](https://jiraeu.epam.com/browse/EPMDEDP-7943)

### Testing

- remove TestRunningInCluster [EPMDEDP-7885](https://jiraeu.epam.com/browse/EPMDEDP-7885)

### Routine

- Update openssh-client version [EPMDEDP-7469](https://jiraeu.epam.com/browse/EPMDEDP-7469)
- Add changelog generator [EPMDEDP-7847](https://jiraeu.epam.com/browse/EPMDEDP-7847)
- Add codecov report [EPMDEDP-7885](https://jiraeu.epam.com/browse/EPMDEDP-7885)
- Update docker image [EPMDEDP-7895](https://jiraeu.epam.com/browse/EPMDEDP-7895)
- Upgrade edp components [EPMDEDP-7930](https://jiraeu.epam.com/browse/EPMDEDP-7930)
- Use custom go build step for operator [EPMDEDP-7932](https://jiraeu.epam.com/browse/EPMDEDP-7932)
- Update go to version 1.17 [EPMDEDP-7932](https://jiraeu.epam.com/browse/EPMDEDP-7932)

### Documentation

- Update the links on GitHub [EPMDEDP-7781](https://jiraeu.epam.com/browse/EPMDEDP-7781)


<a name="v2.9.0"></a>
## [v2.9.0] - 2021-12-03

<a name="v2.8.0"></a>
## [v2.8.0] - 2021-12-03

<a name="v2.7.2"></a>
## [v2.7.2] - 2021-12-03

<a name="v2.7.1"></a>
## [v2.7.1] - 2021-12-03

<a name="v2.7.0"></a>
## [v2.7.0] - 2021-12-03

[Unreleased]: https://github.com/epam/edp-gerrit-operator/compare/v2.10.0...HEAD
[v2.10.0]: https://github.com/epam/edp-gerrit-operator/compare/v2.9.0...v2.10.0
[v2.9.0]: https://github.com/epam/edp-gerrit-operator/compare/v2.8.0...v2.9.0
[v2.8.0]: https://github.com/epam/edp-gerrit-operator/compare/v2.7.2...v2.8.0
[v2.7.2]: https://github.com/epam/edp-gerrit-operator/compare/v2.7.1...v2.7.2
[v2.7.1]: https://github.com/epam/edp-gerrit-operator/compare/v2.7.0...v2.7.1
[v2.7.0]: https://github.com/epam/edp-gerrit-operator/compare/v2.3.0-73...v2.7.0
