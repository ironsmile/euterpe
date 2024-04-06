package subsonic

import (
	"encoding/xml"
	"fmt"
	"net/http"

	"github.com/ironsmile/euterpe/src/version"
)

func encodeResponse(w http.ResponseWriter, resp any) {
	enc := xml.NewEncoder(w)
	enc.Indent("", "\t")

	if err := enc.Encode(resp); err != nil {
		errMsg := fmt.Sprintf("faild to encode XML: %s", err)
		http.Error(w, errMsg, http.StatusInternalServerError)
		return
	}
}

type baseResponse struct {
	XMLName xml.Name `xml:"subsonic-response"`
	Status  string   `xml:"status,attr"`
	Version string   `xml:"version,attr"`
}

func responseOk() baseResponse {
	return baseResponse{
		Status:  "ok",
		Version: version.Version,
	}
}

func responseFailed() baseResponse {
	return baseResponse{
		Status:  "failed",
		Version: version.Version,
	}
}

type errorResponse struct {
	baseResponse

	Error errorElement
}

type errorElement struct {
	XMLName xml.Name `xml:"error"`
	Code    int      `xml:"code,attr"`
	Message string   `xml:"message,attr"`
}

func responseError(code int, msg string) errorResponse {
	return errorResponse{
		baseResponse: responseFailed(),
		Error: errorElement{
			Code:    code,
			Message: msg,
		},
	}
}
