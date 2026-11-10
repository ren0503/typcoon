package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"time"

	aws "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/ren0503/tycoon/internal/audit"
	"github.com/ren0503/tycoon/internal/auth"
	"github.com/ren0503/tycoon/internal/config"
)

func buildAuthService(
	ctx context.Context,
	cfg *config.Config,
	pool *pgxpool.Pool,
	logger *slog.Logger,
	awsCfg aws.Config,
) (auth.Service, auth.Provider, error) {
	verifier, err := auth.NewFirebaseVerifier(ctx, auth.FirebaseVerifierConfig{
		ProjectID:       cfg.Auth.Firebase.ProjectID,
		CredentialsFile: cfg.Auth.Firebase.CredentialsFile,
		EmulatorHost:    cfg.Auth.Firebase.EmulatorHost,
		AppEnv:          cfg.AppEnv,
	})
	if err != nil {
		return nil, nil, err
	}
	keys, err := buildKeySet(ctx, cfg, awsCfg)
	if err != nil {
		return nil, nil, err
	}
	limiter, err := buildRateLimiter(ctx, cfg)
	if err != nil {
		return nil, nil, err
	}
	clock := auth.SystemClock{}
	tokens, err := auth.NewAccessTokenIssuer(keys, cfg.Auth.Token.Issuer, cfg.Auth.Token.Audience, cfg.Auth.Token.AccessTTL, cfg.Auth.Token.ClockSkew, clock)
	if err != nil {
		_ = limiter.Close()
		return nil, nil, err
	}
	store := auth.NewStore(pool)
	service, err := auth.NewService(auth.ServiceOptions{
		Store:      store,
		Verifier:   verifier,
		Keys:       keys,
		Limiter:    limiter,
		Clock:      clock,
		Audit:      audit.NewStore(pool, logger),
		Logger:     logger,
		Issuer:     cfg.Auth.Token.Issuer,
		Audience:   cfg.Auth.Token.Audience,
		AccessTTL:  cfg.Auth.Token.AccessTTL,
		RefreshTTL: cfg.Auth.Token.RefreshTTL,
		ClockSkew:  cfg.Auth.Token.ClockSkew,
		RateLimits: auth.RateLimits{
			ExchangeIPPerMin:       cfg.Auth.RateLimit.ExchangeIPPerMin,
			ExchangeIdentityPerMin: cfg.Auth.RateLimit.ExchangeIdentityPerMin,
			RefreshIPPerMin:        cfg.Auth.RateLimit.RefreshIPPerMin,
			RefreshSessionPerMin:   cfg.Auth.RateLimit.RefreshSessionPerMin,
			LinkAccountPerMin:      cfg.Auth.RateLimit.LinkAccountPerMin,
			LogoutAllAccountPerMin: cfg.Auth.RateLimit.LogoutAllAccountPerMin,
		},
	})
	if err != nil {
		_ = limiter.Close()
		return nil, nil, err
	}
	provider, err := auth.NewAccessTokenProvider(tokens, store)
	if err != nil {
		_ = limiter.Close()
		return nil, nil, err
	}
	return service, provider, nil
}

func buildKeySet(ctx context.Context, cfg *config.Config, awsCfg aws.Config) (auth.KeySet, error) {
	signing := cfg.Auth.Signing
	switch signing.Mode {
	case "local":
		var signer *auth.LocalSigner
		var err error
		if signing.LocalKeyFile != "" {
			signer, err = auth.LoadLocalSigner(signing.ActiveKID, signing.LocalKeyFile)
		} else {
			if cfg.AppEnv == "production" || cfg.AppEnv == "staging" {
				return nil, errors.New("local signing without a key file is not allowed outside development/test")
			}
			signer, err = auth.NewRandomLocalSigner(signing.ActiveKID)
		}
		if err != nil {
			return nil, err
		}
		return auth.NewStaticKeySet(signing.ActiveKID, signer)
	case "kms":
		if len(signing.KMSKeys) == 0 {
			return nil, errors.New("AUTH_SIGNING_KMS_KEYS is required for KMS signing")
		}
		client := kms.NewFromConfig(awsCfg)
		kids := make([]string, 0, len(signing.KMSKeys))
		for kid := range signing.KMSKeys {
			kids = append(kids, kid)
		}
		sort.Strings(kids)
		signers := make([]auth.Signer, 0, len(signing.KMSKeys))
		for _, kid := range kids {
			signer, err := auth.NewKMSSigner(ctx, client, kid, signing.KMSKeys[kid])
			if err != nil {
				return nil, fmt.Errorf("load KMS signer %q: %w", kid, err)
			}
			signers = append(signers, signer)
		}
		return auth.NewStaticKeySet(signing.ActiveKID, signers...)
	default:
		return nil, fmt.Errorf("unsupported signing mode %q", signing.Mode)
	}
}

func buildRateLimiter(ctx context.Context, cfg *config.Config) (auth.RateLimiter, error) {
	if cfg.Auth.RateLimit.RedisURL == "" {
		return nil, errors.New("AUTH_RATE_LIMIT_REDIS_URL is required for the Redis limiter")
	}
	options, err := redis.ParseURL(cfg.Auth.RateLimit.RedisURL)
	if err != nil {
		return nil, fmt.Errorf("parse Redis rate-limit URL: %w", err)
	}
	const redisTimeout = 2 * time.Second
	options.DialTimeout = redisTimeout
	options.ReadTimeout = redisTimeout
	options.WriteTimeout = redisTimeout
	client := redis.NewClient(options)
	pingCtx, cancel := context.WithTimeout(ctx, redisTimeout)
	defer cancel()
	if err := client.Ping(pingCtx).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("connect Redis rate limiter: %w", err)
	}
	limiter, err := auth.NewRedisRateLimiter(client, cfg.Auth.RateLimit.RedisPrefix)
	if err != nil {
		_ = client.Close()
		return nil, err
	}
	return limiter, nil
}
