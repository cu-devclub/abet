package postgres

import (
	"fmt"
	"strings"

	"github.com/exaring/otelpgx"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DBConfig holds the Postgres connection settings as separate values. No URL is ever built from them, so a password
// with any character (@ : / ? # % & = ' " \ space) works.
type DBConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
	SSLMode  string // disable (default), allow, prefer, require, verify-ca, verify-full
}

// PoolConfig builds the pgx pool configuration. Only host, port and sslmode go through a keyword/value string (they decide
// the TLS setup and are quoted); user, database and password are assigned directly on the parsed config, so none of them
// is ever parsed, URL-encoded or interpolated.
func (c DBConfig) PoolConfig() (*pgxpool.Config, error) {
	sslmode := c.SSLMode
	if sslmode == "" {
		sslmode = "disable"
	}
	var parts []string
	for _, kv := range [][2]string{{"host", c.Host}, {"port", c.Port}, {"sslmode", sslmode}} {
		if kv[1] != "" {
			parts = append(parts, kv[0]+"="+quoteDSNValue(kv[1]))
		}
	}
	cfg, err := pgxpool.ParseConfig(strings.Join(parts, " "))
	if err != nil {
		return nil, fmt.Errorf("invalid database settings: %w", err)
	}
	cfg.ConnConfig.User = c.User
	cfg.ConnConfig.Database = c.Name
	cfg.ConnConfig.Password = c.Password
	cfg.ConnConfig.Tracer = otelpgx.NewTracer(otelpgx.WithTrimSQLInSpanName()) // no SQL parameter values are recorded
	return cfg, nil
}

// quoteDSNValue single-quotes a keyword/value DSN value, escaping backslash and single quote.
func quoteDSNValue(v string) string {
	v = strings.ReplaceAll(v, `\`, `\`)
	v = strings.ReplaceAll(v, `'`, `\'`)
	return "'" + v + "'"
}
