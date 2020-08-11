package internal

import (
	"io/ioutil"
	"os"
	"strconv"
	"strings"

	"github.com/iotdomain/iotdomain-go/publisher"
	"github.com/iotdomain/iotdomain-go/types"
	"github.com/sirupsen/logrus"
)

// AppID application name used for configuration file and default publisherID
const AppID = "wallpaper"

// AppConfig with application configuration, loaded from wallpaper.yaml
type AppConfig struct {
	// PublisherID string                    `yaml:"publisherId"` // default publisher is app ID
	Wallpapers []*MontageConfig `yaml:"wallpapers"` // collection of wallpapers
	UseLibJPEG bool             `yaml:"useLibJPEG"` // Use the faster libjpeg library instead of the golang image library
}

// WallpaperApp publisher app
type WallpaperApp struct {
	config   *AppConfig // wallpaper application configuration
	pub      *publisher.Publisher
	montages map[string]*Montage // active wallpaper montages
}

// CreateWallpaper creates wallpaper nodes, inputs and and montages from the given config
func (app *WallpaperApp) CreateWallpaper(config *MontageConfig) *Montage {
	pub := app.pub
	logrus.Infof("CreateWallpaper %s", config.ID)
	deviceID := config.ID

	pub.CreateNode(deviceID, types.NodeTypeWallpaper)
	// pub.SetNodeAttr(wpid, types.NodeAttrMap{types.NodeAttrDescription: wpInfo.Address})

	pub.UpdateNodeConfig(deviceID, "border", &types.ConfigAttr{
		DataType:    types.DataTypeInt,
		Description: "Thickness of the border between images in number of pixels",
		Default:     "1",
		Min:         0,
		Max:         100,
	})
	pub.UpdateNodeConfig(deviceID, "height", &types.ConfigAttr{
		DataType:    types.DataTypeInt,
		Description: "Height of the wallpaper image",
		Default:     "1080",
		Min:         16,
		Max:         2160,
	})
	pub.UpdateNodeConfig(deviceID, "width", &types.ConfigAttr{
		DataType:    types.DataTypeInt,
		Description: "Width of the wallpaper image",
		Default:     "1920",
		Min:         16,
		Max:         4096,
	})
	pub.UpdateNodeConfig(deviceID, "publish", &types.ConfigAttr{
		DataType:    types.DataTypeBool,
		Description: "Publish the wallpaper image on the $raw output address",
		Default:     "false",
	})
	pub.UpdateNodeConfig(deviceID, "resize", &types.ConfigAttr{
		DataType:    types.DataTypeEnum,
		Description: "Resize the resulting composition to the given dimensions ",
		Default:     "scale",
		Enum:        []string{"scale", "crop", "none", "height", "width"},
	})
	pub.UpdateNodeConfig(deviceID, "rows", &types.ConfigAttr{
		DataType:    types.DataTypeInt,
		Description: "Number of rows to organize images in",
		Default:     "1",
		Min:         1,
		Max:         5,
	})
	// the image and build time are both outputs
	if config.Publish {
		pub.CreateOutput(deviceID, types.OutputTypeImage, types.DefaultOutputInstance)
	}
	pub.CreateOutput(deviceID, types.OutputTypeLatency, types.DefaultOutputInstance)

	// Subscribe to source images...
	// TODO: can we define inputs that link/subscribe to other outputs?
	// TODO: configure inputs
	for index, placement := range config.ProposedPlacements {
		// each image is an input
		// input := pub.NewInput(wpID, types.InputTypeImage, strconv.Itoa(index))
		// input.Attr[types.NodeAttrAddress] = placement.Source
		if strings.HasPrefix(placement.Source, "file://") {
			app.pub.CreateInputFromFile(deviceID, types.InputTypeImage, strconv.Itoa(index),
				placement.Source, app.HandleInputImage)
			// app.fileWatcher.Add(placement.Source)
		} else if strings.HasPrefix(placement.Source, "http://") {
			login := ""
			pass := ""
			app.pub.CreateInputFromHTTP(deviceID, types.InputTypeImage, strconv.Itoa(index),
				placement.Source, login, pass, placement.Interval, app.HandleInputImage)
			logrus.Errorf("CreateWallpapersFromConfig: http source '%s' not yet supported: ", placement.Source)
		} else {
			app.pub.CreateInputFromOutput(deviceID, types.InputTypeImage, strconv.Itoa(index),
				placement.Source, app.HandleInputImage)
			// pub.messenger.Subscribe(placement.Source, HandleInputCommand)
		}
	}

	//
	montage := NewMontage(config, app.config.UseLibJPEG)
	app.montages[deviceID] = montage
	return montage
}

