// Package internal will updating of wallpaper
package internal

import (
	"io/ioutil"
	"os"

	"github.com/iotdomain/iotdomain-go/publisher"
	"github.com/iotdomain/iotdomain-go/types"
)

// CheckUpdateWallpapers checks each montage image if it needs to be exported
func (app *WallpaperApp) CheckUpdateWallpapers(pub *publisher.Publisher) {

	for _, montage := range app.montages {
		if montage.UpdateCount > 0 {
			app.UpdateWallpaper(montage)
		}
	}
}

// UpdateWallpaper generates a new wallpaper image.
// Depending on the configuration, the image is saved and/or published
func (app *WallpaperApp) UpdateWallpaper(montage *Montage) {
	jpegData, err := montage.CreateMontageAsJPEG()
	if err != nil {
		// app.logger.Errorf("Updatewallpaper: Error generating montage image for %s: %s", montage.Config.ID, err)
		return
	}
	filename := montage.Config.Filename
	if filename != "" {
		err = ioutil.WriteFile(filename, jpegData, os.ModePerm)
	}
	if montage.Config.Publish {
		wallpaperNode := app.pub.GetNodeByID(montage.Config.ID)
		output := app.pub.GetOutputByType(wallpaperNode.NodeID, types.OutputTypeImage, types.DefaultOutputInstance)
		app.pub.PublishRaw(output, false, jpegData)
	}

}
