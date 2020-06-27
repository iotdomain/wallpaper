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

const TestMontage1File = "test/montage1.jpeg"

const TestMontage2File = "test/montage2.jpeg"

var appConfig *WallpaperAppConfig = &WallpaperAppConfig{}

var config1 = MontageConfig{
	ID:           "screen1",
	Name:         "Screen 1",
	Filename:     "test/montage1.jpeg",
	Width:        1680,
	Height:       1080,
	Rows:         1,
	Border:       1,
	Resize:       MontageResizeNone, // 'height', 'width', 'crop', 'scale', 'none'
	MissingImage: "test/missing.jpeg'",
	WaitTime:     11,
}
var images1 = []*ImagePlacement{
	{Source: "test/ipcam/snowshed/image/0",
		Resize: "height",
		Y:      30,
	},
	{Source: "test/ipcam/kelowna1/image/0",
		Resize: "width",
		Y:      0,
	},
}

var config2 = &MontageConfig{
	ID:           "screen2",
	Name:         "Screen 2",
	Filename:     "test/montage2.jpeg",
	Width:        1680,
	Height:       1050,
	Rows:         2,
	Border:       4,
	Resize:       MontageResizeNone,
	MissingImage: "test/missing.jpeg'",
	WaitTime:     11,
}
var images2 = []*ImagePlacement{
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
}

// Create a montage and layout 2 images onto its canvas
func TestMontageLayout(t *testing.T) {
	pub, _ := publisher.NewAppPublisher(AppID, configFolder, cacheFolder, appConfig, false)
	app := NewWallpaperApp(appConfig, pub)
	wallpaper := app.AddWallpaper(&config1, images1)
	assert.NotNil(t, wallpaper)

	assert.Len(t, wallpaper.ImagePlacement, 2, "Expected 2 image layouts for this montage")
	assert.Equal(t, 1680, wallpaper.canvas.Rect.Max.X)

	image, _ := ioutil.ReadFile("test/camera-sshed.jpeg")
	wallpaper.OnImageUpdate("test/ipcam/snowshed/image/0", image)
	image, _ = ioutil.ReadFile("test/camera-zkioskn.jpeg")
	wallpaper.OnImageUpdate("test/ipcam/kelowna1/image/0", image)
	err := wallpaper.WriteToFile(TestMontage1File)
	assert.NoError(t, err)
}

// Combine 4 test images into a montage and save as tmp/montage2.jpeg
func TestImageUpdate(t *testing.T) {
	pub, _ := publisher.NewAppPublisher(AppID, configFolder, cacheFolder, appConfig, false)
	app := NewWallpaperApp(appConfig, pub)
	wallpaper := app.AddWallpaper(config2, images2)

	image, _ := ioutil.ReadFile("test/camera-sshed.jpeg")
	wallpaper.OnImageUpdate("test/ipcam/snowshed/image/0", image)
	image, _ = ioutil.ReadFile("test/camera-zkioskn.jpeg")
	wallpaper.OnImageUpdate("test/ipcam/kelowna1/image/0", image)
	image, _ = ioutil.ReadFile("test/camera-cam6.jpeg")
	wallpaper.OnImageUpdate("test/ipcam/cam6/image/0", image)
	image, _ = ioutil.ReadFile("test/camera-cam7.jpeg")
	wallpaper.OnImageUpdate("test/ipcam/cam7/image/0", image)

	assert.Equal(t, 4, wallpaper.UpdateCount, "4 Updates expected")

	_ = os.Remove(TestMontage2File)
	err := wallpaper.WriteToFile(TestMontage2File)
	assert.NoError(t, err)
	assert.FileExists(t, TestMontage2File, "Montage file missing")
}

func BenchmarkWallpaper(b *testing.B) {
	pub, _ := publisher.NewAppPublisher(AppID, configFolder, cacheFolder, appConfig, false)
	app := NewWallpaperApp(appConfig, pub)
	wallpaper := app.AddWallpaper(config2, images2)

	image1, _ := ioutil.ReadFile("test/camera-sshed.jpeg")
	image2, _ := ioutil.ReadFile("test/camera-zkioskn.jpeg")
	image3, _ := ioutil.ReadFile("test/camera-cam6.jpeg")
	//image4, _ := ioutil.ReadFile("test/camera-cam7.jpeg")
	image4, _ := ioutil.ReadFile("test/circles.png")

	t1 := time.Now()
	for i := 0; i < 25; i++ {
		wallpaper.OnImageUpdate("test/ipcam/snowshed/image/0", image1)
		wallpaper.OnImageUpdate("test/ipcam/kelowna1/image/0", image2)
		wallpaper.OnImageUpdate("test/ipcam/cam6/image/0", image3)
		wallpaper.OnImageUpdate("test/ipcam/cam7/image/0", image4)
		data, err := wallpaper.ExportMontage()
		assert.NoError(b, err)
		assert.NotNil(b, data)
	}
	t2 := time.Now()
	duration := t2.Sub(t1)
	err := wallpaper.WriteToFile("test/montage.jpeg")
	assert.NoError(b, err)

	b.Logf("OnImageUpdate Duration: %.0f msec per update of 4 with export", (duration.Seconds() / 25 * 1000))

	assert.Equal(b, 100, wallpaper.UpdateCount, "Updates expected")
}