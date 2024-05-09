package subsonic

import "net/http"

func (s *subsonic) getOpenSubsonicExtensions(w http.ResponseWriter, req *http.Request) {
	resp := osExtensionsResponse{
		baseResponse: responseOk(),
		Extensions: []osExtension{
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

	Extensions []osExtension `xml:"openSubsonicExtensions" json:"openSubsonicExtensions"`
}

type osExtension struct {
	Name     string `xml:"name,attr" json:"name"`
	Versions []int  `xml:"versions" json:"versions"`
}
