package caa

// These constants can be used to indicate the required image size to the various image retrieval methods
const (
	// 250px
	ImageSizeSmall = iota
	// 500px
	ImageSizeLarge
	ImageSizeOriginal
	// 250px
	ImageSize250 = ImageSizeSmall
	// 500px
	ImageSize500 = ImageSizeLarge
	// 1200px
	ImageSize1200
)
