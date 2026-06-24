package history

import (
	"context"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/pressly/goose/v3"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

type RestRequest struct {
	ID        int    `gorm:"column:id;primaryKey;autoIncrement"`
	Username  string `gorm:"column:username;index:username_idx"`
	Path      string `gorm:"column:path;not null"`
	Method    string `gorm:"column:method;not null"`
	Body      string `gorm:"column:body;not null"`
	MD5       string `gorm:"column:md5;not null;uniqueIndex:md5_idx"`
	CreatedAt int64  `gorm:"column:created_at"`
}

func (RestRequest) TableName() string { return "rest_requests" }

type Store struct {
	db          *gorm.DB
	historySize int
}

func Open(dbPath string, historySize int) (*Store, error) {
	gormDB, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, fmt.Errorf("open sqlite %s: %w", dbPath, err)
	}
	sqlDB, err := gormDB.DB()
	if err != nil {
		return nil, err
	}
	goose.SetBaseFS(migrationsFS)
	if err := goose.SetDialect("sqlite3"); err != nil {
		return nil, err
	}
	if err := goose.Up(sqlDB, "migrations"); err != nil {
		return nil, fmt.Errorf("run migrations: %w", err)
	}
	if historySize <= 0 {
		historySize = 50
	}
	return &Store{db: gormDB, historySize: historySize}, nil
}

func (s *Store) Close() error {
	d, err := s.db.DB()
	if err != nil {
		return err
	}
	return d.Close()
}

func (s *Store) Save(ctx context.Context, username, path, method, body string) error {
	body = RedactBody(body)
	req := RestRequest{
		Username:  username,
		Path:      path,
		Method:    method,
		Body:      body,
		MD5:       hashSig(path, method, body, username),
		CreatedAt: time.Now().UnixMilli(),
	}
	tx := s.db.WithContext(ctx).Where("md5 = ?", req.MD5).First(&RestRequest{})
	if tx.Error != nil && !errors.Is(tx.Error, gorm.ErrRecordNotFound) {
		return tx.Error
	}
	if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
		if err := s.db.WithContext(ctx).Create(&req).Error; err != nil {
			return err
		}
		s.trim(ctx, username)
		return nil
	}
	return s.db.WithContext(ctx).
		Model(&RestRequest{}).
		Where("md5 = ?", req.MD5).
		Update("created_at", req.CreatedAt).Error
}

func (s *Store) trim(ctx context.Context, username string) {
	s.db.WithContext(ctx).Exec(`
		DELETE FROM rest_requests
		WHERE id IN (
			SELECT id FROM rest_requests
			WHERE username = ?
			ORDER BY created_at DESC
			LIMIT -1 OFFSET ?
		)
	`, username, s.historySize)
}

func (s *Store) All(ctx context.Context, username string) ([]RestRequest, error) {
	var rows []RestRequest
	if err := s.db.WithContext(ctx).
		Where("username = ?", username).
		Order("created_at DESC").
		Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func (s *Store) Clear(ctx context.Context, username string) (int64, error) {
	tx := s.db.WithContext(ctx).Where("username = ?", username).Delete(&RestRequest{})
	return tx.RowsAffected, tx.Error
}

func hashSig(path, method, body, username string) string {
	h := sha256.Sum256([]byte(path + method + body + username))
	return hex.EncodeToString(h[:])
}

const maxStoredBodyBytes = 32 << 10

var sensitiveKeys = map[string]bool{
	"access_key":    true,
	"access_token":  true,
	"api_key":       true,
	"authorization": true,
	"bind_pw":       true,
	"client_secret": true,
	"credentials":   true,
	"key":           true,
	"pass":          true,
	"password":      true,
	"refresh_token": true,
	"secret":        true,
	"service_token": true,
	"token":         true,
}

func RedactBody(body string) string {
	body = strings.TrimSpace(body)
	if body == "" || body == "{}" {
		return "{}"
	}
	var value any
	if err := json.Unmarshal([]byte(body), &value); err == nil {
		redactJSON(value)
		raw, err := json.Marshal(value)
		if err == nil {
			body = string(raw)
		}
	}
	if len(body) > maxStoredBodyBytes {
		return body[:maxStoredBodyBytes] + "...<truncated>"
	}
	return body
}

func redactJSON(value any) {
	switch v := value.(type) {
	case map[string]any:
		for k, child := range v {
			if sensitiveKeys[strings.ToLower(k)] {
				v[k] = "<redacted>"
				continue
			}
			redactJSON(child)
		}
	case []any:
		for _, child := range v {
			redactJSON(child)
		}
	}
}
