package spec

const (
	// Image = openfrontier/gerrit
	Image = "openfrontier/gerrit"

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

	//GerritDefaultScriptsPath - default scripts for uploading to Gerrit
	GerritDefaultScriptsPath = "/usr/local/configs/scripts"

	//LocalConfigsRelativePath - default directory for configs
	LocalConfigsRelativePath = "configs"

	//GerritCIToolsGroupName - default group name for Continuous Integration users
	GerritCIToolsGroupName = "Continuous Integration Tools"

	//GerritCIToolsGroupDescription - default group description for Continuous Integration Tools group
	GerritCIToolsGroupDescription = "Contains Jenkins and any other CI tools that get +2/-2 access on reviews"

	//GerritProjectBootstrappersGroupName - default group name for Project Bootstrappers users
	GerritProjectBootstrappersGroupName = "Project Bootstrappers"

	//GerritProjectBootstrappersGroupDescription - default group description for Project Bootstrappers group
	GerritProjectBootstrappersGroupDescription = "Grants all the permissions needed to set up a new project"

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

	//EdpCiUserSuffix entity prefix for integration functionality
	EdpProjectCreatorUserSuffix string = "project-creator-credentials"

	//EdpCiUserKeySuffix entity prefix for integration functionality
	EdpProjectCreatorSshKeySuffix string = "project-creator-sshkey"
)
