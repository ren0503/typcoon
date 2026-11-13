package app

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
	goredis "github.com/redis/go-redis/v9"

	appauth "github.com/ren0503/tycoon/internal/application/auth"
	"github.com/ren0503/tycoon/internal/infrastructure/aws"
	redisadapter "github.com/ren0503/tycoon/internal/infrastructure/cache/redis"
	"github.com/ren0503/tycoon/internal/infrastructure/health"
	"github.com/ren0503/tycoon/internal/infrastructure/identity/firebase"
	"github.com/ren0503/tycoon/internal/infrastructure/persistence/postgres"
	airepo "github.com/ren0503/tycoon/internal/infrastructure/persistence/postgres/ai"
	authrepo "github.com/ren0503/tycoon/internal/infrastructure/persistence/postgres/auth"
	bookingrepo "github.com/ren0503/tycoon/internal/infrastructure/persistence/postgres/booking"
	feedbackrepo "github.com/ren0503/tycoon/internal/infrastructure/persistence/postgres/feedback"
	guestrepo "github.com/ren0503/tycoon/internal/infrastructure/persistence/postgres/guest"
	partnerrepo "github.com/ren0503/tycoon/internal/infrastructure/persistence/postgres/partner"
	shiprepo "github.com/ren0503/tycoon/internal/infrastructure/persistence/postgres/ship"
	staffrepo "github.com/ren0503/tycoon/internal/infrastructure/persistence/postgres/staff"
	syncrepo "github.com/ren0503/tycoon/internal/infrastructure/persistence/postgres/sync"
	tenantrepo "github.com/ren0503/tycoon/internal/infrastructure/persistence/postgres/tenant"
	voyagerepo "github.com/ren0503/tycoon/internal/infrastructure/persistence/postgres/voyage"
	securitykms "github.com/ren0503/tycoon/internal/infrastructure/security/kms"
	"github.com/ren0503/tycoon/internal/infrastructure/security/local"
	"github.com/ren0503/tycoon/internal/platform/config"
	httpserver "github.com/ren0503/tycoon/internal/transport/http"
	"github.com/ren0503/tycoon/internal/transport/http/middleware"
)

// RunAPI assembles and runs the HTTP API runtime.
func RunAPI(ctx context.Context, cfg *config.Config, logger *slog.Logger) error {
	db, err := postgres.New(ctx, cfg.Database)
	if err != nil {
		return err
	}

	awsCfg, err := awsconfig.Load(ctx, cfg.AWS)
	if err != nil {
		db.Close()
		return err
	}

	authService, provider, err := buildAuthService(ctx, cfg, db, logger, awsCfg)
	if err != nil {
		db.Close()
		return err
	}
	defer func() { _ = authService.Close() }()

	documentCipher, err := local.NewDevDocumentCipher()
	if err != nil {
		db.Close()
		return fmt.Errorf("guest document cipher: %w", err)
	}

	modules := BuildModules(ModuleDeps{
		Logger:         logger,
		Auth:           authService,
		Accounts:       authrepo.NewAccountService(db),
		TenantRepo:     tenantrepo.New(db),
		ShipRepo:       shiprepo.New(db),
		VoyageRepo:     voyagerepo.New(db),
		StaffRepo:      staffrepo.New(db),
		PartnerRepo:    partnerrepo.New(db),
		BookingRepo:    bookingrepo.New(db),
		GuestRepo:      guestrepo.New(db),
		AIRepo:         airepo.New(db),
		FeedbackRepo:   feedbackrepo.New(db),
		SyncRepo:       syncrepo.New(db),
		DocumentCipher: documentCipher,
	})

	router := httpserver.NewRouter(httpserver.RouterConfig{
		Logger:             logger,
		DB:                 db,
		CORSAllowedOrigins: cfg.CORS.AllowedOrigins,
		Provider:           provider,
		PublicAPIRoutes:    modules.RegisterPublicRoutes,
		APIRoutes:          modules.RegisterRoutes,
	})
	server := httpserver.NewServer(cfg, router)
	monitor := health.New(db, logger)

	return New(cfg, logger, db, server, monitor).Run(ctx)
}

