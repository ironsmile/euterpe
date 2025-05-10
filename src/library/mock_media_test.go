package library

import "time"

// MockMedia is a type used for testing the media insertion methods
type MockMedia struct {
	artist  string
	album   string
	title   string
	track   int
	length  time.Duration
	year    int
	bitrate int
}

// Artist satisfies the MediaFile interface and just returns the object attribute.
func (m *MockMedia) Artist() string {
	return m.artist
}

// Album satisfies the MediaFile interface and just returns the object attribute.
func (m *MockMedia) Album() string {
	return m.album
}

// Title satisfies the MediaFile interface and just returns the object attribute.
func (m *MockMedia) Title() string {
	return m.title
}

// Track satisfies the MediaFile interface and just returns the object attribute.
func (m *MockMedia) Track() int {
	return m.track
}

// Length satisfies the MediaFile interface and just returns the object attribute.
func (m *MockMedia) Length() time.Duration {
	return m.length
}

// Year satisfies the MediaFile interface. Returns the object attribute or a default
// value if one is not set.
func (m *MockMedia) Year() int {
	return m.year
}

// Bitrate satisfies the MediaFile interface. Returns the object attribute or a default
// value if one is not set.
func (m *MockMedia) Bitrate() int {
	if m.bitrate == 0 {
		return 256
	}
	return m.bitrate
}
