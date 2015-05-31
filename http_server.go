package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/gorilla/context"
	"github.com/gorilla/mux"
	"github.com/gorilla/schema"
	"github.com/justinas/alice"
	"gopkg.in/dgrijalva/jwt-go.v2"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"
)

type HttpServer struct {
	HS256key   []byte // to sign JWT tokens
	Accounts   map[string]ConfigUser
	Downloader Downloader
}

func NewHttpServer(config *Config) (*HttpServer, error) {
	log.Debug("config: %+v", config)
	server := &HttpServer{}
	server.HS256key = []byte(config.HS256key)
	server.Accounts = config.Accounts
	repoConfig := &S3VideoRepoConfig{config.S3Config.AccessKey, config.S3Config.SecretKey, config.S3Config.Bucket}
	videoRepo := NewS3VideoRepository(repoConfig)
	mailer, err := NewMailgunMailer(config.MailgunConfig.From, config.MailgunConfig.Key, config.MailgunConfig.Domain)
	if err != nil {
		return nil, err
	}
	server.Downloader = NewDefaultDownloader(videoRepo, mailer)
	return server, nil
}

func (s *HttpServer) CreateRouter() *mux.Router {
	commonHandlers := alice.New(s.LoggingHandler)

	router := mux.NewRouter().StrictSlash(true)

	// service status
	router.Handle("/status", commonHandlers.ThenFunc(s.HandleStatus)).Methods("GET")
	router.Handle("/login", commonHandlers.ThenFunc(s.HandleLogin)).Methods("POST")
	router.Handle("/download/mailgun", commonHandlers.ThenFunc(s.HandleDownloadMailgun)).Methods("POST")
	router.Handle("/download", commonHandlers.Append(s.AuthenticationHandler).ThenFunc(s.HandleDownload)).Methods("POST")

	return router
}

func (s *HttpServer) LoggingHandler(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		t1 := time.Now()
		next.ServeHTTP(w, r)
		t2 := time.Now()
		log.Debug("[%s] %q %v\n", r.Method, r.URL.String(), t2.Sub(t1))
		if *verbose {
			if dump, err := httputil.DumpRequest(r, true); err != nil {
				log.Debug("Error dumping request: %s", err)
			} else {
				// FIXME: no body in output :S
				log.Debug("Request dump: %s", dump)
			}
		}
	}

	return http.HandlerFunc(fn)
}

type StatusInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

func (s *HttpServer) HandleStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	response := StatusInfo{"Yutubaas server", version}
	json.NewEncoder(w).Encode(response)
}

func (s *HttpServer) AuthenticationHandler(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		f := func(t *jwt.Token) (interface{}, error) { return s.HS256key, nil }
		token, err := jwt.ParseFromRequest(r, f)
		if err != nil {
			http.Error(w, fmt.Sprintf("invalid token: %s", err), http.StatusUnauthorized)
			return
		}
		if token == nil {
			http.Error(w, "missing Authorization token", http.StatusUnauthorized)
			return
		}
		log.Debug("token claims: %+v", token.Claims)
		if username, ok := token.Claims["sub"]; !ok {
			http.Error(w, "missing sub in token claims", http.StatusUnauthorized)
			return
		} else {
			context.Set(r, "sub", username)
		}

		next.ServeHTTP(w, r)
	}

	return http.HandlerFunc(fn)
}

type Video struct {
	Url string `json:"url"`
}

func (s *HttpServer) HandleDownload(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	video := Video{}
	if err := decoder.Decode(&video); err != nil {
		http.Error(w, "invalid json message.", http.StatusBadRequest)
		return
	}
	log.Debug("parsing %s", video.Url)
	videoUrl, err := url.ParseRequestURI(video.Url)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusCreated)

	// download
	username := context.Get(r, "sub").(string)
	account, _ := s.Accounts[username]
	videoDwn := &DownloadVideo{}
	videoDwn.SrcUrl = videoUrl
	videoDwn.DstUrl = nil // downloader set this
	videoDwn.Title = ""   // downloader set this
	videoDwn.Name = account.Name
	videoDwn.Username = username
	videoDwn.Email = account.Email
	videoDwn.Error = nil
	go s.Downloader.DownloadVideo(videoDwn)
}

type MailgunMessage struct {
	Sender       string `schema:"sender"`
	Timestamp    string `schema:"timestamp"`
	Token        string `schema:"token"`
	Signature    string `schema:"signature"`
	StrippedText string `schema:"stripped-text"`
}

func (s *HttpServer) HandleDownloadMailgun(w http.ResponseWriter, r *http.Request) {
	log.Debug("Download from Mailgun!")
	err := r.ParseForm()
	if err != nil {
		log.Error("error parsing form from Mailgun! %s", err)
		w.WriteHeader(http.StatusOK) // sending 200 anyway
		return
	}

	// decode params
	mgmsg := &MailgunMessage{}
	decoder := schema.NewDecoder()
	decoder.IgnoreUnknownKeys(true)
	err = decoder.Decode(mgmsg, r.PostForm)
	if err != nil {
		log.Error("error decoding parameter from Mailgun! %s", err)
		w.WriteHeader(http.StatusOK) // sending 200 anyway
		return
	}
	log.Debug("parameters: %+v", mgmsg)

	// TODO: check signature

	// get account
	account := s.GetAccountFromEmail(mgmsg.Sender)
	if account == nil {
		log.Error("unknown sender from Mailgun: %s", mgmsg.Sender)
		w.WriteHeader(http.StatusOK) // sending 200 anyway
		return
	}

	// TODO: have to move this to send the error message to the user
	log.Debug("parsing %s", mgmsg.StrippedText)
	scanner := bufio.NewScanner(strings.NewReader(mgmsg.StrippedText))
	if !scanner.Scan() {
		log.Error("error extracting url from Mailgun message: %s", mgmsg.StrippedText)
		w.WriteHeader(http.StatusOK) // sending 200 anyway
		return
	}
	videoUrl, err := url.ParseRequestURI(scanner.Text())
	if err != nil {
		log.Error("wrong url(%s) in Mailgun message: %s", mgmsg.StrippedText, err)
		w.WriteHeader(http.StatusOK) // sending 200 anyway
		return
	}

	// everything is ok, download!
	w.WriteHeader(http.StatusOK)

	// download
	videoDwn := &DownloadVideo{}
	videoDwn.SrcUrl = videoUrl
	videoDwn.DstUrl = nil // downloader set this
	videoDwn.Title = ""   // downloader set this
	videoDwn.Name = account.Name
	videoDwn.Username = account.Username
	videoDwn.Email = account.Email
	videoDwn.Error = nil
	go s.Downloader.DownloadVideo(videoDwn)
}

func (s *HttpServer) GetAccountFromEmail(email string) *ConfigUser {
	for _, account := range s.Accounts {
		if account.Email == email {
			return &account
		}
	}
	return nil
}
