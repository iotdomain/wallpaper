// Package internal with wallpaper montage
package internal

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"

	"github.com/disintegration/imaging"
	libjpeg "github.com/pixiv/go-libjpeg/jpeg"
	"github.com/sirupsen/logrus"
)

// Montage for montage of an image out of multiple parts as defined by the MontageConfig
// This holds the montage canvas in which images are written
type Montage struct {
	Config      MontageConfig // Wallpaper configuration for this montage
	UpdateCount int           // Canvas update count since last ExportMontage
	useLibJpeg  bool          // use the faster libjpeg instead of the image library to draw images on canvas.
	isActive    bool          // Montage background update is active
	//layout      []MontageImage  // Actual layout of images on canvas
	canvas *image.RGBA    // canvas to draw the montage on
	logger *logrus.Logger // module logger
	// imagePlacement []*ImagePlacement      // Configuration layout of images with their source
	resizing imaging.ResampleFilter // default method used for resizing
}

// MontageConfig containing the definition of a wallpaper
type MontageConfig struct {
	ID           string           `yaml:"ID"`                 // ID of the wallpaper
	Border       int              `yaml:"border,omitempty"`   // border around image
	Name         string           `yaml:"name"`               // montage name
	Filename     string           `yaml:"filename,omitempty"` // file to save montage image as
	Height       int              `yaml:"height,omitempty"`   // montage height
	Width        int              `yaml:"width,omitempty"`    // montage width
	WaitTime     int              `yaml:"waitTime,omitempty"` // Time to wait for updates and rebuild the montage. Default is 3 seconds
	Publish      bool             `yaml:"publish"`            // publish the resulting image
	Resize       MontageResize    `yaml:"resize,omitempty"`   // Image resize in this montage: 'crop', 'width' or 'height'. Default is height.
	Rows         int              `yaml:"rows,omitempty"`     // Number of rows to organize images in.
	MissingImage string           `yaml:"noimage,omitempty"`  // substitute for missing images, default is to keep the last image
	Images       []ImagePlacement `yaml:"images"`             // The images to montage
}

// ImagePlacement describes the placement of an image on the canvas
// The source can be a topic or file
type ImagePlacement struct {
	//Order  int           // Optional order in which to sort the images.
	Source   string        `yaml:"source"`             // Image source. Topic, file://filename, or http://url
	X        int           `yaml:"x,omitempty"`        // Optional x-offset to use instead of automatic layout. 0 is automatic
	Y        int           `yaml:"y,omitempty"`        // Optional y-offset to use instead of automatic layout. 0 is automatic
	Width    int           `yaml:"width,omitempty"`    // Optional width to use instead of automatic calculated. 0 is automatic
	Height   int           `yaml:"height,omitempty"`   // Optional height to use instead of automatic calculated. 0 is automatic
	Interval int           `yaml:"interval,omitempty"` // Interval to poll source, in case of IP camera, default is 900 seconds
	Resize   MontageResize `yaml:"resize,omitempty"`   // Optional resize to use instead of the montage setting
}

// MontageResize method of resizing
type MontageResize string

// Available resize methods for montage images
const (
	MontageResizeCrop   MontageResize = "crop"
	MontageResizeHeight MontageResize = "height"
	MontageResizeNone   MontageResize = "none"
	MontageResizeScale  MontageResize = "scale"
	MontageResizeWidth  MontageResize = "width"
)

