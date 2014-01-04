package library

type SearchResult struct {
}

type Library struct {
	paths []string
}

func (lib *Library) Search(searchTerm string) []SearchResult {
	return []SearchResult{}
}

func (lib *Library) GetFilePath(trackID int64) string {
	return ""
}

func (lib *Library) Scan() error {
	return nil
}
