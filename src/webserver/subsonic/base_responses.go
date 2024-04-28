package subsonic

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/http"

	"github.com/ironsmile/euterpe/src/version"
)

func encodeResponse(w http.ResponseWriter, req *http.Request, resp any) {
	if req.Form.Get("f") == "json" {
		encodeResponseJSON(w, req, resp)
		return
	}

	encodeResponseXML(w, req, resp)
}

func encodeResponseJSON(w http.ResponseWriter, _ *http.Request, resp any) {
	w.Header().Set("Content-Type", "application/json")

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")

	if err := enc.Encode(jsonResponse{Response: resp}); err != nil {
		errMsg := fmt.Sprintf("failed to encode JSON: %s", err)
		http.Error(w, errMsg, http.StatusInternalServerError)
		return
	}
}

func encodeResponseXML(w http.ResponseWriter, _ *http.Request, resp any) {
	w.Header().Set("Content-Type", "application/xml")

	enc := xml.NewEncoder(w)
	enc.Indent("", "  ")

	if err := enc.Encode(resp); err != nil {
		errMsg := fmt.Sprintf("failed to encode XML: %s", err)
		http.Error(w, errMsg, http.StatusInternalServerError)
		return
	}
}

type jsonResponse struct {
	Response any `json:"subsonic-response"`
}

type baseResponse struct {
	XMLName       xml.Name `xml:"subsonic-response" json:"-"`
	XMLNS         string   `xml:"xmlns,attr" json:"-"`
	Status        string   `xml:"status,attr" json:"status"`
	Version       string   `xml:"version,attr" json:"version"`
	Type          string   `xml:"type,attr" json:"type"`
	ServerVersion string   `xml:"serverVersion,attr" json:"serverVersion"`
	OpenSubsonic  bool     `xml:"openSubsonic,attr" json:"openSubsonic"`
}

func responseOk() baseResponse {
	return baseResponse{
		XMLNS:         `http://subsonic.org/restapi`,
		Status:        "ok",
		Version:       "1.16.1",
		Type:          "euterpe",
		ServerVersion: version.Version,
		OpenSubsonic:  true,
	}
}

func responseFailed() baseResponse {
	return baseResponse{
		XMLNS:         `http://subsonic.org/restapi`,
		Status:        "failed",
		Version:       "1.16.1",
		Type:          "euterpe",
		ServerVersion: version.Version,
		OpenSubsonic:  true,
	}
}

type errorResponse struct {
	baseResponse

	Error errorElement `xml:"error" json:"error"`
}

type errorElement struct {
	Code    apiErrorCode `xml:"code,attr" json:"code"`
	Message string       `xml:"message,attr" json:"message"`
}

func responseError(code apiErrorCode, msg string) errorResponse {
	return errorResponse{
		baseResponse: responseFailed(),
		Error: errorElement{
			Code:    code,
			Message: msg,
		},
	}
}
