package subsonic

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"
)

func (s *subsonic) scrobble(w http.ResponseWriter, req *http.Request) {
	if submission := req.Form.Get("submission"); submission == "false" {
		// This is for setting "now playing" and should not increase the
		// play count and other track stats. I haven't decided what to do
		// with this so at the moment it is a no-op.
		encodeResponse(w, req, responseOk())
		return
	}

	ids := req.Form["id"]
	if len(ids) == 0 {
		resp := responseError(errCodeMissingParameter, "no track ID set")
		encodeResponse(w, req, resp)
		return
	}

	var idInts []int64
	for _, idString := range ids {
		trackID, err := strconv.ParseInt(idString, 10, 64)
		if err != nil || !isTrackID(trackID) {
			if req.Form.Get("c") == "substreamer" {
				// Substreamer does not handle "not found" errors well. It starts to
				// retry the request to scrobble every second without ever stopping. No
				// further scrobbling is done for other tasks as well, I think. So
				// for it we ignore this type of errors and do nothing with the offending
				// IDs.
				continue
			}

			resp := responseError(
				errCodeNotFound,
				fmt.Sprintf("track ID '%s' not found", idString),
			)
			encodeResponse(w, req, resp)
			return
		}

		idInts = append(idInts, toTrackDBID(trackID))
	}

	ctx := req.Context()
	scrobbleTime := time.Now()
	if timeArg := req.Form.Get("time"); timeArg != "" {
		unixTimeMs, err := strconv.ParseInt(timeArg, 10, 64)
		if err != nil || unixTimeMs <= 0 {
			resp := responseError(
				errCodeGeneric,
				"bad `time` in parameters. It must be a positive int.",
			)
			encodeResponse(w, req, resp)
			return
		}

		scrobbleTime = time.Unix(unixTimeMs/1000, 0)
	}

	for _, trackID := range idInts {
		err := s.lib.RecordTrackPlay(ctx, trackID, scrobbleTime)
		if err != nil {
			log.Printf("failed to update track %d stats: %s", trackID, err)
			resp := responseError(errCodeGeneric, err.Error())
			encodeResponse(w, req, resp)
			return
		}
	}

	encodeResponse(w, req, responseOk())
}
