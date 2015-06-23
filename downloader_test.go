package main

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mailer mock
type MockMailer struct {
	mock.Mock
	T *testing.T
}

func NewMockMailer(t *testing.T) Mailer {
	m := &MockMailer{}
	m.T = t
	return m
}

func (m *MockMailer) Notify(video *DownloadVideo) {
	m.T.Logf("sending mail mock: %+v", video)
}

// VideoRepo mock

type MockVideoRepository struct {
	mock.Mock
	T *testing.T
}

func (m *MockVideoRepository) SaveVideo(video *DownloadVideo) error {
	m.T.Logf("video repo mock: %+v", video)
	return nil
}

func NewMockVideoRepository(t *testing.T) VideoRepository {
	m := &MockVideoRepository{}
	m.T = t
	return m
}

func TestCompleteMetadata(t *testing.T) {
	downloader := NewDefaultDownloader(NewMockVideoRepository(t), NewMockMailer(t))
	video := &DownloadVideo{}
	var err error
	video.SrcUrl, err = url.ParseRequestURI("https://www.youtube.com/watch?v=bS5P_LAqiVg")
	assert.Nil(t, err)
	err = downloader.CompleteMetadata(video)
	assert.Nil(t, err)
	assert.Equal(t, "KUNG FURY Official Movie [HD]", video.Title)
	assert.Equal(t, "KUNG FURY Official Movie [HD]-bS5P_LAqiVg.mp4", video.File)
}
