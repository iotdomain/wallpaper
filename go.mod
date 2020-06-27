module wallpaper

go 1.14

require (
	github.com/google/go-cmp v0.5.0 // indirect
	github.com/iotdomain/iotdomain-go v0.0.0-20200623050445-f9200737c15b
	github.com/sirupsen/logrus v1.6.0
	github.com/disintegration/imaging v1.6.2
	github.com/pixiv/go-libjpeg v0.0.0-20190822045933-3da21a74767d
	github.com/stretchr/testify v1.6.0
)

// Temporary for testing iotdomain-go
replace github.com/iotdomain/iotdomain-go => ../iotdomain-go
