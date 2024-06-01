package subsonic

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/ironsmile/euterpe/src/radio"
)

func (s *subsonic) updateInternetRadioStation(w http.ResponseWriter, req *http.Request) {
	station, ok := s.radioStationFromParams(w, req)
	if !ok {
		// Response has already been written by radioStationFromParams.
		return
	}

	id := req.Form.Get("id")
	if id == "" {
		resp := responseError(
			errCodeMissingParameter,
			"the parameter `id` is required",
		)
		encodeResponse(w, req, resp)
		return
	}

	idInt, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		resp := responseError(
			errCodeNotFound,
			fmt.Sprintf("could not parse radio station ID: %s", err),
		)
		encodeResponse(w, req, resp)
		return
	}

	station.ID = idInt
	if err := s.radio.Replace(req.Context(), station); errors.Is(err, radio.ErrNotFound) {
		resp := responseError(
			errCodeNotFound,
			"radio station not found",
		)
		encodeResponse(w, req, resp)
		return
	} else if err != nil {
		resp := responseError(
			errCodeGeneric,
			fmt.Sprintf("could not update radio station: %s", err),
		)
		encodeResponse(w, req, resp)
		return
	}

	encodeResponse(w, req, responseOk())
}
