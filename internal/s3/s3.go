package s3

import (
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

// Client wraps the AWS S3 SDK for pre-signed URL generation.
type Client struct {
	svc    *s3.S3
	bucket string
}

// NewClient creates a new S3 client.
func NewClient(accessKey, secretKey, region, bucket string) (*Client, error) {
	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String(region),
		Credentials: credentials.NewStaticCredentials(accessKey, secretKey, ""),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create AWS session: %w", err)
	}

	log.Println("S3 client initialized")
	return &Client{
		svc:    s3.New(sess),
		bucket: bucket,
	}, nil
}

// GeneratePresignedUploadURL creates a pre-signed PUT URL for uploading
// a file directly to S3. The URL expires after 10 minutes.
func (c *Client) GeneratePresignedUploadURL(applicationID string) (string, string, error) {
	objectKey := fmt.Sprintf("applications/%s/jd.pdf", applicationID)

	req, _ := c.svc.PutObjectRequest(&s3.PutObjectInput{
		Bucket:      aws.String(c.bucket),
		Key:         aws.String(objectKey),
		ContentType: aws.String("application/pdf"),
	})

	url, err := req.Presign(10 * time.Minute)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate pre-signed URL: %w", err)
	}

	return url, objectKey, nil
}