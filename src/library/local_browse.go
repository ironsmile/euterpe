package library

import (
	"fmt"
	"log"
)

// BrowseArtists implements the Library interface for the local library by getting artists from
// the database ordered by their name.
func (lib *LocalLibrary) BrowseArtists(page, perPage uint) ([]Artist, int) {
	var output []Artist

	artistsCount := lib.getTableSize("artists")
	rows, err := lib.db.Query(`
        SELECT
            ar.id,
            ar.name
        FROM
            artists ar
        ORDER BY
            ar.name ASC
        LIMIT
            ?, ?
    `, page*perPage, perPage)

	if err != nil {
		log.Printf("Query for browsing artists not successful: %s\n", err.Error())
		return output, artistsCount
	}

	defer rows.Close()
	for rows.Next() {
		var res Artist
		rows.Scan(&res.ID, &res.Name)
		output = append(output, res)
	}

	return output, artistsCount
}

// BrowseAlbums implements the Library interface for the local library by getting albums from
// the database ordered by their name.
func (lib *LocalLibrary) BrowseAlbums(page, perPage uint) ([]Album, int) {
	var output []Album

	albumsCount := lib.getTableSize("albums")
	rows, err := lib.db.Query(`
        SELECT
            al.id,
            al.name as album_name,
            CASE WHEN (SELECT COUNT(DISTINCT artist_id) FROM tracks WHERE album_id = al.id) = 1
            THEN (SELECT ar.name FROM tracks t LEFT JOIN artists ar ON ar.id = t.artist_id
                    WHERE t.album_id = al.id LIMIT 1)
            ELSE "Various Artists"
            END AS arist_name
        FROM
            albums al
        ORDER BY
            al.name ASC
        LIMIT
            ?, ?
    `, page*perPage, perPage)

	if err != nil {
		log.Printf("Query for browsing albums not successful: %s\n", err.Error())
		return output, albumsCount
	}

	defer rows.Close()
	for rows.Next() {
		var res Album
		rows.Scan(&res.ID, &res.Name, &res.Artist)
		output = append(output, res)
	}

	return output, albumsCount
}

func (lib *LocalLibrary) getTableSize(table string) int {
	smt, err := lib.db.Prepare(fmt.Sprintf(`
        SELECT
            COUNT(*) as cnt
        FROM
            %s
    `, table))

	var count int

	if err != nil {
		log.Printf("Query for getting %s count not prepared: %s\n", table, err.Error())
		return count
	}

	err = smt.QueryRow().Scan(&count)

	if err != nil {
		log.Printf("Query for getting %s count not successful: %s\n", table, err.Error())
		return 0
	}

	return count
}
