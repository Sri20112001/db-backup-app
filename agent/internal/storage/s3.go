package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/transfermanager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go"
)

// Upload tuning: 64 MiB parts × 4 concurrent uploads. Conservative on
// purpose — a backup agent must not saturate RAM, CPU, or the link, and
// S3-compatible services (MinIO, R2, Wasabi) all tolerate this shape.
const (
	s3PartSizeMiB  = 64
	s3Concurrency  = 4
	s3DefaultRegion = "us-east-1"
)

// S3Storage persists objects in any S3-compatible API (AWS S3, Cloudflare
// R2, Wasabi, MinIO, Backblaze B2). Keys are VaultGuard object keys
// (see BuildKey) — never URLs. Location in ObjectInfo is the key.
type S3Storage struct {
	client *s3.Client
	tm     *transfermanager.Client
	bucket string
}

// NewS3Storage validates config eagerly so misconfigured jobs fail before
// the backup runs. Region falls back to us-east-1 (ignored by most custom
// endpoints, required for signing). Endpoint switches to custom-endpoint
// mode (MinIO/R2/Wasabi/Backblaze); UsePathStyle selects path-style
// addressing (MinIO-friendly) over virtual-hosted style (AWS/R2/Wasabi).
func NewS3Storage(cfg *S3Config) (*S3Storage, error) {
	if cfg == nil {
		return nil, fmt.Errorf("S3 storage target has no configuration")
	}
	if strings.TrimSpace(cfg.Bucket) == "" {
		return nil, fmt.Errorf("S3 storage target has no bucket configured")
	}
	if strings.TrimSpace(cfg.AccessKey) == "" || strings.TrimSpace(cfg.SecretKey) == "" {
		return nil, fmt.Errorf("S3 storage target has no credentials configured")
	}
	region := strings.TrimSpace(cfg.Region)
	if region == "" {
		region = s3DefaultRegion
	}
	awsCfg := aws.Config{
		Region:      region,
		Credentials: credentials.NewStaticCredentialsProvider(cfg.AccessKey, cfg.SecretKey, ""),
	}
	var optFns []func(*s3.Options)
	if endpoint := strings.TrimSpace(cfg.Endpoint); endpoint != "" {
		optFns = append(optFns, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(endpoint)
		})
	}
	if cfg.UsePathStyle {
		optFns = append(optFns, func(o *s3.Options) {
			o.UsePathStyle = true
		})
	}
	client := s3.NewFromConfig(awsCfg, optFns...)
	tm := transfermanager.New(client, func(o *transfermanager.Options) {
		o.PartSizeBytes = s3PartSizeMiB << 20
		o.Concurrency = s3Concurrency
	})
	return &S3Storage{client: client, tm: tm, bucket: cfg.Bucket}, nil
}

// Put uploads via the multipart-aware manager (single PUT for small
// objects, concurrent parts above the part size). Retries follow the SDK
// defaults: network timeouts, 429, and 5xx are retried with backoff and
// jitter; 403/404 and credential errors are not.
//
// The SHA-256 travels as object metadata (S3 ETags are NOT content hashes
// for multipart uploads) — the caller supplies it in opts.Metadata and
// Head() compares it on verify.
func (s *S3Storage) Put(ctx context.Context, key string, r io.Reader, size int64, opts PutOptions) (*ObjectInfo, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if strings.TrimSpace(key) == "" {
		return nil, fmt.Errorf("S3 object key is empty")
	}
	meta := map[string]string{MetaVersion: FormatVersion}
	for k, v := range opts.Metadata {
		meta[k] = v
	}
	contentType := opts.ContentType
	if contentType == "" {
		contentType = ContentTypeForName(key)
	}
	out, err := s.tm.UploadObject(ctx, &transfermanager.UploadObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		Body:        r,
		ContentType: aws.String(contentType),
		Metadata:    meta,
	})
	if err != nil {
		return nil, fmt.Errorf("S3 upload %s: %w", key, err)
	}
	return &ObjectInfo{
		Key:            key,
		Size:           size,
		ETag:           strings.Trim(aws.ToString(out.ETag), `"`),
		ChecksumSHA256: meta[MetaChecksumSHA256],
		Location:       key,
		Metadata:       meta,
	}, nil
}

// Get downloads the object for restore / verification.
func (s *S3Storage) Get(ctx context.Context, key string) (io.ReadCloser, *ObjectInfo, error) {
	out, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, nil, fmt.Errorf("S3 download %s: %w", key, err)
	}
	var size int64
	if out.ContentLength != nil {
		size = *out.ContentLength
	}
	return out.Body, &ObjectInfo{
		Key:            key,
		Size:           size,
		ETag:           strings.Trim(aws.ToString(out.ETag), `"`),
		ChecksumSHA256: out.Metadata[MetaChecksumSHA256],
		Location:       key,
		Metadata:       out.Metadata,
	}, nil
}

// Delete removes the object. S3 deletes are idempotent (no error when the
// key is already gone), which keeps retention cleanup safe to retry.
func (s *S3Storage) Delete(ctx context.Context, key string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("S3 delete %s: %w", key, err)
	}
	return nil
}

// Head returns size, ETag, and VaultGuard metadata without the body — the
// basis of post-upload verification.
func (s *S3Storage) Head(ctx context.Context, key string) (*ObjectInfo, error) {
	out, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("S3 head %s: %w", key, err)
	}
	var size int64
	if out.ContentLength != nil {
		size = *out.ContentLength
	}
	return &ObjectInfo{
		Key:            key,
		Size:           size,
		ETag:           strings.Trim(aws.ToString(out.ETag), `"`),
		ChecksumSHA256: out.Metadata[MetaChecksumSHA256],
		Location:       key,
		Metadata:       out.Metadata,
	}, nil
}

// IsNotFound reports whether err is a missing-object error (404 on
// Head/Get), so callers can distinguish "gone" from "broken".
func IsNotFound(err error) bool {
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		code := apiErr.ErrorCode()
		return code == "NotFound" || code == "NoSuchKey" || code == "NoSuchBucket"
	}
	return false
}
