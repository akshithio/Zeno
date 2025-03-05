package lq

import (
	"context"
	"database/sql"
	_ "embed"
	"path"

	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"

	"github.com/google/uuid"
	"github.com/internetarchive/Zeno/internal/pkg/config"
	"github.com/internetarchive/Zeno/internal/pkg/source/lq/sqlc_model"
)

type LQClient struct {
	dbWrite     *sql.DB
	dbWriteSqlc *sqlc_model.Queries
}

//go:embed schema.sql
var ddl string

func Init(job string) (*LQClient, error) {
	dbWrite, err := sql.Open("sqlite3", "file:"+path.Join(config.Get().JobPath, "lq.db"))
	if err != nil {
		return nil, err
	}
	dbWrite.SetMaxOpenConns(1)

	if _, err := dbWrite.Exec(ddl); err != nil {
		logger.Error("error creating lq database schema", "err", err.Error(), "func", "lq.Init")
		return nil, err
	}

	dbWriteSqlc := sqlc_model.New(dbWrite)

	return &LQClient{
		dbWrite:     dbWrite,
		dbWriteSqlc: dbWriteSqlc,
	}, nil
}

func (c *LQClient) ResetURL(ctx context.Context, seed string) error {
	return c.dbWriteSqlc.ResetURL(ctx, seed)
}

func (c *LQClient) Get(ctx context.Context, limit int) ([]sqlc_model.Url, error) {
	tx, err := globalLQ.client.dbWrite.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	qtx := globalLQ.client.dbWriteSqlc.WithTx(tx)

	freshUrls, err := qtx.GetFreshURLs(ctx, int64(limit))
	if err != nil {
		return nil, err
	}

	for _, record := range freshUrls {
		if err = qtx.ClaimThisURL(ctx, record.ID); err != nil {
			logger.Error("error claiming URL", "err", err.Error(), "func", "lq.getURLs", "id", record.ID)
			return nil, err
		}
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}

	return freshUrls, nil
}

func (c *LQClient) Add(ctx context.Context, urls []sqlc_model.Url, bypassSeencheck bool) error {
	tx, err := globalLQ.client.dbWrite.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	qtx := globalLQ.client.dbWriteSqlc.WithTx(tx)

	for _, url := range urls {
		if url.ID == "" {
			url.ID = uuid.New().String()
		}
		err = qtx.AddURL(ctx, sqlc_model.AddURLParams{
			ID:    url.ID,
			Value: url.Value,
			Via:   url.Via,
			Hops:  int64(url.Hops),
		})
		if err != nil {
			if err.Error() == "sqlite3: constraint failed: UNIQUE constraint failed: urls.value" {
				logger.Debug("URL.Value already exists in LQ", "value", url.Value, "via", url.Via)
				continue
			}
			logger.Error("error adding URL", "err", err.Error(), "func", "lq.Add", "value", url.Value, "via", url.Via)
			return err
		}
	}

	if err = tx.Commit(); err != nil {
		return err
	}

	return nil
}

func (c *LQClient) Delete(ctx context.Context, urls []sqlc_model.Url, bypassSeencheck bool) error {
	tx, err := globalLQ.client.dbWrite.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	qtx := globalLQ.client.dbWriteSqlc.WithTx(tx)

	for _, url := range urls {
		err = qtx.DoneURL(ctx, url.ID)
		if err != nil {
			logger.Error("error deleting URL", "err", err.Error(), "func", "lq.Delete", "id", url.ID)
			return err
		}
	}

	if err = tx.Commit(); err != nil {
		return err
	}
	return nil
}
