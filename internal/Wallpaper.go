package internal

import (
	"github.com/iotdomain/iotdomain-go/publisher"
	"github.com/sirupsen/logrus"
)

// AppID application name used for configuration file and default publisherID
const AppID = "wallpaper"

// WallpaperConfig with application configuration, loaded from wallpaper.yaml
type WallpaperConfig struct {
	PublisherID string `yaml:"publisherId"` // default publisher is app ID
}

// Wallpaper publisher app
type Wallpaper struct {
	config *WallpaperConfig
	pub    *publisher.Publisher
	logger *logrus.Logger
}

// NewWallpaper creates the app
func NewWallpaper(config *WallpaperConfig, pub *publisher.Publisher) *Wallpaper {
	app := Wallpaper{
		config: config,
		pub:    pub,
		logger: logrus.New(),
	}
	if app.config.PublisherID == "" {
		app.config.PublisherID = AppID
	}
	//	pub.SetPollInterval(60, app.Poll)
	// pub.SetNodeInputHandler(app.HandleInputCommand)
	pub.SetNodeConfigHandler(app.HandleConfigCommand)
	// // Discover the node(s) and outputs. Use default for republishing discovery
	// onewirePub.SetDiscoveryInterval(0, app.Discover)

	return &app
}

// Run the publisher until the SIGTERM  or SIGINT signal is received
func Run() {
	appConfig := &WallpaperConfig{PublisherID: AppID}
	pub, _ := publisher.NewAppPublisher(AppID, "", "", appConfig, true)

	NewWallpaper(appConfig, pub)

	pub.Start()
	pub.WaitForSignal()
	pub.Stop()
}
