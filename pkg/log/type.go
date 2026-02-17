package log

import "go.uber.org/zap"

type ZapConfig struct {
	Level        string
	Mode         string
	Encoding     string
	ColorEnabled bool
}

type zapLogger struct {
	sugarLogger *zap.SugaredLogger
	cfg         *ZapConfig
}
