package s3

import (
	"context"
	"fmt"
	"io"
	"path"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/pkkulhari/backme/internal/config"
)

type Client struct {
	s3Client *s3.Client
	bucket   string
}

type Options struct {
	Endpoint string
}

func New(cfg *config.Config, opts *Options) (*Client, error) {
	awsCfg, err := awsconfig.LoadDefaultConfig(
		context.TODO(),
		awsconfig.WithRegion(cfg.AWS.Region),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			cfg.AWS.AccessKeyID,
			cfg.AWS.SecretAccessKey,
			"",
		)),
	)
	if err != nil {
		return nil, fmt.Errorf("unable to load AWS config: %w", err)
	}

	s3Client := s3.NewFromConfig(awsCfg)
	if opts != nil && opts.Endpoint != "" {
		// Use custom endpoint for testing
		awsCfg.BaseEndpoint = aws.String(opts.Endpoint)
		s3Client = s3.NewFromConfig(awsCfg)
	}

	return &Client{
		s3Client: s3Client,
		bucket:   cfg.AWS.Bucket,
	}, nil
}

func (c *Client) Upload(ctx context.Context, key string, reader io.Reader) error {
	_, err := c.s3Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
		Body:   reader,
	})
	if err != nil {
		return fmt.Errorf("failed to upload file to S3: %w", err)
	}

	return nil
}

func (c *Client) Download(ctx context.Context, key string) (io.ReadCloser, error) {
	result, err := c.s3Client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to download file from S3: %w", err)
	}

	return result.Body, nil
}

func (c *Client) ListObjects(ctx context.Context, prefix string) ([]string, error) {
	var objects []string
	paginator := s3.NewListObjectsV2Paginator(c.s3Client, &s3.ListObjectsV2Input{
		Bucket: aws.String(c.bucket),
		Prefix: aws.String(prefix),
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list objects in S3: %w", err)
		}

		for _, obj := range page.Contents {
			objects = append(objects, *obj.Key)
		}
	}

	return objects, nil
}

func (c *Client) GetObjectMetadata(ctx context.Context, key string) (*s3.HeadObjectOutput, error) {
	result, err := c.s3Client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get object metadata from S3: %w", err)
	}

	return result, nil
}

func (c *Client) DeleteObject(ctx context.Context, key string) error {
	_, err := c.s3Client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("failed to delete object from S3: %w", err)
	}

	return nil
}

func (c *Client) CreateBucket(ctx context.Context) error {
	_, err := c.s3Client.CreateBucket(ctx, &s3.CreateBucketInput{
		Bucket: aws.String(c.bucket),
	})
	if err != nil {
		return fmt.Errorf("failed to create bucket: %w", err)
	}

	// Wait for bucket to exist
	waiter := s3.NewBucketExistsWaiter(c.s3Client)
	if err := waiter.Wait(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(c.bucket),
	}, 30*time.Second); err != nil {
		return fmt.Errorf("failed to wait for bucket to exist: %w", err)
	}

	return nil
}

func (c *Client) DeleteBucket(ctx context.Context) error {
	// First, delete all objects in the bucket
	objects, err := c.ListObjects(ctx, "")
	if err != nil {
		return fmt.Errorf("failed to list objects for bucket deletion: %w", err)
	}

	for _, key := range objects {
		if err := c.DeleteObject(ctx, key); err != nil {
			return fmt.Errorf("failed to delete object %s during bucket deletion: %w", key, err)
		}
	}

	// Now delete the bucket
	_, err = c.s3Client.DeleteBucket(ctx, &s3.DeleteBucketInput{
		Bucket: aws.String(c.bucket),
	})
	if err != nil {
		return fmt.Errorf("failed to delete bucket: %w", err)
	}

	// Wait for bucket to be deleted
	waiter := s3.NewBucketNotExistsWaiter(c.s3Client)
	if err := waiter.Wait(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(c.bucket),
	}, 30*time.Second); err != nil {
		return fmt.Errorf("failed to wait for bucket deletion: %w", err)
	}

	return nil
}

func GetObjectKey(prefix string, parts ...string) string {
	if prefix == "" {
		return path.Join(parts...)
	}
	return path.Join(append([]string{prefix}, parts...)...)
}
