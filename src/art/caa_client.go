package art

import (
	"github.com/pborman/uuid"
	cca "gopkg.in/mineo/gocaa.v1"
)

//counterfeiter:generate . CAAClient

// CAAClient represents a Cover Art Archive client for getting a release front
// image.
type CAAClient interface {
	GetReleaseFront(mbid uuid.UUID, size int) (image cca.CoverArtImage, err error)
}
