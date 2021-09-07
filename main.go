// A web interface to your media library.
//
// This file is only here to make installing with go get easier.
// At the moment I don't see any other way to stash my source in the src directory
// instead of dumping it in the project root.
package main

import (
	"embed"
	"fmt"
	"io/fs"
	"os"

	"github.com/ironsmile/euterpe/src"
)

var (
	// sqlFilesFS the migrations directory which contains SQL
	// migrations for sql-migrate and the initial schema. If the
	// embedded directory name changes, remember to change it in
	// main() too.
	//
	//go:embed sqls
	sqlFilesFS embed.FS

	// httpRootFS is the directory which contains the
	// static files served by HTTPMS. If the embedded directory
	// name changes remember to change it in main() too.
	//
	//go:embed http_root
	httpRootFS embed.FS

	// htmlTemplatesFS is the directory with HTML templates. If
	// the embedded directory name changes, remember to change it
	// in main() too.
	//
	//go:embed templates
	htmlTemplatesFS embed.FS
)

func main() {
	fsRoot, err := fs.Sub(httpRootFS, "http_root")
	if err != nil {
		fmt.Fprintf(os.Stderr, "loading HTTP root subFS: %s\n", err)
		os.Exit(1)
	}

	tpls, err := fs.Sub(htmlTemplatesFS, "templates")
	if err != nil {
		fmt.Fprintf(os.Stderr, "loading templates subFS: %s\n", err)
		os.Exit(1)
	}

	sqls, err := fs.Sub(sqlFilesFS, "sqls")
	if err != nil {
		fmt.Fprintf(os.Stderr, "loading sqls subFS: %s\n", err)
		os.Exit(1)
	}

	src.Main(fsRoot, tpls, sqls)
}
