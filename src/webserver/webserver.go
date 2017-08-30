// Package webserver contains the webserver which deals with processing requests
// from the user, presenting him with the interface of the application.
package webserver

import (
	"context"
	"crypto/tls"
	"log"
	"net"
	"net/http"
	"sync"
	"time"

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
	library library.Library

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
	mux := http.NewServeMux()

	mux.Handle("/", srv.withBasicAuth(http.FileServer(http.Dir(srv.cfg.HTTPRoot))))
	searchHandler := srv.withBasicAuth(NewSearchHandler(srv.library))
	mux.Handle("/search/", http.StripPrefix("/search/", searchHandler))
	mux.Handle("/file/", http.StripPrefix("/file/", NewFileHandler(srv.library)))
	albumHandler := srv.withBasicAuth(NewAlbumHandler(srv.library))
	mux.Handle("/album/", http.StripPrefix("/album/", albumHandler))
	browseHandler := srv.withBasicAuth(NewBrowseHandler(srv.library))
	mux.Handle("/browse/", http.StripPrefix("/browse/", browseHandler))

	handler := NewTerryHandler(mux)

	if srv.cfg.Gzip {
		log.Println("Adding gzip handler")
		handler = NewGzipHandler(handler)
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

func (srv *Server) withBasicAuth(handler http.Handler) http.Handler {
	if !srv.cfg.Auth {
		return handler
	}

	return BasicAuthHandler{
		handler,
		srv.cfg.Authenticate.User,
		srv.cfg.Authenticate.Password,
	}
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
	log.Println("Webserver started.")
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
	log.Println("Webserver started.")
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
func NewServer(ctx context.Context, cfg config.Config, lib library.Library) *Server {
	ctx, cancelCtx := context.WithCancel(ctx)
	return &Server{
		ctx:        ctx,
		cancelFunc: cancelCtx,
		cfg:        cfg,
		library:    lib,
	}
}
