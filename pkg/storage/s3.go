package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awscfg "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"

	"github.com/shashiranjanraj/kashvi/config"
)

// s3Disk is the S3-compatible object storage driver.
// Works with AWS S3, MinIO, DigitalOcean Spaces, Cloudflare R2.
type s3Disk struct {
	client  *s3.Client
	bucket  string
	baseURL string
	region  string
}

func newS3Disk() (*s3Disk, error) {
	bucket := config.Get("S3_BUCKET", "")
	region := config.Get("S3_REGION", "us-east-1")
	key := config.Get("S3_KEY", "")
	secret := config.Get("S3_SECRET", "")
	endpoint := config.Get("S3_ENDPOINT", "") // leave empty for real AWS
	baseURL := strings.TrimRight(config.Get("S3_URL", ""), "/")

	if bucket == "" {
		return nil, fmt.Errorf("storage/s3: S3_BUCKET is not configured")
	}

	opts := []func(*awscfg.LoadOptions) error{
		awscfg.WithRegion(region),
	}

	// Static credentials (required for MinIO / R2 / Spaces)
	if key != "" && secret != "" {
		opts = append(opts, awscfg.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(key, secret, ""),
		))
	}

	cfg, err := awscfg.LoadDefaultConfig(context.Background(), opts...)
	if err != nil {
		return nil, fmt.Errorf("storage/s3: load config: %w", err)
	}

	clientOpts := []func(*s3.Options){}
	if endpoint != "" {
		clientOpts = append(clientOpts, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(endpoint)
			o.UsePathStyle = true // required for MinIO
		})
	}
	if baseURL == "" {
		baseURL = fmt.Sprintf("https://%s.s3.%s.amazonaws.com", bucket, region)
	}

	return &s3Disk{
		client:  s3.NewFromConfig(cfg, clientOpts...),
		bucket:  bucket,
		baseURL: baseURL,
		region:  region,
	}, nil
}

// ── Write ─────────────────────────────────────────────────────────────────────

func (d *s3Disk) Put(path string, content []byte) error {
	return d.PutStream(path, bytes.NewReader(content))
}

func (d *s3Disk) PutStream(path string, r io.Reader) error {
	data, err := io.ReadAll(r)
	if err != nil {
		return fmt.Errorf("storage/s3: read: %w", err)
	}
	_, err = d.client.PutObject(context.Background(), &s3.PutObjectInput{
		Bucket: aws.String(d.bucket),
		Key:    aws.String(path),
		Body:   bytes.NewReader(data),
	})
	if err != nil {
		return fmt.Errorf("storage/s3: put %s: %w", path, err)
	}
	return nil
}

// ── Read ──────────────────────────────────────────────────────────────────────

func (d *s3Disk) Get(path string) ([]byte, error) {
	rc, err := d.GetStream(path)
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	return io.ReadAll(rc)
}

func (d *s3Disk) GetStream(path string) (io.ReadCloser, error) {
	out, err := d.client.GetObject(context.Background(), &s3.GetObjectInput{
		Bucket: aws.String(d.bucket),
		Key:    aws.String(path),
	})
	if err != nil {
		return nil, fmt.Errorf("storage/s3: get %s: %w", path, err)
	}
	return out.Body, nil
}

// ── Metadata ──────────────────────────────────────────────────────────────────

func (d *s3Disk) Exists(path string) bool {
	_, err := d.client.HeadObject(context.Background(), &s3.HeadObjectInput{
		Bucket: aws.String(d.bucket),
		Key:    aws.String(path),
	})
	return err == nil
}

func (d *s3Disk) Missing(path string) bool { return !d.Exists(path) }

func (d *s3Disk) Size(path string) (int64, error) {
	out, err := d.client.HeadObject(context.Background(), &s3.HeadObjectInput{
		Bucket: aws.String(d.bucket),
		Key:    aws.String(path),
	})
	if err != nil {
		return 0, fmt.Errorf("storage/s3: head %s: %w", path, err)
	}
	if out.ContentLength == nil {
		return 0, nil
	}
	return *out.ContentLength, nil
}

