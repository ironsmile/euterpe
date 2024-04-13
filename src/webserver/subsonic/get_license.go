package subsonic

import (
	"net/http"
	"time"
)

func (s *subsonic) getLicense(w http.ResponseWriter, req *http.Request) {
	resp := licenseResponse{
		baseResponse: responseOk(),
	}

	// Always valid for 10 years into the future.
	t := time.Now().Add(365 * 10 * 24 * time.Hour)

	resp.License.Valid = "true"
	resp.License.Email = "always-valid@listen-to-euterpe.eu"
	resp.License.LicenseExpires = t.Format(time.RFC3339)

	encodeResponse(w, req, resp)
}

type licenseResponse struct {
	baseResponse

	License struct {
		Valid          string `xml:"valid,attr" json:"valid"`
		Email          string `xml:"email,attr" json:"email"`
		LicenseExpires string `xml:"licenseExpires,attr" json:"licenseExpires"`
	} `xml:"license" json:"license"`
}
