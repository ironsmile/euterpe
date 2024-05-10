package subsonic

import (
    "net/http"
)

func (s *subsonic) unstar(w http.ResponseWriter, req *http.Request) {
    favs, ok := s.parseStarArguments(w, req)
    if !ok {
        return
    }

    if err := s.lib.RemoveFavourite(req.Context(), favs); err != nil {
        resp := responseError(errCodeGeneric, err.Error())
        encodeResponse(w, req, resp)
        return
    }

    encodeResponse(w, req, responseOk())
}
