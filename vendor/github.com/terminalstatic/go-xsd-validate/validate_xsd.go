// Package xsdvalidate is a go package for xsd validation that utilizes libxml2.

//The goal of this package is to preload xsd files into memory and to validate xml (fast) using libxml2, like post bodys of xml service endpoints or api routers. At the time of writing, similar packages I found on github either didn't provide error details or got stuck under load. In addition to providing error strings it also exposes some fields of libxml2 return structs.
package xsdvalidate

import "C"
import (
	"sync"
	"sync/atomic"
	"time"
)

type guard struct {
	sync.Mutex
	initialized uint32
}

func (guard *guard) isInitialized() bool {
	if atomic.LoadUint32(&guard.initialized) == 0 {
		return false
	}
	return true
}

func (guard *guard) setInitialized(b bool) {
	switch b {
	case true:
		atomic.StoreUint32(&guard.initialized, 1)
	case false:
		atomic.StoreUint32(&guard.initialized, 0)
	}
}

var g guard

// Options type for parser/validation options.
type Options uint8

// The parser options, ParsErrVerbose will slow down parsing considerably!
const (
	ParsErrDefault Options = 1 << iota // Default parser error output
	ParsErrVerbose                     // Verbose parser error output, considerably slower!
)

// Validation options for possible future enhancements.
const (
	ValidErrDefault Options = 128 << iota // Default validation error output
)

var quit chan struct{}

// Init initializes libxml2, see http://xmlsoft.org/threads.html.
func Init() error {
	g.Lock()
	defer g.Unlock()
	if g.isInitialized() {
		return Libxml2Error{errorMessage{"Libxml2 already initialized"}}
	}

	libXml2Init()
	g.setInitialized(true)
	return nil
}

// InitWithGc initializes lbxml2 with a goroutine that runs the go gc every d duration.
// Not required but might help to keep the memory footprint at bay when doing tons of validations.
func InitWithGc(d time.Duration) {
	Init()
	quit = make(chan struct{})
	go gcTicker(d, quit)
}

// Cleanup cleans up libxml2 memory and finishes gc goroutine when running.
func Cleanup() {
	g.Lock()
	defer g.Unlock()
	libXml2Cleanup()
	g.setInitialized(false)
	if quit != nil {
		quit <- struct{}{}
		quit = nil
	}
}

// NewXmlHandlerMem creates a xml handler struct.
// If an error is returned it can be of type Libxml2Error or XmlParserError.
// Always use the Free() method when done using this handler or memory will be leaking.
// The go garbage collector will not collect the allocated resources.
func NewXmlHandlerMem(inXml []byte, options Options) (*XmlHandler, error) {
	if !g.isInitialized() {
		return nil, Libxml2Error{errorMessage{"Libxml2 not initialized"}}
	}

	xPtr, err := parseXmlMem(inXml, options)
	return &XmlHandler{xPtr}, err
}

// NewXsdHandlerUrl creates a xsd handler struct.
// Always use Free() method when done using this handler or memory will be leaking.
// If an error is returned it can be of type Libxml2Error or XsdParserError.
// The go garbage collector will not collect the allocated resources.
func NewXsdHandlerUrl(url string, options Options) (*XsdHandler, error) {
	g.Lock()
	defer g.Unlock()
	if !g.isInitialized() {
		return nil, Libxml2Error{errorMessage{"Libxml2 not initialized"}}
	}
	sPtr, err := parseUrlSchema(url, options)
	return &XsdHandler{sPtr}, err
}

// NewXsdHandlerMem creates an xsd handler struct.
// Always use Free() method when done using this handler or memory will leak.
// If an error is returned it can be of type Libxml2Error or XsdParserError.
// The go garbage collector will not collect the allocated resources.
func NewXsdHandlerMem(inSchema []byte, options Options) (*XsdHandler, error) {
	g.Lock()
	defer g.Unlock()
	if !g.isInitialized() {
		return nil, Libxml2Error{errorMessage{"Libxml2 not initialized"}}
	}
	sPtr, err := parseMemSchema(inSchema, options)
	return &XsdHandler{sPtr}, err
}

// Validate validates an xmlHandler against an xsdHandler and returns a ValidationError.
// If an error is returned it is of type Libxml2Error, XsdParserError, XmlParserError or ValidationError.
// Both xmlHandler and xsdHandler have to be created first.
func (xsdHandler *XsdHandler) Validate(xmlHandler *XmlHandler, options Options) error {
	if !g.isInitialized() {
		return Libxml2Error{errorMessage{"Libxml2 not initialized"}}
	}

	if xsdHandler == nil || xsdHandler.schemaPtr == nil {
		return XsdParserError{errorMessage{"Xsd handler not properly initialized"}}

	}
	if xmlHandler == nil || xmlHandler.docPtr == nil {
		return XmlParserError{errorMessage{"Xml handler not properly initialized"}}
	}
	return validateWithXsd(xmlHandler, xsdHandler)

}

// ValidateMem validates an xml byte slice against an xsdHandler.
// If an error is returned it can be of type Libxml2Error, XsdParserError, XmlParserError or ValidationError.
// The xsdHandler has to be created first.
func (xsdHandler *XsdHandler) ValidateMem(inXml []byte, options Options) error {
	if !g.isInitialized() {
		return Libxml2Error{errorMessage{"Libxml2 not initialized"}}
	}
	if xsdHandler == nil || xsdHandler.schemaPtr == nil {
		return XsdParserError{errorMessage{"Xsd handler not properly initialized"}}

	}
	return validateBufWithXsd(inXml, options, xsdHandler)

}

// Free frees the wrapped schemaPtr, call this when this handler is not needed anymore.
func (xsdHandler *XsdHandler) Free() {
	freeSchemaPtr(xsdHandler)
}

// Free frees the wrapped xml docPtr, call this when this handler is not needed anymore.
func (xmlHandler *XmlHandler) Free() {
	freeDocPtr(xmlHandler)
}
