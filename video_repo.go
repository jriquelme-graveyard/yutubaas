package main

import (
	"fmt"
	"net/http"
	"net/url"
	"os"

	"github.com/mitchellh/goamz/aws"
	"github.com/mitchellh/goamz/s3"
)

type VideoRepository interface {
	SaveVideo(video *DownloadVideo) error
}

type S3VideoRepository struct {
	AwsAuth    aws.Auth
	BucketName string
}

type S3VideoRepoConfig struct {
	AccessKey  string
	SecretKey  string
	BucketName string
}

func NewS3VideoRepository(config *S3VideoRepoConfig) *S3VideoRepository {
	repo := &S3VideoRepository{}
	repo.AwsAuth = aws.Auth{
		AccessKey: config.AccessKey,
		SecretKey: config.SecretKey,
	}
	repo.BucketName = config.BucketName
	return repo
}

func (repo *S3VideoRepository) SaveVideo(video *DownloadVideo) error {
	region := aws.USEast
	connection := s3.New(repo.AwsAuth, region)

	// bucket
	bucket := connection.Bucket(repo.BucketName)
	s3path := video.File

	// open file
	file, err := os.Open(video.File)
	if err != nil {
		return err
	}
	defer file.Close()

	// set up for multipart upload
	filetype, err := repo.DetectContentType(file)
	if err != nil {
		return err
	}
	multi, err := bucket.InitMulti(s3path, filetype, s3.ACL("public-read"))
	if err != nil {
		return err
	}

	// upload parts
	const fileChunk = 5242880
	parts, err := multi.PutAll(file, fileChunk)
	if err != nil {
		return err
	}
	err = multi.Complete(parts)
	if err != nil {
		return err
	}

	video.DstUrl, err = url.ParseRequestURI(fmt.Sprintf("https://s3.amazonaws.com/%s/%s", repo.BucketName, video.File))
	if err != nil {
		return err
	}
	log.Debug("video uploaded to %s", video.DstUrl.String())
	return nil
}

func (repo *S3VideoRepository) DetectContentType(file *os.File) (string, error) {
	buff := make([]byte, 512)
	_, err := file.Read(buff)
	return http.DetectContentType(buff), err
}
