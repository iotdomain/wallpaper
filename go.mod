module wallpaper

go 1.14

require (
	github.com/disintegration/imaging v1.6.2
	github.com/iotdomain/iotdomain-go v0.0.0-20200928060533-3e6dc24cf1bb
	github.com/pixiv/go-libjpeg v0.0.0-20190822045933-3da21a74767d
	github.com/sirupsen/logrus v1.7.0
	github.com/stretchr/testify v1.6.1
	golang.org/x/image v0.0.0-20200927104501-e162460cd6b5 // indirect
)

// Temporary for testing iotdomain-go
replace github.com/iotdomain/iotdomain-go => ../iotdomain-go