func (d *s3Disk) LastModified(path string) (time.Time, error) {
	out, err := d.client.HeadObject(context.Background(), &s3.HeadObjectInput{
		Bucket: aws.String(d.bucket),
		Key:    aws.String(path),
	})
	if err != nil {
		return time.Time{}, fmt.Errorf("storage/s3: head %s: %w", path, err)
	}
	if out.LastModified == nil {
		return time.Time{}, nil
	}
	return *out.LastModified, nil
}

func (d *s3Disk) URL(path string) string {
	return d.baseURL + "/" + strings.TrimLeft(path, "/")
}

// ── Delete ────────────────────────────────────────────────────────────────────

func (d *s3Disk) Delete(path string) error {
	_, err := d.client.DeleteObject(context.Background(), &s3.DeleteObjectInput{
		Bucket: aws.String(d.bucket),
		Key:    aws.String(path),
	})
	if err != nil {
		return fmt.Errorf("storage/s3: delete %s: %w", path, err)
	}
	return nil
}

// ── Copy / Move ───────────────────────────────────────────────────────────────

func (d *s3Disk) Copy(src, dst string) error {
	_, err := d.client.CopyObject(context.Background(), &s3.CopyObjectInput{
		Bucket:     aws.String(d.bucket),
		CopySource: aws.String(d.bucket + "/" + src),
		Key:        aws.String(dst),
	})
	if err != nil {
		return fmt.Errorf("storage/s3: copy %s → %s: %w", src, dst, err)
	}
	return nil
}

func (d *s3Disk) Move(src, dst string) error {
	if err := d.Copy(src, dst); err != nil {
		return err
	}
	return d.Delete(src)
}

// ── Directory listing ─────────────────────────────────────────────────────────

func (d *s3Disk) Files(directory string) ([]string, error) {
	return d.list(directory, "/")
}

func (d *s3Disk) AllFiles(directory string) ([]string, error) {
	return d.list(directory, "")
}

func (d *s3Disk) list(prefix, delimiter string) ([]string, error) {
	pfx := strings.TrimLeft(prefix, "/")
	if pfx != "" && !strings.HasSuffix(pfx, "/") {
		pfx += "/"
	}
	input := &s3.ListObjectsV2Input{
		Bucket: aws.String(d.bucket),
		Prefix: aws.String(pfx),
	}
	if delimiter != "" {
		input.Delimiter = aws.String(delimiter)
	}

	var keys []string
	paginator := s3.NewListObjectsV2Paginator(d.client, input)
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(context.Background())
		if err != nil {
			return nil, fmt.Errorf("storage/s3: list %s: %w", prefix, err)
		}
		for _, obj := range page.Contents {
			keys = append(keys, *obj.Key)
		}
	}
	return keys, nil
}

func (d *s3Disk) Directories(directory string) ([]string, error) {
	pfx := strings.TrimLeft(directory, "/")
	if pfx != "" && !strings.HasSuffix(pfx, "/") {
		pfx += "/"
	}
	out, err := d.client.ListObjectsV2(context.Background(), &s3.ListObjectsV2Input{
		Bucket:    aws.String(d.bucket),
		Prefix:    aws.String(pfx),
		Delimiter: aws.String("/"),
	})
	if err != nil {
		return nil, fmt.Errorf("storage/s3: directories %s: %w", directory, err)
	}
	var dirs []string
	for _, cp := range out.CommonPrefixes {
		if cp.Prefix != nil {
			dirs = append(dirs, *cp.Prefix)
		}
	}
	return dirs, nil
}

// S3 has no real directory concept — these are no-ops.
func (d *s3Disk) MakeDirectory(_ string) error { return nil }
func (d *s3Disk) DeleteDirectory(path string) error {
	keys, err := d.AllFiles(path)
	if err != nil {
		return err
	}
	if len(keys) == 0 {
		return nil
	}
	objs := make([]types.ObjectIdentifier, len(keys))
	for i, k := range keys {
		k := k
		objs[i] = types.ObjectIdentifier{Key: &k}
	}
	_, err = d.client.DeleteObjects(context.Background(), &s3.DeleteObjectsInput{
		Bucket: aws.String(d.bucket),
		Delete: &types.Delete{Objects: objs},
	})
	return err
}
