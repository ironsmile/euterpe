// Package webserver contains the webserver which deals with processing requests
// from the user, presenting him with the interface of the application.
package webserver

import (
	"context"
	"crypto/tls"
	"log"
	"net"
	"net/http"
	"path/filepath"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/ironsmile/httpms/src/config"
	"github.com/ironsmile/httpms/src/library"
)

// Server represends our webserver. It will be controlled from here
type Server struct {
	// Used for server-wide stopping, cancelation and stuff
	ctx context.Context

	// Calling this function will stop the server
	cancelFunc context.CancelFunc

	// Configuration of this server
	cfg config.Config

	// Makes sure Serve does not return before all the starting work ha been finished
	startWG sync.WaitGroup

	// The actual http.Server doing the HTTP work
	httpSrv *http.Server

	// The server's net.Listener. Used in the Server.Stop func
	listener net.Listener

	// This server's library with media
	library *library.LocalLibrary

	// Makes the server lockable. This lock should be used for accessing the
	// listener
	sync.Mutex
}

// Serve actually starts the webserver. It attaches all the handlers
// and starts the webserver while consulting the ServerConfig supplied. Trying to call
// this method more than once for the same server will result in panic.
func (srv *Server) Serve() {
	srv.Lock()
	defer srv.Unlock()
	if srv.listener != nil {
		panic("Second Server.Serve call for the same server")
	}
	srv.startWG.Add(1)
	go srv.serveGoroutine()
	srv.startWG.Wait()
}

func (srv *Server) serveGoroutine() {
	notFoundAlbumImage := filepath.Join(srv.cfg.HTTPRoot, "images", "unknownAlbum.png")

	staticFilesHandler := http.FileServer(http.Dir(srv.cfg.HTTPRoot))
	searchHandler := NewSearchHandler(srv.library)
	albumHandler := NewAlbumHandler(srv.library)
	artoworkHandler := NewAlbumArtworkHandler(srv.library, notFoundAlbumImage)
	browseHandler := NewBrowseHandler(srv.library)
	mediaFileHandler := NewFileHandler(srv.library)

	router := mux.NewRouter()
	router.StrictSlash(true)
	router.UseEncodedPath()
	router.Handle("/file/{fileID}", mediaFileHandler).Methods("GET")
	router.Handle("/album/{albumID}/artwork", artoworkHandler).Methods("GET", "PUT", "DELETE")
	router.Handle("/album/{albumID}", albumHandler).Methods("GET")
	router.Handle("/browse", browseHandler).Methods("GET")
	router.Handle("/search/{searchQuery}", searchHandler).Methods("GET")
	router.Handle("/search", searchHandler).Methods("GET")
	router.PathPrefix("/").Handler(staticFilesHandler).Methods("GET")

	handler := NewTerryHandler(router)

	if srv.cfg.Gzip {
		handler = NewGzipHandler(handler)
	}

	if srv.cfg.Auth {
		handler = BasicAuthHandler{
			handler,
			srv.cfg.Authenticate.User,
			srv.cfg.Authenticate.Password,
		}
	}

	handler = func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, closeRequest := context.WithCancel(srv.ctx)
			h.ServeHTTP(w, r.WithContext(ctx))
			closeRequest()
		})
	}(handler)

	srv.httpSrv = &http.Server{
		Addr:           srv.cfg.Listen,
		Handler:        handler,
		ReadTimeout:    time.Duration(srv.cfg.ReadTimeout) * time.Second,
		WriteTimeout:   time.Duration(srv.cfg.WriteTimeout) * time.Second,
		MaxHeaderBytes: srv.cfg.MaxHeadersSize,
	}

	var reason error

	if srv.cfg.SSL {
		reason = srv.listenAndServeTLS(srv.cfg.SSLCertificate.Crt,
			srv.cfg.SSLCertificate.Key)
	} else {
		reason = srv.listenAndServe()
	}

	log.Println("Webserver stopped.")

	if reason != nil {
		log.Printf("Reason: %s\n", reason.Error())
	}

	srv.cancelFunc()
}

// Uses our own listener to make our server stoppable. Similar to
// net.http.Server.ListenAndServer only this version saves a reference to the listener
func (srv *Server) listenAndServe() error {
	addr := srv.httpSrv.Addr
	if addr == "" {
		addr = ":http"
	}
	lsn, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	srv.listener = lsn
	log.Printf("Webserver started on http://%s\n", addr)
	srv.startWG.Done()
	return srv.httpSrv.Serve(lsn)
}

// Uses our own listener to make our server stoppable. Similar to
// net.http.Server.ListenAndServerTLS only this version saves a reference
// to the listener
func (srv *Server) listenAndServeTLS(certFile, keyFile string) error {
	addr := srv.httpSrv.Addr
	if addr == "" {
		addr = ":https"
	}

	var config *tls.Config

	if srv.httpSrv.TLSConfig != nil {
		config = srv.httpSrv.TLSConfig
	} else {
		config = &tls.Config{}
	}

	if config.NextProtos == nil {
		config.NextProtos = []string{"http/1.1"}
	}

	var err error
	config.Certificates = make([]tls.Certificate, 1)
	config.Certificates[0], err = tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return err
	}

	conn, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	tlsListener := tls.NewListener(conn, config)
	srv.listener = tlsListener
	log.Printf("Webserver started on https://%s\n", addr)
	srv.startWG.Done()
	return srv.httpSrv.Serve(tlsListener)
}

// Stop stops the webserver
func (srv *Server) Stop() {
	srv.Lock()
	defer srv.Unlock()
	if srv.listener != nil {
		srv.listener.Close()
		srv.listener = nil
	}
}

// Wait syncs whoever called this with the server's stop
func (srv *Server) Wait() {
	<-srv.ctx.Done()
}

// NewServer Returns a new Server using the supplied configuration cfg. The returned
// server is ready and calling its Serve method will start it.
func NewServer(ctx context.Context, cfg config.Config, lib *library.LocalLibrary) *Server {
	ctx, cancelCtx := context.WithCancel(ctx)
	return &Server{
		ctx:        ctx,
		cancelFunc: cancelCtx,
		cfg:        cfg,
		library:    lib,
	}
}
