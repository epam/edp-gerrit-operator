package spec

const (
	// Image = openfrontier/gerrit
	Image = "openfrontier/gerrit"

	// Port = 8080
	Port = 8080

	// SSHPort = 30001
	SSHPort = 30001

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
)