func buildAuthService(
	ctx context.Context,
	cfg *config.Config,
	pool *pgxpool.Pool,
	logger *slog.Logger,
	awsCfg aws.Config,
) (appauth.Service, middleware.Provider, error) {
	verifier, err := firebase.NewFirebaseVerifier(ctx, firebase.FirebaseVerifierConfig{
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
	clock := appauth.SystemClock{}
	tokens, err := appauth.NewAccessTokenIssuer(keys, cfg.Auth.Token.Issuer, cfg.Auth.Token.Audience, cfg.Auth.Token.AccessTTL, cfg.Auth.Token.ClockSkew, clock)
	if err != nil {
		_ = limiter.Close()
		return nil, nil, err
	}
	store := authrepo.NewStore(pool)
	service, err := appauth.NewService(appauth.ServiceOptions{
		Store:      store,
		Verifier:   verifier,
		Keys:       keys,
		Limiter:    limiter,
		Clock:      clock,
		Logger:     logger,
		Issuer:     cfg.Auth.Token.Issuer,
		Audience:   cfg.Auth.Token.Audience,
		AccessTTL:  cfg.Auth.Token.AccessTTL,
		RefreshTTL: cfg.Auth.Token.RefreshTTL,
		ClockSkew:  cfg.Auth.Token.ClockSkew,
		RateLimits: appauth.RateLimits{
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
	provider, err := middleware.NewAccessTokenProvider(tokens, store)
	if err != nil {
		_ = service.Close()
		return nil, nil, err
	}
	resolved, err := resolveProvider(cfg, provider)
	if err != nil {
		_ = service.Close()
		return nil, nil, err
	}
	return service, resolved, nil
}

func resolveProvider(cfg *config.Config, tokenProvider middleware.Provider) (middleware.Provider, error) {
	if cfg.Auth.Mode == "dev" {
		return middleware.NewDevHeaderProvider(cfg.Auth.DevPrincipals)
	}
	if tokenProvider != nil {
		return tokenProvider, nil
	}
	return middleware.DenyAllProvider{}, nil
}

func buildKeySet(ctx context.Context, cfg *config.Config, awsCfg aws.Config) (appauth.KeySet, error) {
	signing := cfg.Auth.Signing
	switch signing.Mode {
	case "local":
		var signer *local.LocalSigner
		var err error
		if signing.LocalKeyFile != "" {
			signer, err = local.LoadLocalSigner(signing.ActiveKID, signing.LocalKeyFile)
		} else {
			if cfg.AppEnv == "production" || cfg.AppEnv == "staging" {
				return nil, errors.New("local signing without a key file is not allowed outside development/test")
			}
			signer, err = local.NewRandomLocalSigner(signing.ActiveKID)
		}
		if err != nil {
			return nil, err
		}
		return local.NewStaticKeySet(signing.ActiveKID, signer)
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
		signers := make([]appauth.Signer, 0, len(signing.KMSKeys))
		for _, kid := range kids {
			signer, err := securitykms.NewKMSSigner(ctx, client, kid, signing.KMSKeys[kid])
			if err != nil {
				return nil, fmt.Errorf("load KMS signer %q: %w", kid, err)
			}
			signers = append(signers, signer)
		}
		return local.NewStaticKeySet(signing.ActiveKID, signers...)
	default:
		return nil, fmt.Errorf("unsupported signing mode %q", signing.Mode)
	}
}

func buildRateLimiter(ctx context.Context, cfg *config.Config) (appauth.RateLimiter, error) {
	if cfg.Auth.RateLimit.RedisURL == "" {
		return nil, errors.New("AUTH_RATE_LIMIT_REDIS_URL is required for the Redis limiter")
	}
	options, err := goredis.ParseURL(cfg.Auth.RateLimit.RedisURL)
	if err != nil {
		return nil, fmt.Errorf("parse Redis rate-limit URL: %w", err)
	}
	const redisTimeout = 2 * time.Second
	options.DialTimeout = redisTimeout
	options.ReadTimeout = redisTimeout
	options.WriteTimeout = redisTimeout
	client := goredis.NewClient(options)
	pingCtx, cancel := context.WithTimeout(ctx, redisTimeout)
	defer cancel()
	if err := client.Ping(pingCtx).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("connect Redis rate limiter: %w", err)
	}
	limiter, err := redisadapter.NewRateLimiter(client, cfg.Auth.RateLimit.RedisPrefix)
	if err != nil {
		_ = client.Close()
		return nil, err
	}
	return limiter, nil
}
