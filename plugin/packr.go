package plugin

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/gobuffalo/packr/v2"
)

var (
	pluginBox      *packr.Box
	pluginKey      string
	pluginFileName string
)

func init() {
	// embed go binary
	if strings.ToLower(runtime.GOOS) == "windows" {
		pluginFileName = "session-manager-plugin.exe"
	} else {
		pluginFileName = "session-manager-plugin"
	}
	pluginKey = fmt.Sprintf("%s_%s/%s",
		strings.ToLower(runtime.GOOS), strings.ToLower(runtime.GOARCH), pluginFileName)
	pluginBox = packr.New("pluginBox", "./")
}

// GetPlugin is returning plugin.
func GetPlugin() ([]byte, error) {
	return pluginBox.Find(pluginKey)
}

// GetPluginFileName is returning plugin name.
func GetPluginFileName() string {
	return pluginFileName
}
