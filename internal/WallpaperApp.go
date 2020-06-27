package internal

import (
	"strconv"
	"strings"

	"github.com/fsnotify/fsnotify"
	"github.com/iotdomain/iotdomain-go/publisher"
	"github.com/iotdomain/iotdomain-go/types"
	"github.com/sirupsen/logrus"
)

// AppID application name used for configuration file and default publisherID
const AppID = "wallpaper"

// WallpaperAppConfig with application configuration, loaded from wallpaper.yaml
type WallpaperAppConfig struct {
	PublisherID string                    `yaml:"publisherId"` // default publisher is app ID
	Wallpapers  map[string]*MontageConfig `yaml:"wallpapers"`  // collection of wallpapers this service provides
	UseLibJPEG  bool                      `yaml:"useLibJPEG"`  // Use the faster libjpeg library instead of the golang image library
}

// WallpaperApp publisher app
type WallpaperApp struct {
	config      *WallpaperAppConfig // wallpaper application configuration
	pub         *publisher.Publisher
	logger      *logrus.Logger
	montages    map[string]*Montage // active wallpaper montages
	fileWatcher *fsnotify.Watcher
}

// CreateWallpapersFromConfig create wallpaper nodes and montages from the app config
func (app *WallpaperApp) CreateWallpapersFromConfig(config *WallpaperAppConfig) {
	pub := app.pub
	app.logger.Infof("Loading %d wallpapers from config", len(config.Wallpapers))

	for wpID, wpInfo := range config.Wallpapers {
		pub.NewNode(wpID, types.NodeTypeWallpaper)
		// pub.SetNodeAttr(wpid, types.NodeAttrMap{types.NodeAttrDescription: wpInfo.Address})

		pub.UpdateNodeConfig(wpID, "border", &types.ConfigAttr{
			DataType:    types.DataTypeInt,
			Description: "Thickness of the border between images in number of pixels",
			Default:     "1",
			Min:         0,
			Max:         100,
		})
		pub.UpdateNodeConfig(wpID, "height", &types.ConfigAttr{
			DataType:    types.DataTypeInt,
			Description: "Height of the wallpaper image",
			Default:     "1080",
			Min:         16,
			Max:         2160,
		})
		pub.UpdateNodeConfig(wpID, "width", &types.ConfigAttr{
			DataType:    types.DataTypeInt,
			Description: "Width of the wallpaper image",
			Default:     "1920",
			Min:         16,
			Max:         4096,
		})
		pub.UpdateNodeConfig(wpID, "publish", &types.ConfigAttr{
			DataType:    types.DataTypeBool,
			Description: "Publish the wallpaper image on the $raw output address",
			Default:     "false",
		})
		pub.UpdateNodeConfig(wpID, "resize", &types.ConfigAttr{
			DataType:    types.DataTypeEnum,
			Description: "Resize the resulting composition to the given dimensions ",
			Default:     "scale",
			Enum:        []string{"scale", "crop", "none", "height", "width"},
		})
		pub.UpdateNodeConfig(wpID, "rows", &types.ConfigAttr{
			DataType:    types.DataTypeInt,
			Description: "Number of rows to organize images in",
			Default:     "1",
			Min:         1,
			Max:         5,
		})
		// the image and build time are both outputs
		if wpInfo.Publish {
			pub.NewOutput(wpID, types.OutputTypeImage, types.DefaultOutputInstance)
		}
		pub.NewOutput(wpID, types.OutputTypeLatency, types.DefaultOutputInstance)

		// Subscribe to source images...
		// TODO: can we define inputs that link/subscribe to other outputs?
		// TODO: configure inputs
		for index, placement := range wpInfo.Images {
			input := pub.NewInputSource(wpID, types.InputTypeImage, strconv.Itoa(index), placement.Source)
			input.Description = placement.Source
			if strings.HasPrefix(placement.Source, "file://") {
				app.fileWatcher.Add(placement.Source)
			} else if strings.HasPrefix(placement.Source, "http://") {
				app.logger.Errorf("CreateWallpapersFromConfig: http source not yet supported: ", placement.Source)
			} else {
				// topic subscription
				pub.messenger.Subscribe(placement.Source, HandleInputCommand)
			}
		}

		//
		montage := NewMontage(wpInfo, app.logger, app.config.UseLibJPEG)
		app.montages[wpID] = montage
	}
}

// NewWallpaperApp creates the app
func NewWallpaperApp(config *WallpaperAppConfig, pub *publisher.Publisher) *WallpaperApp {
	app := WallpaperApp{
		config: config,
		pub:    pub,
		logger: logrus.New(),
	}
	if app.config.PublisherID == "" {
		app.config.PublisherID = AppID
	}
	app.fileWatcher, _ = fsnotify.NewWatcher()
	app.CreateWallpapersFromConfig(config)

	// Check if to update a wallpaper
	pub.SetPollInterval(1, app.CheckUpdateWallpapers)

	// pub.SetNodeInputHandler(app.HandleInputCommand)
	pub.SetNodeConfigHandler(app.HandleConfigCommand)
	// // Discover the node(s) and outputs. Use default for republishing discovery
	// onewirePub.SetDiscoveryInterval(0, app.Discover)

	return &app
}

// Run the publisher until the SIGTERM  or SIGINT signal is received
func Run() {
	appConfig := &WallpaperAppConfig{PublisherID: AppID}
	pub, _ := publisher.NewAppPublisher(AppID, "", "", appConfig, true)

	NewWallpaperApp(appConfig, pub)

	pub.Start()
	pub.WaitForSignal()
	pub.Stop()
}
