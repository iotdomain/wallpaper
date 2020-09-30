// Package internal with wallpaper node configuration
package internal

import (
	"github.com/iotdomain/iotdomain-go/types"
	"github.com/sirupsen/logrus"
)

// HandleConfigCommand handles requests to update node configuration
func (app *WallpaperApp) HandleConfigCommand(nodeHWID string, config types.NodeAttrMap) {
	logrus.Infof("Wallpaper.HandleConfigCommand for node %s. ", nodeHWID)
	app.pub.UpdateNodeConfigValues(nodeHWID, config)
}