// CreateWallpapersFromAppConfig creates new wallpapers from the application configuration
// during startup.
func (app *WallpaperApp) CreateWallpapersFromAppConfig(config *AppConfig) {
	logrus.Infof("Loading %d wallpapers from config", len(config.Wallpapers))

	for _, wpInfo := range config.Wallpapers {
		montage := app.CreateWallpaper(wpInfo)
		_ = montage
	}
}

// CheckUpdateWallpapers checks each montage image if it has been updated and a
// new image should be generated.
func (app *WallpaperApp) CheckUpdateWallpapers(pub *publisher.Publisher) {

	for _, montage := range app.montages {
		if montage.UpdateCount > 0 {
			app.GenerateWallpaperImage(montage)
		}
	}
}

// DeleteWallpaper deletes a wallpaper
func (app *WallpaperApp) DeleteWallpaper(ID string) {
	app.montages[ID] = nil
}

// GetWallpaper returns a wallpaper montage instance by its ID
func (app *WallpaperApp) GetWallpaper(ID string) *Montage {
	montage := app.montages[ID]
	return montage
}

// GenerateWallpaperImage generates a new wallpaper image.
// Depending on the configuration, the image is saved and/or published
func (app *WallpaperApp) GenerateWallpaperImage(montage *Montage) {
	jpegData, err := montage.ExportMontageAsJPEG()
	if err != nil {
		// app.logger.Errorf("Updatewallpaper: Error generating montage image for %s: %s", montage.Config.ID, err)
		return
	}
	filename := montage.Config.Filename
	if filename != "" {
		err = ioutil.WriteFile(filename, jpegData, os.ModePerm)
	}
	if montage.Config.Publish {
		output := app.pub.GetOutputByDevice(montage.Config.ID, types.OutputTypeImage, types.DefaultOutputInstance)
		app.pub.PublishRaw(output, false, string(jpegData))
	}

}

// HandleInputImage updates the wallpaper image
func (app *WallpaperApp) HandleInputImage(input *types.InputDiscoveryMessage, sender string, image string) {
	logrus.Infof("HandleInputUpdate: Update to input %s from '%s'", input.InputID, sender)
	montage := app.GetWallpaper(input.DeviceID)
	montage.UpdateImage(input.Source, []byte(image))
}

// NewWallpaperApp creates the wallpapers from config
func NewWallpaperApp(config *AppConfig, pub *publisher.Publisher) *WallpaperApp {
	app := WallpaperApp{
		config:   config,
		pub:      pub,
		montages: make(map[string]*Montage),
	}
	app.CreateWallpapersFromAppConfig(config)

	// Each second check if a new wallpaper image needs to be generated after update
	pub.SetPollInterval(1, app.CheckUpdateWallpapers)
	// Support remote wallpaper configuration
	pub.SetNodeConfigHandler(app.HandleConfigCommand)
	return &app
}

// Run the publisher until the SIGTERM  or SIGINT signal is received
func Run() {
	appConfig := &AppConfig{}
	appConfig.Wallpapers = make([]*MontageConfig, 0)
	pub, _ := publisher.NewAppPublisher(AppID, "", appConfig, true)

	NewWallpaperApp(appConfig, pub)

	pub.Start()
	pub.WaitForSignal()
	pub.Stop()
}
