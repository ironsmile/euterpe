package subsonic

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/ironsmile/euterpe/src/radio"
)

func (s *subsonic) createInternetRadioStation(w http.ResponseWriter, req *http.Request) {
	station, ok := s.radioStationFromParams(w, req)
	if !ok {
		// Response has already been written by radioStationFromParams.
		return
	}

	if _, err := s.radio.Create(req.Context(), station); err != nil {
		resp := responseError(
			errCodeGeneric,
			fmt.Sprintf("failed to create radio station: %s", err),
		)
		encodeResponse(w, req, resp)
		return
	}

	encodeResponse(w, req, responseOk())
}

// radioStationFromParams parses a radio station from the `req` form parameters and
// returns it. When no errors the second return value is `true`.
//
// In case of error or missing information the function sends the appropriate response
// and returns `false` as its second return value.
func (s *subsonic) radioStationFromParams(
	w http.ResponseWriter,
	req *http.Request,
) (radio.Station, bool) {
	name := req.Form.Get("name")
	stream := req.Form.Get("streamUrl")
	homepage := req.Form.Get("homepageUrl")

	if name == "" || stream == "" {
		resp := responseError(
			errCodeMissingParameter,
			"`name` and `streamUrl` parameters are rquired",
		)
		encodeResponse(w, req, resp)
		return radio.Station{}, false
	}

	station := radio.Station{
		Name: name,
	}

	streamURL, err := url.Parse(stream)
	if err != nil {
		resp := responseError(
			errCodeMissingParameter,
			fmt.Sprintf("malformed `streamUrl` parameter: %s", err),
		)
		encodeResponse(w, req, resp)
		return station, false
	}
	if streamURL.Scheme != "http" && streamURL.Scheme != "https" {
		resp := responseError(
			errCodeMissingParameter,
			"only HTTP and HTTPS URLs are supported for the `streamUrl` parameter",
		)
		encodeResponse(w, req, resp)
		return station, false
	}

	station.StreamURL = *streamURL

	if homepage != "" {
		homepageURL, err := url.Parse(homepage)
		if err != nil {
			resp := responseError(
				errCodeMissingParameter,
				fmt.Sprintf("malformed `homepage` parameter: %s", err),
			)
			encodeResponse(w, req, resp)
			return station, false
		}
		if homepageURL.Scheme != "http" && homepageURL.Scheme != "https" {
			resp := responseError(
				errCodeMissingParameter,
				"only HTTP and HTTPS URLs are supported for the `homepage` parameter",
			)
			encodeResponse(w, req, resp)
			return station, false
		}

		station.HomePage = homepageURL
	}

	return station, true
}
