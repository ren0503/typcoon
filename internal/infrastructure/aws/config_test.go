package awsconfig_test

import (
	"context"
	"testing"

	"github.com/ren0503/tycoon/internal/infrastructure/aws"
	"github.com/ren0503/tycoon/internal/platform/config"
)

func clearAWSCredentials(t *testing.T) {
	t.Helper()
	t.Setenv("AWS_EC2_METADATA_DISABLED", "true")
	t.Setenv("AWS_ACCESS_KEY_ID", "")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "")
	t.Setenv("AWS_SESSION_TOKEN", "")
	t.Setenv("AWS_PROFILE", "")
	t.Setenv("AWS_CONFIG_FILE", "")
	t.Setenv("AWS_SHARED_CREDENTIALS_FILE", "")
}

func TestLoadWithoutCredentials(t *testing.T) {
	clearAWSCredentials(t)

	loaded, err := awsconfig.Load(context.Background(), config.AWSConfig{Region: "eu-west-1"})
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if loaded.Region != "eu-west-1" {
		t.Fatalf("region = %q", loaded.Region)
	}
	if loaded.BaseEndpoint != nil {
		t.Fatalf("BaseEndpoint = %v", *loaded.BaseEndpoint)
	}
}

func TestLoadWithEndpointOverride(t *testing.T) {
	clearAWSCredentials(t)

	loaded, err := awsconfig.Load(context.Background(), config.AWSConfig{
		Region:      "us-east-1",
		EndpointURL: "http://localhost:4566",
	})
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if loaded.BaseEndpoint == nil {
		t.Fatal("BaseEndpoint is nil")
	}
	if *loaded.BaseEndpoint != "http://localhost:4566" {
		t.Fatalf("BaseEndpoint = %q", *loaded.BaseEndpoint)
	}
}
