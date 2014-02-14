// This module contains the webserver whcih deals with processing requests
// from the user, presenting him with the interface of the application.
package webserver

import (
	"crypto/tls"
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/ironsmile/httpms/src/config"
	"github.com/ironsmile/httpms/src/library"
)

// Represends our webserver. It will be controlled from here
type Server struct {

	// Configuration of this server
	cfg config.Config

	// WG used in Server.Wait to sync with server's end
	wg sync.WaitGroup

	// Makes sure Serve does not return before all the starting work ha been finished
	startWG sync.WaitGroup

	// The actual http.Server doing the HTTP work
	httpSrv *http.Server

	// The server's net.Listener. Used in the Server.Stop func
	listener net.Listener

	// This server's library with media
	library library.Library
}

// The function that actually starts the webserver. It attaches all the handlers
// and starts the webserver while consulting the ServerConfig supplied. Trying to call
// this method more than once for the same server will result in panic.
func (srv *Server) Serve() {
	if srv.listener != nil {
		panic("Second Server.Serve call for the same server")
	}
	srv.wg.Add(1)
	srv.startWG.Add(1)
	go srv.serveGoroutine()
	srv.startWG.Wait()
}

func (srv *Server) serveGoroutine() {
	defer srv.wg.Done()

	mux := http.NewServeMux()

	mux.Handle("/", http.FileServer(http.Dir(srv.cfg.HTTPRoot)))
	mux.Handle("/search/", http.StripPrefix("/search/", NewSearchHandler(srv.library)))
	mux.Handle("/file/", http.StripPrefix("/file/", NewFileHandler(srv.library)))
	mux.Handle("/album/", http.StripPrefix("/album/", NewAlbumHandler(srv.library)))

	var handler http.Handler

	handler = mux

	if srv.cfg.Gzip {
		log.Println("Adding gzip handler")
		handler = NewGzipHandler(handler)
	}

	if srv.cfg.Auth {
		log.Println("Adding basic authenticate handler")
		handler = BasicAuthHandler{
			handler,
			srv.cfg.Authenticate.User,
			srv.cfg.Authenticate.Password,
		}
	}

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
	config := &tls.Config{}
	if srv.httpSrv.TLSConfig != nil {
		*config = *srv.httpSrv.TLSConfig
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

// Stops the webserver
func (srv *Server) Stop() {
	if srv.listener != nil {
		srv.listener.Close()
		srv.listener = nil
	}
}

// Syncs whoever called this with the server's stop
func (srv *Server) Wait() {
	srv.wg.Wait()
}

// Returns a new Server using the supplied configuration cfg. The returned server
// is ready and calling its Serve method will start it.
func NewServer(cfg config.Config, lib library.Library) (srv *Server) {
	srv = new(Server)
	srv.cfg = cfg
	srv.library = lib
	return
}
