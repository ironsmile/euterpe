// This module contains the webserver whcih deals with processing requests
// from the user, presenting him with the interface of the application.
package webserver

import (
	"crypto/tls"
	"fmt"
	"html/template"
	"log"
	"net"
	"net/http"
	"os"
	"sync"
	"time"
)

// The configuration which should be supplied to the webserver
type ServerConfig struct {
	Address  string // Address on which the server will listen. See http/Server.Addr
	Root     string // The http root directory containing the interface files
	SSL      bool   // Should it use SSL when serving
	SSLCert  string // The SSL certificate. Only makes sens if SSL is true
	SSLKey   string // The SSL key. Only makes sense if SSL is true
	Auth     bool   // Should the server require HTTP auth
	AuthUser string // HTTP basic authenticate username
	AuthPass string // HTTP basic authenticate password
}

// Handler responsible for search requests. It will use the Library to
// return a list of matched files to the interface.
type SearchHandler struct{}

// This method is required by the http.Handler's interface
func (sh SearchHandler) ServeHTTP(writer http.ResponseWriter, req *http.Request) {

	fullPath := fmt.Sprintf("%s?%s", req.URL.Path, req.URL.RawQuery)

	templateDir := fmt.Sprintf("%s/src/github.com/ironsmile/httpms/templates",
		os.ExpandEnv("$GOPATH"))
	templateFile := fmt.Sprintf("%s/test.html", templateDir)

	_, err := os.Stat(templateFile)

	if err != nil {
		writer.WriteHeader(http.StatusInternalServerError)
		writer.Write([]byte(fmt.Sprintf("%s does not exist", templateFile)))
		return
	}

	tmpl, err := template.ParseFiles(templateFile)

	if err != nil {
		writer.WriteHeader(http.StatusInternalServerError)
		writer.Write([]byte(fmt.Sprintf("Error parsing template %s", templateFile)))
		return
	}

	variables := map[string]string{
		"FullPath":     fullPath,
		"TemplateFile": templateFile,
	}

	err = tmpl.Execute(writer, variables)

	if err != nil {
		writer.WriteHeader(http.StatusInternalServerError)
		writer.Write([]byte("Error executing template"))
		return
	}
}

// Represends our webserver. It will be controlled from here
type Server struct {
	cfg      ServerConfig   // Configuration of this server
	wg       sync.WaitGroup // WG used in Server.Wait to sync with server's end
	httpSrv  *http.Server   // The actual http.Server doing the HTTP work
	listener net.Listener   // The server's net.Listener. Used in the Server.Stop func
}

// The function that actually starts the webserver. It attaches all the handlers
// and starts the webserver while consulting the ServerConfig supplied.
func (srv *Server) Serve() {
	srv.wg.Add(1)
	go srv.serveGoroutine()
}

func (srv *Server) serveGoroutine() {
	defer func() {
		srv.wg.Done()
	}()

	mux := http.NewServeMux()

	mux.Handle("/", http.FileServer(http.Dir(srv.cfg.Root)))
	mux.Handle("/search/", SearchHandler{})

	srv.httpSrv = &http.Server{
		Addr:           srv.cfg.Address,
		Handler:        mux,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	var reason error

	if srv.cfg.SSL {
		reason = srv.listenAndServeTLS(srv.cfg.SSLCert, srv.cfg.SSLKey)
	} else {
		reason = srv.listenAndServe()
	}

	// When the listener is nil it is probably a scheduled stop. I can't be sure though
	//!TODO: make sure listener == nil is only possible after Server.Stop()
	if reason != nil && srv.listener != nil {
		log.Print(reason)
	}
}

// Uses our own listener to make our server stoppable. Similar to
// net.http.Server.ListenAndServer
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
	return srv.httpSrv.Serve(lsn)
}

// Uses our own listener to make our server stoppable. Similar to
// net.http.Server.ListenAndServerTLS
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
func NewServer(cfg ServerConfig) (srv Server) {
	srv.cfg = cfg
	return
}
