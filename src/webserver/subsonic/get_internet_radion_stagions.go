package subsonic

import "net/http"

func (s *subsonic) getInternetRadionStations(w http.ResponseWriter, req *http.Request) {
	stations, err := s.radio.GetAll(req.Context())
	if err != nil {
		resp := responseError(errCodeGeneric, err.Error())
		encodeResponse(w, req, resp)
		return
	}

	resp := getStationsResponse{
		baseResponse: responseOk(),
	}

	for _, station := range stations {
		resp.Result.Stations = append(resp.Result.Stations, fromRadioStation(station))
	}
	encodeResponse(w, req, resp)
}

type getStationsResponse struct {
	baseResponse

	Result xsdInternetRadioStations `xml:"internetRadioStations" json:"internetRadioStations"`
}
