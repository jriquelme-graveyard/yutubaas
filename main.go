package main

import (
	"fmt"
	"github.com/op/go-logging"
	"gopkg.in/alecthomas/kingpin.v1"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"net/http"
	"os"
)

// comand line flags
var (
	log        = logging.MustGetLogger("yutubaas")
	verbose    = kingpin.Flag("verbose", "verbose output").Default("false").Bool()
	configfile = kingpin.Flag("config", "config file").Required().String()
	httpPort   = kingpin.Flag("port", "http port").Default("8080").Int()
)

const (
	version = "0.1.0"
)

func init() {
	// setup logger
	stdoutBackend := logging.NewLogBackend(os.Stdout, "", 0)
	format := logging.MustStringFormatter("%{color}%{time:15:04:05.000} %{shortfunc}[%{level:.4s} %{id:03x}]%{color:reset} %{message}")
	backendFmt := logging.NewBackendFormatter(stdoutBackend, format)
	logging.SetBackend(backendFmt)
}

func main() {
	kingpin.Version(version)
	kingpin.Parse()

	// load config
	config, cfgErr := LoadConfig(*configfile)
	if cfgErr != nil {
		log.Fatalf("error loading config from %s: %s", *configfile, cfgErr)
	}

	// bring up http server
	server, err := NewHttpServer(config)
	if err != nil {
		log.Fatalf("Error creating http server: %s", err)
	}
	addr := fmt.Sprintf(":%d", *httpPort)
	log.Debug("http server listening to %s", addr)
	http.Handle("/", server.CreateRouter())
	http.ListenAndServe(addr, nil)
}

// configuration
type Config struct {
	HS256key      string                "hs256key"
	Accounts      map[string]ConfigUser "accounts"
	MailgunConfig MailgunConfig         "mailgun"
	S3Config      S3Config              "s3"
}

type ConfigUser struct {
	Name     string "name"
	Password string "password"
	Email    string "email"
}

type MailgunConfig struct {
	Key    string "key"
	Domain string "domain"
}

type S3Config struct {
	AccessKey string "accessKey"
	SecretKey string "secretKey"
	Bucket    string "bucket"
}

func LoadConfig(path string) (*Config, error) {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	config := &Config{}
	yamlErr := yaml.Unmarshal(b, config)
	return config, yamlErr
}
