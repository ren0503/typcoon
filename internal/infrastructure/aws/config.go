package awsconfig

import (
	"context"
	"fmt"

	aws "github.com/aws/aws-sdk-go-v2/aws"
	sdkconfig "github.com/aws/aws-sdk-go-v2/config"

	"github.com/ren0503/tycoon/internal/platform/config"
)

// Load builds reusable AWS SDK configuration using the default credential provider chain.
// It creates no service clients and makes no AWS API calls.
func Load(ctx context.Context, cfg config.AWSConfig) (aws.Config, error) {
	loaded, err := sdkconfig.LoadDefaultConfig(ctx, sdkconfig.WithRegion(cfg.Region))
	if err != nil {
		return aws.Config{}, fmt.Errorf("load aws configuration: %w", err)
	}

	if cfg.EndpointURL != "" {
		loaded.BaseEndpoint = aws.String(cfg.EndpointURL)
	}

	return loaded, nil
}
