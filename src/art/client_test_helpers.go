package art

// SetCAAClient sets the underlying CAAClient which will be used by the Client. Only
// useful for tests.
func (c *Client) SetCAAClient(caac CAAClient) {
	c.caaClient = caac
}

// SetMusicBrainzAPIURL sets the MusicBrainz API URL. Only useful for tests.
func (c *Client) SetMusicBrainzAPIURL(apiURL string) {
	c.musicBrainzAPIHost = apiURL
}

// SetDiscogsAPIURL sets the Discogs API URL. Only useful for tests.
func (c *Client) SetDiscogsAPIURL(apiURL string) {
	c.discogsAPIHost = apiURL
}
