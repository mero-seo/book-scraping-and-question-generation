package storage

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// R2Client wraps an S3-compatible client for Cloudflare R2.
type R2Client struct {
	client    *s3.Client
	bucket    string
	accountID string
}

// NewR2Client creates a new R2Client by reading configuration from environment
// variables: CLOUDFLARE_R2_ACCOUNT_ID, CLOUDFLARE_R2_ACCESS_KEY_ID,
// CLOUDFLARE_R2_SECRET_ACCESS_KEY, and CLOUDFLARE_R2_BUCKET_NAME.
func NewR2Client(ctx context.Context) (*R2Client, error) {
	accountID := os.Getenv("CLOUDFLARE_R2_ACCOUNT_ID")
	if accountID == "" {
		return nil, fmt.Errorf("CLOUDFLARE_R2_ACCOUNT_ID is required")
	}

	accessKey := os.Getenv("CLOUDFLARE_R2_ACCESS_KEY_ID")
	if accessKey == "" {
		return nil, fmt.Errorf("CLOUDFLARE_R2_ACCESS_KEY_ID is required")
	}

	secretKey := os.Getenv("CLOUDFLARE_R2_SECRET_ACCESS_KEY")
	if secretKey == "" {
		return nil, fmt.Errorf("CLOUDFLARE_R2_SECRET_ACCESS_KEY is required")
	}

	bucket := os.Getenv("CLOUDFLARE_R2_BUCKET_NAME")
	if bucket == "" {
		return nil, fmt.Errorf("CLOUDFLARE_R2_BUCKET_NAME is required")
	}

	endpoint := fmt.Sprintf("https://%s.r2.cloudflarestorage.com", accountID)

	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(accessKey, secretKey, ""),
		),
		config.WithRegion("auto"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(endpoint)
	})

	return &R2Client{
		client:    client,
		bucket:    bucket,
		accountID: accountID,
	}, nil
}

// Upload puts an object into the R2 bucket and returns its public URL.
func (r *R2Client) Upload(ctx context.Context, key string, body io.Reader, contentType string) (string, error) {
	input := &s3.PutObjectInput{
		Bucket:      aws.String(r.bucket),
		Key:         aws.String(key),
		Body:        body,
		ContentType: aws.String(contentType),
	}

	if _, err := r.client.PutObject(ctx, input); err != nil {
		return "", fmt.Errorf("failed to upload object %q: %w", key, err)
	}

	url := fmt.Sprintf("https://%s.%s.r2.cloudflarestorage.com/%s", r.bucket, r.accountID, key)
	return url, nil
}

// Download retrieves an object from the R2 bucket. The caller is responsible
// for closing the returned ReadCloser.
func (r *R2Client) Download(ctx context.Context, key string) (io.ReadCloser, error) {
	input := &s3.GetObjectInput{
		Bucket: aws.String(r.bucket),
		Key:    aws.String(key),
	}

	output, err := r.client.GetObject(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to download object %q: %w", key, err)
	}

	return output.Body, nil
}

// Delete removes an object from the R2 bucket.
func (r *R2Client) Delete(ctx context.Context, key string) error {
	input := &s3.DeleteObjectInput{
		Bucket: aws.String(r.bucket),
		Key:    aws.String(key),
	}

	if _, err := r.client.DeleteObject(ctx, input); err != nil {
		return fmt.Errorf("failed to delete object %q: %w", key, err)
	}

	return nil
}