// MakeGridLayout calculates the placement of each image in the montage based on the configuration
// returns a new layout with actual location and size of each image
// This is a simple grid layout, not optimizing for individual image sizes
func MakeGridLayout(config *MontageConfig, images []*ImagePlacement) []*ImagePlacement {

	cols := (len(images) + (config.Rows - 1)) / config.Rows
	if cols < 1 {
		cols = 1
	}
	rows := config.Rows
	// Leave room for the border
	imageWidth := 0
	imageHeight := int((config.Height-config.Border)/rows) - config.Border

	result := make([]*ImagePlacement, 0)
	x := config.Border
	y := config.Border
	index := 0
	// filename := ""
	for r := 0; r < rows; r++ {
		x = config.Border
		for c := 0; c < cols; c++ {
			if index < len(images) {
				// determine the image width in one of two ways, configured width or remaining space
				imageConfig := images[index]
				// filename = montage.getTopicFilename(imageConfig.Topic)
				resizeMethod := config.Resize
				if imageConfig.Resize != "" {
					resizeMethod = imageConfig.Resize
				}
				//if imageConfig.X > 0 {
				//	// force x-offset
				//	x = imageConfig.X
				//}
				//if imageConfig.Y > 0 {
				//	// force y-offset
				//	y = imageConfig.Y
				//}
				if imageConfig.Width > 0 {
					// force image width
					imageWidth = imageConfig.Width
				} else {
					// space evenly in remaining width. TODO: Actual imageWidth/Height is the remainder after all fixed widths
					remainingCols := cols - c
					remainingWidth := config.Width - x - config.Border*remainingCols
					imageWidth = remainingWidth / remainingCols
				}

				imageLayout := &ImagePlacement{
					Source: imageConfig.Source,
					X:      x + imageConfig.X,
					Y:      y + imageConfig.Y,
					Width:  imageWidth,
					Height: imageHeight,
					Resize: resizeMethod,
				}
				result = append(result, imageLayout)
			}
			index++
			x += imageWidth + config.Border
		}
		y += imageHeight + config.Border
	}
	return result
}

/*
* Draw the image from the layout onto the canvas at the layout position and increase UpdateCount
* This uses a 'not-found' image if the the image is not found
 */
func (montage *Montage) drawImage(img image.Image, imageLayout *ImagePlacement) error {
	// resize to fit the available space
	resizedImg := img
	switch imageLayout.Resize {
	case MontageResizeWidth:
		resizedImg = imaging.Resize(img, imageLayout.Width, 0, montage.resizing)
	case MontageResizeHeight:
		resizedImg = imaging.Resize(img, 0, imageLayout.Height, montage.resizing)
	case MontageResizeCrop:
		resizedImg = imaging.Thumbnail(img, imageLayout.Width, imageLayout.Height, montage.resizing)
	case MontageResizeScale:
		resizedImg = imaging.Resize(img, imageLayout.Width, imageLayout.Height, montage.resizing)
	case MontageResizeNone:
	default: // default is not to resize
		// resizedImg = imaging.Resize(img, 0, imageLayout.Height, montage.Resizing)
		resizedImg = img
	}

	// Embed the image centered in its place into the main montage image
	// imageSize := resizedImg.Bounds()
	// xOffset := (imageLayout.Width - imageSize.Max.X) / 2
	// if xOffset < 0 {
	// 	xOffset = 0
	// }
	// yOffset := (imageLayout.Height - imageSize.Max.Y) / 2
	// if yOffset < 0 {
	// 	yOffset = 0
	// }
	// rectangle := image.Rect(imageLayout.X+xOffset, imageLayout.Y+yOffset,
	// 	imageLayout.X+imageLayout.Width, imageLayout.Y+imageLayout.Height)
	rectangle := image.Rect(imageLayout.X, imageLayout.Y,
		imageLayout.X+imageLayout.Width, imageLayout.Y+imageLayout.Height)
	draw.Draw(montage.canvas, rectangle, resizedImg, image.ZP, draw.Src)

	montage.UpdateCount++
	return nil
}

// DrawImageIntoLayout draws the image on canvas and increase the UpdateCount
// This uses the image library which is a bit slow.
func (montage *Montage) DrawImageIntoLayout(layout *ImagePlacement, imageData []byte) error {
	// var m runtime.MemStats
	// runtime.ReadMemStats(&m)
	// logger.Info("drawImageOfTopic entry Memory: ", m.Alloc)

	buffer := bytes.NewBuffer(imageData)
	// Decode takes 85% of all cpu: https://github.com/golang/go/issues/24499
	img, imageType, err := image.Decode(buffer)

	if err != nil {
		montage.logger.Errorf("montage.DrawImageIntoLayout: Failed decoding image for montage %s: %s",
			montage.Config.Name, err)
		return err
	}
	montage.logger.Debugf("montage.DrawImageIntoLayout: Image of layout %s of type %s decoded", layout.Source, imageType)
	err = montage.drawImage(img, layout)
	return err
}

