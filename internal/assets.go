package internal

import (
	"embed"
	"fmt"
	"runtime"
	"strings"
)

//go:embed assets/*
var assets embed.FS

// GetAsset returns asset file.
// cannot be accessed from outer package.
func GetAsset(filename string) ([]byte, error) {
	return assets.ReadFile("assets/" + filename)
}

// GetSsmPluginName returns filename for aws ssm plugin.
func GetSsmPluginName() string {
	if strings.ToLower(runtime.GOOS) == "windows" {
		return "session-manager-plugin.exe"
	} else {
		return "session-manager-plugin"
	}
}

// GetSsmPlugin returns filepath for aws ssm plugin.
func GetSsmPlugin() ([]byte, error) {
	return GetAsset(getSSMPluginKey())
}

func getSSMPluginKey() string {
	return fmt.Sprintf("plugin/%s_%s/%s",
		strings.ToLower(runtime.GOOS), strings.ToLower(runtime.GOARCH), GetSsmPluginName())
}
