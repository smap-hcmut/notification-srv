package httpserver

import (
	"database/sql"
	"errors"

	"smap-api/pkg/discord"
	"smap-api/pkg/encrypter"
	"smap-api/pkg/log"
	miniopkg "smap-api/pkg/minio"

	"github.com/gin-gonic/gin"
)

const (
	productionMode = "production"
	debugMode      = "debug"
)

var (
	ginDebugMode   = gin.DebugMode
	ginReleaseMode = gin.ReleaseMode
	ginTestMode    = gin.TestMode
)

type HTTPServer struct {
	// Server Configuration
	gin  *gin.Engine
	l    log.Logger
	host string
	port int
	mode string

	// Database Configuration
	postgresDB *sql.DB

	// Storage Configuration
	minio miniopkg.MinIO

	// Authentication & Security Configuration
	jwtSecretKey string
	encrypter    encrypter.Encrypter
	internalKey  string

	// Monitoring & Notification Configuration
	discord *discord.Discord
}

type Config struct {
	// Server Configuration
	Logger log.Logger
	Host   string
	Port   int
	Mode   string

	// Database Configuration
	PostgresDB *sql.DB

	// Storage Configuration
	MinIO miniopkg.MinIO

	// Authentication & Security Configuration
	JwtSecretKey string
	Encrypter    encrypter.Encrypter
	InternalKey  string

	// Monitoring & Notification Configuration
	Discord *discord.Discord
}

// New creates a new HTTPServer instance with the provided configuration.
func New(logger log.Logger, cfg Config) (*HTTPServer, error) {
	gin.SetMode(cfg.Mode)

	srv := &HTTPServer{
		// Server Configuration
		l:    logger,
		gin:  gin.Default(),
		host: cfg.Host,
		port: cfg.Port,
		mode: cfg.Mode,

		// Database Configuration
		postgresDB: cfg.PostgresDB,

		// Storage Configuration
		minio: cfg.MinIO,

		// Authentication & Security Configuration
		jwtSecretKey: cfg.JwtSecretKey,
		encrypter:    cfg.Encrypter,
		internalKey:  cfg.InternalKey,

		// Monitoring & Notification Configuration
		discord: cfg.Discord,
	}

	if err := srv.validate(); err != nil {
		return nil, err
	}

	return srv, nil
}

// validate validates that all required dependencies are provided.
func (srv HTTPServer) validate() error {
	// Server Configuration
	if srv.l == nil {
		return errors.New("logger is required")
	}
	if srv.mode == "" {
		return errors.New("mode is required")
	}
	if srv.host == "" {
		return errors.New("host is required")
	}
	if srv.port == 0 {
		return errors.New("port is required")
	}

	// Database Configuration
	if srv.postgresDB == nil {
		return errors.New("postgresDB is required")
	}

	// Storage Configuration
	if srv.minio == nil {
		return errors.New("minio is required")
	}

	// Authentication & Security Configuration
	if srv.jwtSecretKey == "" {
		return errors.New("jwtSecretKey is required")
	}
	if srv.encrypter == nil {
		return errors.New("encrypter is required")
	}
	if srv.internalKey == "" {
		return errors.New("internalKey is required")
	}

	// Monitoring & Notification Configuration
	if srv.discord == nil {
		return errors.New("discord is required")
	}

	return nil
}
