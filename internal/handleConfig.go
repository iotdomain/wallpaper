// Package internal with wallpaper node configuration
package internal

import (
	"github.com/iotdomain/iotdomain-go/types"
	"github.com/sirupsen/logrus"
)

// HandleConfigCommand handles requests to update node configuration
func (app *WallpaperApp) HandleConfigCommand(address string, config types.NodeAttrMap) types.NodeAttrMap {
	logrus.Infof("Wallpaper.HandleConfigCommand for %s. ", address)
	return config
}
