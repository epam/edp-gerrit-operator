package spec

const (
	// Port = 8080.
	Port = 8080

	// SSHPort = 29418.
	SSHPort = 29418

	// SSHPortName = "ssh".
	SSHPortName = "ssh"

	// SSHListnerEnvName.
	SSHListnerEnvName = "LISTEN_ADDR"

	// LivenessProbeDelay = 180.
	LivenessProbeDelay = 180

	// ReadinessProbeDelay = 60.
	ReadinessProbeDelay = 60

	// MemoryRequest = "500Mi".
	MemoryRequest = "500Mi"

	// StatusInstall = "installing".
	StatusInstall = "installing"

	// StatusFailed = "failed".
	StatusFailed = "failed"

	// StatusCreated = "created".
	StatusCreated = "created"

	// StatusConfiguring = "configuring".
	StatusConfiguring = "configuring"

	// StatusConfigured = "configured".
	StatusConfigured = "configured"

	// StatusReady = "ready".
	StatusReady = "ready"

	// StatuseExposeConf = "exposing config".
	StatuseExposeConf = "exposing config"

	GerritDefaultAdminUser = "admin"

	GerritDefaultAdminPassword = "secret"

	GerritRestApiUrlPath = "a/"

	GerritPort = "8080"

	GerritCIToolsGroupName = "Continuous Integration Tools"

	GerritCIToolsGroupDescription = "Contains Jenkins and any other CI tools that get +2/-2 access on reviews"

	GerritProjectDevelopersGroupName = "Developers"

	GerritReadOnlyGroupName = "ReadOnly"

	GerritProjectDevelopersGroupNameDescription = "Grant access to all projects in Gerrit"

	GerritProjectBootstrappersGroupName = "Project Bootstrappers"

	GerritProjectBootstrappersGroupDescription = "Grants all the permissions needed to set up a new project"

	GerritAdministratorsGroup = "Administrators"

	GerritDefaultCiUserUser = "edp-ci"

	GerritArgoUser = "argocd"

	GerritDefaultCiUserSecretPostfix = "ciuser-password"

	GerritArgoUserSecretPostfix = "argocd-password"

	EdpAnnotationsPrefix = "edp.epam.com"

	EdpCiUserSuffix string = "ci-credentials"

	EdpArgoUserSuffix string = "argocd-credentials"

	EdpCiUSerSshKeySuffix string = "ci-sshkey"

	EdpArgoUserSshKeySuffix string = "argocd-sshkey"

	JenkinsPluginConfigPostfix = "jenkins-plugin-config"

	DefaultGerritReplicationConfigPath = "/var/gerrit/review_site/etc/replication.config"

	DefaultGerritSSHConfigPath = "/var/gerrit/.ssh"

	GerritDefaultVCSKeyPath = "/var/gerrit/review_site/etc"

	GerritDefaultVCSKeyName = "vcs-autouser"

	IdentityServiceCredentialsSecretPostfix = "is-credentials"

	SshKeyPostfix = "-sshkey"
)
