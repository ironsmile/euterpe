package subsonic

import "net/http"

func (s *subsonic) getOpenSubsonicExtensions(w http.ResponseWriter, req *http.Request) {
	resp := osExtensionsResponse{
		baseResponse: responseOk(),
		Extensions: []osExtensin{
			{
				Name:     "formPost",
				Versions: []int{1},
			},
		},
	}

	encodeResponse(w, req, resp)
}

type osExtensionsResponse struct {
	baseResponse

	Extensions []osExtensin `xml:"openSubsonicExtensions" json:"openSubsonicExtensions"`
}

type osExtensin struct {
	Name     string `xml:"name,attr" json:"name"`
	Versions []int  `xml:"versions" json:"versions"`
}
