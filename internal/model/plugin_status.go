package model

type PluginStatus string

const (
	PluginStatusDiscovered PluginStatus = "Discovered"
	PluginStatusLoaded     PluginStatus = "Loaded"
	PluginStatusEnabled    PluginStatus = "Enabled"
	PluginStatusDisabled   PluginStatus = "Disabled"
	PluginStatusError      PluginStatus = "Error"
	PluginStatusRemoved    PluginStatus = "Removed"
)

func IsValidPluginStatus(s PluginStatus) bool {
	switch s {
	case PluginStatusDiscovered, PluginStatusLoaded, PluginStatusEnabled, PluginStatusDisabled, PluginStatusError, PluginStatusRemoved:
		return true
	default:
		return false
	}
}

func IsExecutablePluginStatus(s PluginStatus) bool { return s == PluginStatusEnabled }
