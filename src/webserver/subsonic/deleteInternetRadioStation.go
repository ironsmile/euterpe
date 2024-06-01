package subsonic

import (
    "errors"
    "fmt"
    "net/http"
    "strconv"

    "github.com/ironsmile/euterpe/src/radio"
)

func (s *subsonic) deleteInternetRadioStation(w http.ResponseWriter, req *http.Request) {
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

    if err := s.radio.Delete(req.Context(), idInt); errors.Is(err, radio.ErrNotFound) {
        resp := responseError(
            errCodeNotFound,
            "radio station not found",
        )
        encodeResponse(w, req, resp)
        return
    } else if err != nil {
        resp := responseError(
            errCodeGeneric,
            fmt.Sprintf("could not delete radio station: %s", err),
        )
        encodeResponse(w, req, resp)
        return
    }

    encodeResponse(w, req, responseOk())
}
