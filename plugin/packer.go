package plugin

import (
	"fmt"
	"github.com/gobuffalo/packr/v2"
	"runtime"
	"strings"
)

var (
	pluginBox     *packr.Box
	pluginNameKey string
)

func init() {
	// embed go binary
	pluginNameKey = fmt.Sprintf("%s_%s/session-manager-plugin",
		strings.ToLower(runtime.GOOS), strings.ToLower(runtime.GOARCH))
	pluginBox = packr.New("pluginBox", fmt.Sprintf("./%s", pluginNameKey))
}

func GetPlugin() ([]byte, error) {
	return pluginBox.Find(pluginNameKey)
}
