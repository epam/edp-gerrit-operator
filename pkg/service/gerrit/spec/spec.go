package spec

const (
	// Port = 8080
	Port = 8080

	// SSHPort = 29418
	SSHPort = 29418

	// SSHPortName = "ssh"
	SSHPortName = "ssh"

	// SSHListnerEnvName
	SSHListnerEnvName = "LISTEN_ADDR"

	// LivenessProbeDelay = 180
	LivenessProbeDelay = 180

	// ReadinessProbeDelay = 60
	ReadinessProbeDelay = 60

	// MemoryRequest = "500Mi"
	MemoryRequest = "500Mi"

	// StatusInstall = "installing"
	StatusInstall = "installing"

	// StatusFailed = "failed"
	StatusFailed = "failed"

	// StatusCreated = "created"
	StatusCreated = "created"

	// StatusConfiguring = "configuring"
	StatusConfiguring = "configuring"

	// StatusConfigured = "configured"
	StatusConfigured = "configured"

	// StatusReady = "ready"
	StatusReady = "ready"

	// StatuseExposeConf = "exposing config"
	StatuseExposeConf = "exposing config"

	//GerritDefaultAdminUser - default admin username in Gerrit
	GerritDefaultAdminUser = "admin"

	//GerritDefaultAdminPassword - default admin password in Gerrit
	GerritDefaultAdminPassword = "secret"

	//GerritRestApiUrlPath - Gerrit relative REST API path
	GerritRestApiUrlPath = "a/"

	//Gerrit port
	GerritPort = "8080"

	//GerritCIToolsGroupName - default group name for Continuous Integration users
	GerritCIToolsGroupName = "Continuous Integration Tools"

	//GerritCIToolsGroupDescription - default group description for Continuous Integration Tools group
	GerritCIToolsGroupDescription = "Contains Jenkins and any other CI tools that get +2/-2 access on reviews"

	//GerritProjectBootstrappersGroupName - default group name for Project Bootstrappers users
	GerritProjectBootstrappersGroupName = "Project Bootstrappers"

	//GerritProjectBootstrappersGroupDescription - default group description for Project Bootstrappers group
	GerritProjectBootstrappersGroupDescription = "Grants all the permissions needed to set up a new project"

	//GerritServiceUsersGroup - group for users who perform batch actions on Gerrit
	GerritServiceUsersGroup = "Service Users"

	//GerritAdministratorsGroup - group for users who perform batch actions on Gerrit
	GerritAdministratorsGroup = "Administrators"

	//GerritDefaultCiUserUser - default jenkins username in Gerrit
	GerritDefaultCiUserUser = "jenkins"

	//GerritDefaultProjectCreatorUser - default project-creator username in Gerrit
	GerritDefaultProjectCreatorUser = "project-creator"

	//GerritDefaultCiUserSecretPostfix - default CI user secret postfix for Gerrit
	GerritDefaultCiUserSecretPostfix = "ciuser-password"

	//EdpAnnotationsPrefix general prefix for all annotation made by EDP team
	EdpAnnotationsPrefix = "edp.epam.com"

	//EdpCiUserSuffix entity prefix for integration functionality
	EdpCiUserSuffix string = "ci-credentials"

	//EdpCiUserKeySuffix entity prefix for integration functionality
	EdpCiUSerSshKeySuffix string = "ci-sshkey"

	//GerritDefaultCiUserSecretPostfix - default CI user secret postfix for Gerrit
	GerritDefaultProjectCreatorSecretPostfix = "project-creator"

	//JenkinsPluginConfigPostfix
	JenkinsPluginConfigPostfix = "jenkins-plugin-config"

	//EdpCiUserSuffix entity prefix for integration functionality
	EdpProjectCreatorUserSuffix string = "project-creator-credentials"

	//EdpCiUserKeySuffix entity prefix for integration functionality
	EdpProjectCreatorSshKeySuffix string = "project-creator-sshkey"

	//DefaultGerritReplicationConfigPath replication config path
	DefaultGerritReplicationConfigPath = "/var/gerrit/review_site/etc/replication.config"

	//DefaultGerritSSHConfigPath ssh config path
	DefaultGerritSSHConfigPath = "/var/gerrit/.ssh"

	//GerritDefaultVCSKeyPath - default path for VCS key
	GerritDefaultVCSKeyPath = "/var/gerrit/review_site/etc"

	//GerritDefaultVCSKeyName - default name for VCS key
	GerritDefaultVCSKeyName = "vcs-autouser"

	//IdentityServiceCredentialsSecretPostfix
	IdentityServiceCredentialsSecretPostfix = "is-credentials"

	//SshKeyPostfix - default name for Ssh key postfix
	SshKeyPostfix = "-sshkey"
)