// DrawJpegIntoLayout draws the image on canvas and increase the UpdateCount
// This uses libjpeg, which is faster than the native image library
func (montage *Montage) DrawJpegIntoLayout(layout *ImagePlacement, imageData []byte) error {
	// var m runtime.MemStats
	// runtime.ReadMemStats(&m)
	// logger.Info("drawImageOfTopic entry Memory: ", m.Alloc)

	buffer := bytes.NewBuffer(imageData)
	var err error
	opts := &libjpeg.DecoderOptions{}
	img, err := libjpeg.Decode(buffer, opts)
	//img, err := prism.Decode(buffer)
	if err != nil {
		montage.logger.Errorf("montage.DrawJpegIntoLayout: Failed decoding jpeg image for montage %s: %s",
			montage.Config.Name, err)
		return err
	}
	montage.logger.Debugf("montage.DrawJpegIntoLayout: Jpeg Image of layout %s decoded", layout.Source)
	err = montage.drawImage(img, layout)
	return err
}

// CreateMontageAsJPEG retrieves the montage as JPEG image
func (montage *Montage) CreateMontageAsJPEG() ([]byte, error) {
	montage.logger.Debugf("montage.ExportMontage %s", montage.Config.Name)
	// export image as JPEG
	buf := new(bytes.Buffer)
	var err error
	if montage.useLibJpeg {
		opt := libjpeg.EncoderOptions{}
		opt.Quality = 80
		err = libjpeg.Encode(buf, montage.canvas, &opt)
	} else {
		var opt jpeg.Options
		opt.Quality = 80
		err = jpeg.Encode(buf, montage.canvas, &opt)
	}
	if err != nil {
		montage.logger.Errorf("montage.ExportMontage Error encoding canvas of montage %s: %s", montage.Config.Name, err)
		return nil, err
	}
	imageData := buf.Bytes()
	// Save the final montage image file if a filename is given

	return imageData, err
}

// NewMontage initialises a new Montage instance for the given Config
func NewMontage(config *MontageConfig, logger *logrus.Logger, useLibJpeg bool) *Montage {
	builder := Montage{
		logger:     logger,
		Config:     *config,
		isActive:   false,
		useLibJpeg: useLibJpeg,
		// baseline no resizing is 86ms
		//Resizing: imaging.Lanczos,             // good, 152ms
		//Resizing: imaging.NearestNeighbor,     // poor, 99ms
		//Resizing: imaging.Box,                 // poor, 120ms
		//Resizing: imaging.MitchellNetravali,   // excellent, 128ms
		//Resizing: imaging.Bartlett,            // good, 150ms
		//Resizing: imaging.Blackman,            // okay, 145ms
		resizing: imaging.BSpline, // perfect, 126ms
		//Resizing: imaging.Cosine,              // excellent, 154ms
		//Resizing: imaging.Gaussian,            // excellent, 128ms
		//Resizing: imaging.CatmullRom,          // excellent, 127ms
		//Resizing: imaging.Hamming,             // good, 138ms
		//Resizing: imaging.Hann,                // good, 140ms
		//Resizing: imaging.Hermite,             // okay, 126ms
		//Resizing: imaging.Linear,              // good, 119ms
		//Resizing: imaging.Welch,               // good, 147ms

		// setup the canvas to draw the images onto
		canvas: image.NewRGBA(image.Rect(0, 0, config.Width, config.Height)),
	}
	rgbaBlack := color.NRGBA{R: 0, G: 0, B: 0, A: 0}
	draw.Draw(builder.canvas, builder.canvas.Bounds(), &image.Uniform{C: rgbaBlack}, image.ZP, draw.Src)
	return &builder
}
