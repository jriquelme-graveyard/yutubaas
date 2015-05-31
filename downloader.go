package main

import (
	"bufio"
	"bytes"
	"net/url"
	"os"
	"os/exec"
	"strings"
)

type DownloadVideo struct {
	SrcUrl   *url.URL // youtube url
	DstUrl   *url.URL // youtube url
	Title    string   // video title
	Name     string   // name of the user
	Username string
	Email    string
	File     string
	Error    error
}

type Downloader interface {
	DownloadVideo(video *DownloadVideo)
}

type DefaultDownloader struct {
	VideoRepo VideoRepository
	Mailer    Mailer
}

func NewDefaultDownloader(videoRepo VideoRepository, mailer Mailer) *DefaultDownloader {
	return &DefaultDownloader{videoRepo, mailer}
}

func (dwn *DefaultDownloader) DownloadVideo(video *DownloadVideo) {
	// get title and filename
	var err error
	err = dwn.CompleteMetadata(video)
	if err != nil {
		log.Error("error getting metadata for %s: %s", video.SrcUrl.String(), err)
		video.Error = err
		dwn.Mailer.Notify(video)
		return
	}

	// download video
	cmd := exec.Command("youtube-dl", "--newline", video.SrcUrl.String())
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Error("error downloading %s: %s", video.SrcUrl.String(), err)
		video.Error = err
		dwn.Mailer.Notify(video)
		return
	}
	if err := cmd.Start(); err != nil {
		log.Error("error downloading %s: %s", video.SrcUrl.String(), err)
		video.Error = err
		dwn.Mailer.Notify(video)
		return
	}
	log.Debug("downloading %s...", video.SrcUrl)
	ro := bufio.NewReader(stdout)
	scanner := bufio.NewScanner(ro)
	for scanner.Scan() {
		log.Debug("[%s] %s\n", video.SrcUrl, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		log.Error("error downloading %s: %s", video.SrcUrl.String(), err)
		video.Error = err
		dwn.Mailer.Notify(video)
		return
	}
	if err := cmd.Wait(); err != nil {
		log.Error("error downloading %s: %s", video.SrcUrl.String(), err)
		video.Error = err
		dwn.Mailer.Notify(video)
		return
	}

	// put into S3
	log.Debug("uploading %s to S3", video.Title)
	if err = dwn.VideoRepo.SaveVideo(video); err != nil {
		log.Error("error downloading %s: %s", video.SrcUrl.String(), err)
		video.Error = err
		dwn.Mailer.Notify(video)
		return
	}

	if err := os.Remove(video.File); err != nil {
		log.Error("error removing file %s: %s", video.File, err)
	}

	log.Debug("done with %s, sending success email", video.Title)
	dwn.Mailer.Notify(video)
}

func (dwn *DefaultDownloader) CompleteMetadata(video *DownloadVideo) error {
	out, err := exec.Command("youtube-dl", "-e", "--get-filename", video.SrcUrl.String()).Output()
	if err != nil {
		return err
	}
	buffer := bytes.NewBuffer(out)
	if sz, err := buffer.ReadString('\n'); err != nil {
		return err
	} else {
		video.Title = strings.TrimSpace(sz)
	}
	if sz, err := buffer.ReadString('\n'); err != nil {
		return err
	} else {
		video.File = strings.TrimSpace(sz)
	}
	return nil
}
