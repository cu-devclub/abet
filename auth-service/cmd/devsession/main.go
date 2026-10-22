// Command devsession mints a login session for an existing user WITHOUT Google (development only).
//
//	go run ./cmd/devsession user@chula.ac.th
//
// It asks main-service (login-sync) for the user id, inserts a session row into auth_db and prints
// `abet_session=<jwt>` on stdout. It refuses to run unless APP_ENV=development and is not part of the
// server binary (the production image builds only ./cmd/api).
package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"abet/auth-service/config"
	"abet/auth-service/internal/adapter/inbound/http/handler"
	"abet/auth-service/internal/adapter/outbound/jwt"
	"abet/auth-service/internal/adapter/outbound/mainclient"
	"abet/auth-service/internal/adapter/outbound/postgres"
	"abet/auth-service/internal/core/domain"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "devsession:", err)
		os.Exit(1)
	}
}

func run() error {
	if os.Getenv("APP_ENV") != "development" {
		return fmt.Errorf("refusing to run: APP_ENV must be 'development'")
	}
	if len(os.Args) != 2 || !strings.Contains(os.Args[1], "@") {
		return fmt.Errorf("usage: devsession <email>")
	}
	email := strings.ToLower(strings.TrimSpace(os.Args[1]))

	cfg, err := config.Load()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	sync, err := mainclient.NewHTTPClient(cfg.MainServiceURL, cfg.InternalAPIToken, cfg.MainServiceTimeout).LoginSync(ctx, email, "", "")
	if err != nil {
		return fmt.Errorf("login-sync for %s (is the user provisioned?): %w", email, err)
	}

	pool, err := postgres.Connect(ctx, cfg.DB)
	if err != nil {
		return err
	}
	defer pool.Close()

	now := time.Now().UTC()
	session, err := domain.NewSession(sync.UserID, email, "devsession", "127.0.0.1", cfg.SessionTTL, now)
	if err != nil {
		return err
	}
	if err := postgres.NewSessionRepository(pool).Create(ctx, session); err != nil {
		return err
	}
	token, err := jwt.NewTokenService(cfg.SessionSecret).Issue(domain.SessionClaims{
		UserID: session.UserID, Email: email, SessionID: session.ID, IssuedAt: now, ExpiresAt: session.ExpiresAt,
	})
	if err != nil {
		return err
	}
	fmt.Printf("%s=%s\n", handler.SessionCookieName, token)
	return nil
}
