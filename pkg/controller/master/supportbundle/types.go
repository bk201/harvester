package supportbundle

const (
	StateNone        = ""
	StateGenerating  = "generating"
	StateManagerDone = "managerdone"
	StateAgentDone   = "agentdone"
	StateError       = "error"
	StateReady       = "ready"

	// labels
	HarvesterNodeLabelKey   = "harvesterhci.io/managed"
	HarvesterNodeLabelValue = "true"
	SupportBundleLabelKey   = "harvesterhci.io/supportbundle"

	AppManager = "support-bundle-manager"
	AppAgent   = "support-bundle-agent"
)
