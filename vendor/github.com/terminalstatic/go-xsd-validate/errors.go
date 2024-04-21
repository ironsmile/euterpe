package xsdvalidate

import (
	"fmt"
	"strings"
)

// Common String and Error implementations.
type errorMessage struct {
	Message string
}

// Implementation of the Stringer Interface.
func (e errorMessage) String() string {
	return e.Message
}

// Implementation of the Error Interface.
func (e errorMessage) Error() string {
	return e.String()
}

// Libxml2Error is returned when a Libxm2 initialization error occured.
type Libxml2Error struct {
	errorMessage
}

// XmlParserError is returned when xml parsing caused error(s).
type XmlParserError struct {
	errorMessage
}

// XsdParserError is returned when xsd parsing caused a error(s).
type XsdParserError struct {
	errorMessage
}

// StructError is a subset of libxml2 xmlError struct.
type StructError struct {
	Code     int
	Message  string
	Level    int
	Line     int
	NodeName string
}

// ValidationError is returned when xsd validation caused an error, to access the fields of the Errors slice use type assertion (see example).
type ValidationError struct {
	Errors []StructError
}

// Implementation of the Stringer interface. Aggregates line numbers and messages of the Errors slice.
func (e ValidationError) String() string {
	var em string
	for _, eelem := range e.Errors {
		em = em + fmt.Sprintf("%d: %s\n", eelem.Line, eelem.Message)
	}
	return strings.TrimRight(em, "\n")
}

// Implementation of the Error interface.
func (e ValidationError) Error() string {
	return e.String()
}
