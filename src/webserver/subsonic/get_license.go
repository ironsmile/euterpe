package subsonic

import (
	"encoding/xml"
	"net/http"
	"time"
)

func (s *subsonic) getLicense(w http.ResponseWriter, _ *http.Request) {
	resp := licenseResponse{
		baseResponse: responseOk(),
	}

	// Always valid for 10 years into the future.
	t := time.Now().Add(365 * 10 * 24 * time.Hour)

	resp.License.Valid = "true"
	resp.License.Email = "always-valid@listen-to-euterpe.eu"
	resp.License.LicenseExpires = t.Format(time.RFC3339)

	encodeResponse(w, resp)
}

type licenseResponse struct {
	baseResponse

	License struct {
		XMLName        xml.Name `xml:"license"`
		Valid          string   `xml:"valid,attr"`
		Email          string   `xml:"email,attr"`
		LicenseExpires string   `xml:"licenseExpires,attr"`
	}
}
