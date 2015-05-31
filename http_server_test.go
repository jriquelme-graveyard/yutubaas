package main

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"gopkg.in/dgrijalva/jwt-go.v2"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// Downloader mock
type MockDownloader struct {
	mock.Mock
	T *testing.T
}

func NewMockDownloader(t *testing.T) Downloader {
	m := &MockDownloader{}
	m.T = t
	return m
}

func (m *MockDownloader) DownloadVideo(video *DownloadVideo) {
	m.T.Logf("downloading video mock: %+v", video)
}

// API tests
type ApiRestSuite struct {
	suite.Suite
	HS256key []byte
	server   *httptest.Server
}

func (s *ApiRestSuite) SetupSuite() {
	*verbose = true

	// config
	config := &Config{}
	config.HS256key = "eCTEHBp97YKY4Bf89UKrV4az8FFe34fTYu4eLX8aryj6TUpycRkMJkHYRjbykCh"
	config.Accounts = map[string]ConfigUser{"jriquelme": ConfigUser{"Jorge", "asdf", "jorge@larix.cl", "jriquelme"}}

	s.HS256key = []byte(config.HS256key)

	// setup server
	httpServer, err := NewHttpServer(config)
	assert.Nil(s.T(), err)
	httpServer.Downloader = NewMockDownloader(s.T())
	s.server = httptest.NewServer(httpServer.CreateRouter())
}

func (s *ApiRestSuite) TearDownSuite() {
	s.server.Close()
}

func (s *ApiRestSuite) CreateToken(sub string) string {
	token := jwt.New(jwt.SigningMethodHS256)
	token.Claims["sub"] = sub
	token.Claims["iat"] = time.Now().Unix()
	token.Claims["exp"] = time.Now().Add(1 * time.Hour).Unix()
	signedToken, signErr := token.SignedString(s.HS256key)
	assert.Nil(s.T(), signErr)
	return signedToken
}

// run suite
func TestApiRestSuite(t *testing.T) {
	suite.Run(t, new(ApiRestSuite))
}

func (s *ApiRestSuite) TestDownloadNoToken() {
	// request
	r, err := http.NewRequest("POST", fmt.Sprintf("%s/download", s.server.URL), nil)
	assert.Nil(s.T(), err)
	res, err := http.DefaultClient.Do(r)
	assert.Nil(s.T(), err)

	// check response
	assert.Equal(s.T(), http.StatusUnauthorized, res.StatusCode)
	body, err := ioutil.ReadAll(res.Body)
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), "invalid token: no token present in request.\n", string(body))
}

func (s *ApiRestSuite) TestDownloadInvalidToken() {
	// request
	r, err := http.NewRequest("POST", fmt.Sprintf("%s/download", s.server.URL), nil)
	r.Header.Add("Authorization", fmt.Sprintf("Bearer %s", "invalid-token"))
	assert.Nil(s.T(), err)
	res, err := http.DefaultClient.Do(r)
	assert.Nil(s.T(), err)

	// check response
	assert.Equal(s.T(), http.StatusUnauthorized, res.StatusCode)
	body, err := ioutil.ReadAll(res.Body)
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), "invalid token: token contains an invalid number of segments\n", string(body))
}

func (s *ApiRestSuite) TestDownloadInvalidSignatureToken() {
	// request
	r, err := http.NewRequest("POST", fmt.Sprintf("%s/download", s.server.URL), nil)
	token := s.CreateToken("jriquelme")
	r.Header.Add("Authorization", fmt.Sprintf("Bearer %s", fmt.Sprintf("%sx", token)))
	assert.Nil(s.T(), err)
	res, err := http.DefaultClient.Do(r)
	assert.Nil(s.T(), err)

	// check response
	assert.Equal(s.T(), http.StatusUnauthorized, res.StatusCode)
	body, err := ioutil.ReadAll(res.Body)
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), "invalid token: signature is invalid\n", string(body))
}

func (s *ApiRestSuite) TestDownloadEmptyUrl() {
	// request
	r, err := http.NewRequest("POST", fmt.Sprintf("%s/download", s.server.URL), strings.NewReader("{}"))
	token := s.CreateToken("jriquelme")
	r.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))
	assert.Nil(s.T(), err)
	res, err := http.DefaultClient.Do(r)
	assert.Nil(s.T(), err)

	// check response
	assert.Equal(s.T(), http.StatusBadRequest, res.StatusCode)
	body, err := ioutil.ReadAll(res.Body)
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), "parse : empty url\n", string(body))
}

func (s *ApiRestSuite) TestDownloadWrongUrl() {
	// request
	json := "{\"url\": \"asdf\"}"
	r, err := http.NewRequest("POST", fmt.Sprintf("%s/download", s.server.URL), strings.NewReader(json))
	token := s.CreateToken("jriquelme")
	r.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))
	assert.Nil(s.T(), err)
	res, err := http.DefaultClient.Do(r)
	assert.Nil(s.T(), err)

	// check response
	assert.Equal(s.T(), http.StatusBadRequest, res.StatusCode)
	body, err := ioutil.ReadAll(res.Body)
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), "parse asdf: invalid URI for request\n", string(body))
}
