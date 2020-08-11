package internal

import (
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/iotdomain/iotdomain-go/publisher"
	"github.com/stretchr/testify/assert"
)

const cacheFolder = "../test/cache"
const configFolder = "../test"

const TestMontageFile = "../test/montage.jpeg"

var appConfig *AppConfig = &AppConfig{}

var config1 = MontageConfig{
	ID:           "screen1",
	Name:         "Screen 1",
	Filename:     TestMontageFile,
	Width:        1680,
	Height:       1080,
	Rows:         1,
	Border:       1,
	Resize:       MontageResizeNone, // 'height', 'width', 'crop', 'scale', 'none'
	MissingImage: "../test/missing.jpeg'",
	WaitTime:     11,
	ProposedPlacements: []ImagePlacement{
		{Source: "test/ipcam/snowshed/image/0",
			Resize: "height",
			Y:      30,
		},
		{Source: "test/ipcam/kelowna1/image/0",
			Resize: "width",
			Y:      0,
		},
	},
}

var config2 = &MontageConfig{
	ID:           "screen2",
	Name:         "Screen 2",
	Filename:     TestMontageFile,
	Width:        1680,
	Height:       1050,
	Rows:         2,
	Border:       4,
	Resize:       MontageResizeNone,
	MissingImage: "../test/missing.jpeg'",
	WaitTime:     11,
	ProposedPlacements: []ImagePlacement{
		{Source: "test/ipcam/snowshed/image/0",
			Resize: "height",
		},
		{Source: "test/ipcam/kelowna1/image/0",
			Resize: "width",
		},
		{Source: "test/ipcam/cam6/image/0",
			Resize: "height",
		},
		{Source: "test/ipcam/cam7/image/0",
			Resize: "width",
		},
	},
}

// Create a montage and layout 2 images onto its canvas
func TestMontageLayout(t *testing.T) {
	os.Remove(TestMontageFile)
	pub, _ := publisher.NewAppPublisher(AppID, configFolder, appConfig, false)
	app := NewWallpaperApp(appConfig, pub)
	montage := app.CreateWallpaper(&config1)
	assert.NotNil(t, montage)

	assert.Len(t, montage.actualPlacement, 2, "Expected 2 image placements for this montage")
	assert.Equal(t, 1680, montage.canvas.Rect.Max.X)

	image, _ := ioutil.ReadFile("../test/camera-sshed.jpeg")
	montage.UpdateImage("test/ipcam/snowshed/image/0", image)
	image, _ = ioutil.ReadFile("../test/camera-zkioskn.jpeg")
	montage.UpdateImage("test/ipcam/kelowna1/image/0", image)
	err := montage.WriteToFile(TestMontageFile)
	assert.NoError(t, err)
}

// Combine 4 test images into a montage and save as tmp/montage2.jpeg
func TestImageUpdate(t *testing.T) {
	pub, _ := publisher.NewAppPublisher(AppID, configFolder, appConfig, false)
	app := NewWallpaperApp(appConfig, pub)
	montage := app.CreateWallpaper(config2)

	image, _ := ioutil.ReadFile("../test/camera-sshed.jpeg")
	montage.UpdateImage("test/ipcam/snowshed/image/0", image)
	image, _ = ioutil.ReadFile("../test/camera-zkioskn.jpeg")
	montage.UpdateImage("test/ipcam/kelowna1/image/0", image)
	image, _ = ioutil.ReadFile("../test/camera-cam6.jpeg")
	montage.UpdateImage("test/ipcam/cam6/image/0", image)
	image, _ = ioutil.ReadFile("../test/camera-cam7.jpeg")
	montage.UpdateImage("test/ipcam/cam7/image/0", image)

	assert.Equal(t, 4, montage.UpdateCount, "4 Updates expected")

	_ = os.Remove(TestMontageFile)
	err := montage.WriteToFile(TestMontageFile)
	assert.NoError(t, err)
	assert.FileExists(t, TestMontageFile, "Montage file missing")
}

func BenchmarkWallpaper(b *testing.B) {
	os.Remove(TestMontageFile)
	pub, _ := publisher.NewAppPublisher(AppID, configFolder, appConfig, false)
	app := NewWallpaperApp(appConfig, pub)
	montage := app.CreateWallpaper(config2)

	image1, _ := ioutil.ReadFile("../test/camera-sshed.jpeg")
	image2, _ := ioutil.ReadFile("../test/camera-zkioskn.jpeg")
	image3, _ := ioutil.ReadFile("../test/camera-cam6.jpeg")
	//image4, _ := ioutil.ReadFile("test/camera-cam7.jpeg")
	image4, _ := ioutil.ReadFile("../test/circles.png")

	t1 := time.Now()
	for i := 0; i < 25; i++ {
		montage.UpdateImage("test/ipcam/snowshed/image/0", image1)
		montage.UpdateImage("test/ipcam/kelowna1/image/0", image2)
		montage.UpdateImage("test/ipcam/cam6/image/0", image3)
		montage.UpdateImage("test/ipcam/cam7/image/0", image4)
		data, err := montage.ExportMontageAsJPEG()
		assert.NoError(b, err)
		assert.NotNil(b, data)
	}
	t2 := time.Now()
	duration := t2.Sub(t1)
	err := montage.WriteToFile(TestMontageFile)
	assert.NoError(b, err)

	b.Logf("UpdateImage Duration: %.0f msec per update of 4", (duration.Seconds() / 25 * 1000))

	assert.Equal(b, 100, montage.UpdateCount, "Updates expected")
}
