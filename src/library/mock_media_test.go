package library

import "time"

// MockMedia is a type used for testing the media insertion methods
type MockMedia struct {
	artist string
	album  string
	title  string
	track  int
	length time.Duration
}

// Artist satisfiees the MediaFile interface and just returns the objec attribute
func (m *MockMedia) Artist() string {
	return m.artist
}

// Album satisfiees the MediaFile interface and just returns the objec attribute
func (m *MockMedia) Album() string {
	return m.album
}

// Title satisfiees the MediaFile interface and just returns the objec attribute
func (m *MockMedia) Title() string {
	return m.title
}

// Track satisfiees the MediaFile interface and just returns the objec attribute
func (m *MockMedia) Track() int {
	return m.track
}

// Length satisfiees the MediaFile interface and just returns the objec attribute
func (m *MockMedia) Length() time.Duration {
	return m.length
}
