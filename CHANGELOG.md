<a name="unreleased"></a>
## [Unreleased]


<a name="v2.16.0"></a>
## [v2.16.0] - 2023-08-15
### Features

- Add optional external url to spec [EPMDEDP-11854](https://jiraeu.epam.com/browse/EPMDEDP-11854)

### Bug Fixes

- Wsl linter errors [EPMDEDP-12466](https://jiraeu.epam.com/browse/EPMDEDP-12466)

### Routine

- Bump github.com/cloudflare/circl from 1.1.0 to 1.3.3 [EPMDEDP-00000](https://jiraeu.epam.com/browse/EPMDEDP-00000)
- Update current development version [EPMDEDP-11826](https://jiraeu.epam.com/browse/EPMDEDP-11826)
- Bump alpine docker image to 3.18.2 [EPMDEDP-12253](https://jiraeu.epam.com/browse/EPMDEDP-12253)
- Bump alpine docker image to 3.18.3 [EPMDEDP-12461](https://jiraeu.epam.com/browse/EPMDEDP-12461)


<a name="v2.15.0"></a>
## [v2.15.0] - 2023-05-25
### Features

- Move EDP Component creation to deploy-templates [EPMDEDP-11726](https://jiraeu.epam.com/browse/EPMDEDP-11726)
- Add commentadded event to the webhook [EPMDEDP-11899](https://jiraeu.epam.com/browse/EPMDEDP-11899)

### Bug Fixes

- Remove project-creator user [EPMDEDP-11871](https://jiraeu.epam.com/browse/EPMDEDP-11871)
- Exclude Gerrit configs from test coverage [EPMDEDP-11899](https://jiraeu.epam.com/browse/EPMDEDP-11899)

### Testing

- Add GetExternalEndpoint test suites [EPMDEDP-11726](https://jiraeu.epam.com/browse/EPMDEDP-11726)

### Routine

- Update current development version [EPMDEDP-11472](https://jiraeu.epam.com/browse/EPMDEDP-11472)
- Upgrade alpine image version to 3.16.4 [EPMDEDP-11764](https://jiraeu.epam.com/browse/EPMDEDP-11764)
- Add templates for github issues [EPMDEDP-11928](https://jiraeu.epam.com/browse/EPMDEDP-11928)
- Upgrade alpine image version to 3.18.0 [EPMDEDP-12085](https://jiraeu.epam.com/browse/EPMDEDP-12085)


<a name="v2.14.0"></a>
## [v2.14.0] - 2023-03-25
### Features

- Updated Operator SDK [EPMDEDP-11176](https://jiraeu.epam.com/browse/EPMDEDP-11176)
- Custom gerrit rest and ssh url in spec [EPMDEDP-11198](https://jiraeu.epam.com/browse/EPMDEDP-11198)
- Updated EDP components [EPMDEDP-11206](https://jiraeu.epam.com/browse/EPMDEDP-11206)
- Provide opportunity to use default cluster storageClassName [EPMDEDP-11230](https://jiraeu.epam.com/browse/EPMDEDP-11230)
- Improved Jenkins-related logging [EPMDEDP-11479](https://jiraeu.epam.com/browse/EPMDEDP-11479)
- Add the ability to add additional certs to Gerrit [EPMDEDP-11529](https://jiraeu.epam.com/browse/EPMDEDP-11529)
- Added ability to set constant requeue time in GroupMember Reconciler to master branch [EPMDEDP-11690](https://jiraeu.epam.com/browse/EPMDEDP-11690)

### Bug Fixes

- Gerrit project syncer and controller conflict [EPMDEDP-11142](https://jiraeu.epam.com/browse/EPMDEDP-11142)

### Routine

- Update current development version [EPMDEDP-10610](https://jiraeu.epam.com/browse/EPMDEDP-10610)
- Get gerrit sshPort form global section [EPMDEDP-10642](https://jiraeu.epam.com/browse/EPMDEDP-10642)
- Update current development version [EPMDEDP-11009](https://jiraeu.epam.com/browse/EPMDEDP-11009)
- Update git-chglog for gerrit-operator [EPMDEDP-11518](https://jiraeu.epam.com/browse/EPMDEDP-11518)
- Bump golang.org/x/net from 0.5.0 to 0.8.0 [EPMDEDP-11578](https://jiraeu.epam.com/browse/EPMDEDP-11578)
- Upgrade alpine image version to 3.16.4 [EPMDEDP-11764](https://jiraeu.epam.com/browse/EPMDEDP-11764)

### Documentation

- Update chart and application version in Readme file [EPMDEDP-11221](https://jiraeu.epam.com/browse/EPMDEDP-11221)


<a name="v2.13.5"></a>
## [v2.13.5] - 2023-03-16
### Features

- Added ability to set constant requeue time in GroupMember Reconciler [EPMDEDP-11690](https://jiraeu.epam.com/browse/EPMDEDP-11690)


<a name="v2.13.4"></a>
## [v2.13.4] - 2023-01-23
### Features

- Custom gerrit rest and ssh url in spec [EPMDEDP-11198](https://jiraeu.epam.com/browse/EPMDEDP-11198)

### Routine

- Update git package version to 2.36.4-r0 [EPMDEDP-11260](https://jiraeu.epam.com/browse/EPMDEDP-11260)


<a name="v2.13.3"></a>
## [v2.13.3] - 2022-12-17
### Bug Fixes

- Gerrit project syncer and controller conflict [EPMDEDP-11142](https://jiraeu.epam.com/browse/EPMDEDP-11142)


<a name="v2.13.2"></a>
## [v2.13.2] - 2022-12-06
### Routine

- Get gerrit sshPort form global section [EPMDEDP-10642](https://jiraeu.epam.com/browse/EPMDEDP-10642)


<a name="v2.13.1"></a>
## [v2.13.1] - 2022-11-26
### Routine

- Bump gerrit image version to 3.6.2 [EPMDEDP-11009](https://jiraeu.epam.com/browse/EPMDEDP-11009)


<a name="v2.13.0"></a>
## [v2.13.0] - 2022-11-26
### Features

- Add webhooks plugin configuration [EPMDEDP-10428](https://jiraeu.epam.com/browse/EPMDEDP-10428)
- Eanble webhooks plugin installation [EPMDEDP-10428](https://jiraeu.epam.com/browse/EPMDEDP-10428)
- Do not configure jenkins if not found [EPMDEDP-10643](https://jiraeu.epam.com/browse/EPMDEDP-10643)
- Create gerrit argocd user [EPMDEDP-10988](https://jiraeu.epam.com/browse/EPMDEDP-10988)
- Add base path to gerrit spec [EPMDEDP-11045](https://jiraeu.epam.com/browse/EPMDEDP-11045)

### Bug Fixes

- Escape double quotes for webhooks.config [EPMDEDP-10428](https://jiraeu.epam.com/browse/EPMDEDP-10428)
- Ignore variable expansion for gerrit config [EPMDEDP-10428](https://jiraeu.epam.com/browse/EPMDEDP-10428)
- Write ed25519 private key into the OpenSSH private key format [EPMDEDP-10988](https://jiraeu.epam.com/browse/EPMDEDP-10988)
- SSH command log session close errors [EPMDEDP-10994](https://jiraeu.epam.com/browse/EPMDEDP-10994)
- Ignore EOF error for session close [EPMDEDP-8343](https://jiraeu.epam.com/browse/EPMDEDP-8343)
- Nil pointer panic [EPMDEDP-8343](https://jiraeu.epam.com/browse/EPMDEDP-8343)

### Code Refactoring

- Address linting issues [EPMDEDP-10627](https://jiraeu.epam.com/browse/EPMDEDP-10627)
- Rename Gerrit CI username jenkins [EPMDEDP-10640](https://jiraeu.epam.com/browse/EPMDEDP-10640)
- Enable golangci linter [EPMDEDP-8343](https://jiraeu.epam.com/browse/EPMDEDP-8343)
- Resolve all issues pointed by linter [EPMDEDP-8343](https://jiraeu.epam.com/browse/EPMDEDP-8343)
- Initial preparation for linter [EPMDEDP-8343](https://jiraeu.epam.com/browse/EPMDEDP-8343)

### Routine

- Update current development version [EPMDEDP-10274](https://jiraeu.epam.com/browse/EPMDEDP-10274)
- Downgrade gerrit to version 3.6.2 [EPMDEDP-10752](https://jiraeu.epam.com/browse/EPMDEDP-10752)
- Upgrade gerrit to version 3.7.0 [EPMDEDP-10752](https://jiraeu.epam.com/browse/EPMDEDP-10752)
- Upgrade gerrit to version 3.6.2 [EPMDEDP-10752](https://jiraeu.epam.com/browse/EPMDEDP-10752)
- Upgrade gerrit to version 3.6.2 [EPMDEDP-10752](https://jiraeu.epam.com/browse/EPMDEDP-10752)


<a name="v2.12.1"></a>
## [v2.12.1] - 2023-02-03
### Features

- Add base path to gerrit spec [EPMDEDP-11045](https://jiraeu.epam.com/browse/EPMDEDP-11045)
- Custom gerrit rest and ssh url in spec [EPMDEDP-11198](https://jiraeu.epam.com/browse/EPMDEDP-11198)

### Bug Fixes

- Gerrit project syncer and controller conflict [EPMDEDP-11142](https://jiraeu.epam.com/browse/EPMDEDP-11142)

### Routine

- Update git package version [EPMDEDP-11319](https://jiraeu.epam.com/browse/EPMDEDP-11319)


<a name="v2.12.0"></a>
## [v2.12.0] - 2022-08-26
### Features

- Switch to use V1 apis of EDP components [EPMDEDP-10082](https://jiraeu.epam.com/browse/EPMDEDP-10082)
- Download required tools for Makefile targets [EPMDEDP-10105](https://jiraeu.epam.com/browse/EPMDEDP-10105)
- Use exponential back-off in retries for GerritGroupMemeber reconciliation [EPMDEDP-10341](https://jiraeu.epam.com/browse/EPMDEDP-10341)
- Switch to Ingress v1 [EPMDEDP-8286](https://jiraeu.epam.com/browse/EPMDEDP-8286)
- Switch CRDs to v1 version [EPMDEDP-9218](https://jiraeu.epam.com/browse/EPMDEDP-9218)

### Bug Fixes

- Set proper gerrit image value [EPMDEDP-10120](https://jiraeu.epam.com/browse/EPMDEDP-10120)

### Code Refactoring

- Deprecate unused Spec components for Gerrit v1 [EPMDEDP-10120](https://jiraeu.epam.com/browse/EPMDEDP-10120)
- Remove createCodeReviewPipeline from CR [EPMDEDP-10156](https://jiraeu.epam.com/browse/EPMDEDP-10156)
- Ensure having consisten secret name during gerrit configuration [EPMDEDP-10190](https://jiraeu.epam.com/browse/EPMDEDP-10190)
- Remove unused psp creation [EPMDEDP-10228](https://jiraeu.epam.com/browse/EPMDEDP-10228)
- Properly update status for GerritGroupMember CR [EPMDEDP-10341](https://jiraeu.epam.com/browse/EPMDEDP-10341)
- Use repository and tag for image reference in chart [EPMDEDP-10389](https://jiraeu.epam.com/browse/EPMDEDP-10389)

### Routine

- Upgrade go version to 1.18 [EPMDEDP-10110](https://jiraeu.epam.com/browse/EPMDEDP-10110)
- Fix Jira Ticket pattern for changelog generator [EPMDEDP-10159](https://jiraeu.epam.com/browse/EPMDEDP-10159)
- Update alpine base image to 3.16.2 version [EPMDEDP-10274](https://jiraeu.epam.com/browse/EPMDEDP-10274)
- Update alpine base image version [EPMDEDP-10280](https://jiraeu.epam.com/browse/EPMDEDP-10280)
- Update gerrit to version v3.6.1 [EPMDEDP-10335](https://jiraeu.epam.com/browse/EPMDEDP-10335)
- Upgrade gerrit to version 3.6.1 [EPMDEDP-10335](https://jiraeu.epam.com/browse/EPMDEDP-10335)
- Change 'go get' to 'go install' for git-chglog [EPMDEDP-10337](https://jiraeu.epam.com/browse/EPMDEDP-10337)
- Use deployments as default deploymentType for OpenShift [EPMDEDP-10344](https://jiraeu.epam.com/browse/EPMDEDP-10344)
- Update Gerrit to 3.6.1 release version [EPMDEDP-10374](https://jiraeu.epam.com/browse/EPMDEDP-10374)
- Remove VERSION file [EPMDEDP-10387](https://jiraeu.epam.com/browse/EPMDEDP-10387)
- Add gcflags for go build artifact [EPMDEDP-10411](https://jiraeu.epam.com/browse/EPMDEDP-10411)
- Update current development version [EPMDEDP-8832](https://jiraeu.epam.com/browse/EPMDEDP-8832)
- Update chart annotation [EPMDEDP-9515](https://jiraeu.epam.com/browse/EPMDEDP-9515)

### Documentation

- Align README.md [EPMDEDP-10274](https://jiraeu.epam.com/browse/EPMDEDP-10274)


<a name="v2.11.0"></a>
## [v2.11.0] - 2022-05-25
### Features

- Manage Gerrit access [EPMDEDP-7502](https://jiraeu.epam.com/browse/EPMDEDP-7502)
- Implement Developers group creation in go code [EPMDEDP-7502](https://jiraeu.epam.com/browse/EPMDEDP-7502)
- implement gerrit merge request CR [EPMDEDP-8162](https://jiraeu.epam.com/browse/EPMDEDP-8162)
- add git executable to docker image [EPMDEDP-8162](https://jiraeu.epam.com/browse/EPMDEDP-8162)
- Update Makefile changelog target [EPMDEDP-8218](https://jiraeu.epam.com/browse/EPMDEDP-8218)
- add status tracking to gerrit merge request [EPMDEDP-8222](https://jiraeu.epam.com/browse/EPMDEDP-8222)
- make time interval configurable. [EPMDEDP-8244](https://jiraeu.epam.com/browse/EPMDEDP-8244)
- Allow to re-define project sync time [EPMDEDP-8244](https://jiraeu.epam.com/browse/EPMDEDP-8244)
- cmd args for git merge [EPMDEDP-8305](https://jiraeu.epam.com/browse/EPMDEDP-8305)
- add .golangci-lint config [EPMDEDP-8343](https://jiraeu.epam.com/browse/EPMDEDP-8343)
- Add ingress tls certificate option when using ingress controller [EPMDEDP-8377](https://jiraeu.epam.com/browse/EPMDEDP-8377)
- Generate CRDs and helm docs automatically [EPMDEDP-8385](https://jiraeu.epam.com/browse/EPMDEDP-8385)
- Add read only group [EPMDEDP-8890](https://jiraeu.epam.com/browse/EPMDEDP-8890)
- Merge request with files contents from config map [EPMDEDP-9108](https://jiraeu.epam.com/browse/EPMDEDP-9108)

### Bug Fixes

- add author name email to merge request CRD [EPMDEDP-8162](https://jiraeu.epam.com/browse/EPMDEDP-8162)
- Fix changelog generation in GH Release Action [EPMDEDP-8468](https://jiraeu.epam.com/browse/EPMDEDP-8468)

### Code Refactoring

- remove legacy fields in Gerrit CR spec. [EPMDEDP-7536](https://jiraeu.epam.com/browse/EPMDEDP-7536)

### Testing

- Add tests [EPMDEDP-7992](https://jiraeu.epam.com/browse/EPMDEDP-7992)
- Add tests and mocks [EPMDEDP-7992](https://jiraeu.epam.com/browse/EPMDEDP-7992)
- Fix unit tests [EPMDEDP-7992](https://jiraeu.epam.com/browse/EPMDEDP-7992)
- Add tests and mocks [EPMDEDP-7992](https://jiraeu.epam.com/browse/EPMDEDP-7992)
- add tests for git and gerrit client [EPMDEDP-8162](https://jiraeu.epam.com/browse/EPMDEDP-8162)
- fix gerrit service tests [EPMDEDP-8222](https://jiraeu.epam.com/browse/EPMDEDP-8222)
- fix running in cluster function [EPMDEDP-8222](https://jiraeu.epam.com/browse/EPMDEDP-8222)

### Routine

- Update release CI pipelines [EPMDEDP-7847](https://jiraeu.epam.com/browse/EPMDEDP-7847)
- Fix CI for codecov report [EPMDEDP-7992](https://jiraeu.epam.com/browse/EPMDEDP-7992)
- Populate chart with Artifacthub annotations [EPMDEDP-8049](https://jiraeu.epam.com/browse/EPMDEDP-8049)
- Update gerrit URL baseline link [EPMDEDP-8204](https://jiraeu.epam.com/browse/EPMDEDP-8204)
- Update changelog [EPMDEDP-8227](https://jiraeu.epam.com/browse/EPMDEDP-8227)
- Update base docker image to alpine 3.15.4 [EPMDEDP-8853](https://jiraeu.epam.com/browse/EPMDEDP-8853)
- Update changelog [EPMDEDP-9185](https://jiraeu.epam.com/browse/EPMDEDP-9185)

### BREAKING CHANGE:


Respective GerritGroupMember Custom Resources must be created to replace existing users[] mapping. Consult release upgrade instruction

Update gerrit config according to groups.

* Implement Developers group creation;
* Assign users to admins and developers groups using cr GerritGroupMember;
* Align permission for groups.


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

[Unreleased]: https://github.com/epam/edp-gerrit-operator/compare/v2.16.0...HEAD
[v2.16.0]: https://github.com/epam/edp-gerrit-operator/compare/v2.15.0...v2.16.0
[v2.15.0]: https://github.com/epam/edp-gerrit-operator/compare/v2.14.0...v2.15.0
[v2.14.0]: https://github.com/epam/edp-gerrit-operator/compare/v2.13.5...v2.14.0
[v2.13.5]: https://github.com/epam/edp-gerrit-operator/compare/v2.13.4...v2.13.5
[v2.13.4]: https://github.com/epam/edp-gerrit-operator/compare/v2.13.3...v2.13.4
[v2.13.3]: https://github.com/epam/edp-gerrit-operator/compare/v2.13.2...v2.13.3
[v2.13.2]: https://github.com/epam/edp-gerrit-operator/compare/v2.13.1...v2.13.2
[v2.13.1]: https://github.com/epam/edp-gerrit-operator/compare/v2.13.0...v2.13.1
[v2.13.0]: https://github.com/epam/edp-gerrit-operator/compare/v2.12.1...v2.13.0
[v2.12.1]: https://github.com/epam/edp-gerrit-operator/compare/v2.12.0...v2.12.1
[v2.12.0]: https://github.com/epam/edp-gerrit-operator/compare/v2.11.0...v2.12.0
[v2.11.0]: https://github.com/epam/edp-gerrit-operator/compare/v2.10.0...v2.11.0
[v2.10.0]: https://github.com/epam/edp-gerrit-operator/compare/v2.9.0...v2.10.0
[v2.9.0]: https://github.com/epam/edp-gerrit-operator/compare/v2.8.0...v2.9.0
[v2.8.0]: https://github.com/epam/edp-gerrit-operator/compare/v2.7.2...v2.8.0
[v2.7.2]: https://github.com/epam/edp-gerrit-operator/compare/v2.7.1...v2.7.2
[v2.7.1]: https://github.com/epam/edp-gerrit-operator/compare/v2.7.0...v2.7.1
[v2.7.0]: https://github.com/epam/edp-gerrit-operator/compare/v2.3.0-73...v2.7.0
