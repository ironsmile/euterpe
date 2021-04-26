package webserver

import (
    "context"
    "fmt"
    "io"
    "log"
    "net/http"
    "os"
    "strconv"
    "time"

    "github.com/gorilla/mux"

    "github.com/ironsmile/httpms/src/library"
)

// ArtstImageHandler is a http.Handler which provides CRUD operations for
// artist images.
type ArtstImageHandler struct {
    imageManager library.ArtistImageManager
}

// ServeHTTP is required by the http.Handler's interface
func (aih ArtstImageHandler) ServeHTTP(writer http.ResponseWriter, req *http.Request) {
    vars := mux.Vars(req)

    idString, ok := vars["artistID"]
    if !ok {
        http.NotFoundHandler().ServeHTTP(writer, req)
        return
    }

    id, err := strconv.ParseInt(idString, 10, 64)
    if err != nil {
        writer.WriteHeader(http.StatusBadRequest)
        fmt.Fprintf(writer, "Bad request. Parsing artistID: %s\n", err)
        return
    }

    if req.Method == http.MethodDelete {
        err = aih.remove(writer, req, id)
    } else if req.Method == http.MethodPut {
        err = aih.upload(writer, req, id)
    } else {
        err = aih.find(writer, req, id)
    }

    if err != nil {
        writer.WriteHeader(http.StatusInternalServerError)
        if _, err := writer.Write([]byte(err.Error())); err != nil {
            log.Printf("error writing body in ArtstImageHandler: %s", err)
        }
    }
}

// Actually searches through the library for the image of an artist and serves
// it as a raw image
func (aih ArtstImageHandler) find(
    writer http.ResponseWriter,
    req *http.Request,
    id int64,
) error {

    ctx, cancel := context.WithTimeout(req.Context(), 5*time.Minute)
    defer cancel()

    imgSize := library.OriginalImage
    if req.URL.Query().Get("size") == "small" {
        imgSize = library.SmallImage
    }

    imgReader, err := aih.imageManager.FindAndSaveArtistImage(ctx, id, imgSize)

    if err == library.ErrArtworkNotFound || os.IsNotExist(err) {
        writer.WriteHeader(http.StatusNotFound)
        fmt.Fprintln(writer, "404 image not found")
        return nil
    }

    if err != nil {
        log.Printf("Error finding artist %d artwork: %s\n", id, err)
        return err
    }

    defer imgReader.Close()

    writer.Header().Set("Cache-Control", "max-age=604800")
    _, err = io.Copy(writer, imgReader)

    if err != nil {
        log.Printf("еrror sending HTTP data for artwork %d: %s", id, err)
    }

    return nil
}

func (aih ArtstImageHandler) remove(
    writer http.ResponseWriter,
    req *http.Request,
    id int64,
) error {
    if err := aih.imageManager.RemoveArtistImage(req.Context(), id); err != nil {
        return err
    }

    writer.WriteHeader(http.StatusNoContent)
    return nil
}

func (aih ArtstImageHandler) upload(
    writer http.ResponseWriter,
    req *http.Request,
    id int64,
) error {
    err := aih.imageManager.SaveArtistImage(req.Context(), id, req.Body)
    if err == library.ErrArtworkTooBig {
        writer.WriteHeader(413)
        _, _ = writer.Write([]byte("Uploaded artwork is too large."))
        return nil
    } else if _, ok := err.(*library.ArtworkError); ok {
        writer.WriteHeader(http.StatusBadRequest)
        _, _ = writer.Write([]byte(err.Error()))
        return nil
    } else if err != nil {
        return err
    }

    writer.WriteHeader(http.StatusCreated)
    return nil
}

// NewArtistImagesHandler returns a new Artist image handler.
// It needs an implementation of the ArtistImageManager.
func NewArtistImagesHandler(
    am library.ArtistImageManager,
) *ArtstImageHandler {
    return &ArtstImageHandler{
        imageManager: am,
    }
}
