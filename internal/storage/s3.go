package storage

import (
	"context"
	"fmt"
	"io"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	appConfig "github.com/blau/strength-leaderboard2/internal/config"
)

type S3Storage struct {
	client    *s3.Client
	bucket    string
	publicURL string
}

func NewS3Storage(ctx context.Context, cfg appConfig.Config) (*S3Storage, error) {
	customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		return aws.Endpoint{
			URL:               cfg.S3Endpoint,
			SigningRegion:     cfg.S3Region,
			HostnameImmutable: true,
		}, nil
	})

	sdkConfig, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(cfg.S3Region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(cfg.S3AccessKey, cfg.S3SecretKey, "")),
		config.WithEndpointResolverWithOptions(customResolver),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load S3 config: %w", err)
	}

	client := s3.NewFromConfig(sdkConfig)

	// Ensure bucket exists
	_, err = client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(cfg.S3Bucket),
	})
	if err != nil {
		log.Printf("bucket %s does not exist, creating it...", cfg.S3Bucket)
		_, err = client.CreateBucket(ctx, &s3.CreateBucketInput{
			Bucket: aws.String(cfg.S3Bucket),
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create bucket: %w", err)
		}
	}

	return &S3Storage{
		client:    client,
		bucket:    cfg.S3Bucket,
		publicURL: cfg.S3PublicURL,
	}, nil
}

func (s *S3Storage) Upload(ctx context.Context, key string, body io.Reader, contentType string) (string, error) {
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		Body:        body,
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload to S3: %w", err)
	}

	return fmt.Sprintf("%s/%s/%s", s.publicURL, s.bucket, key), nil
}
