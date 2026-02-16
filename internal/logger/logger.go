package logger

import (
	"github.com/pkg/errors"
	"github.com/samber/do/v2"
	"github.com/wickedv43/TAM-backend/internal/config"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Logger struct {
	*zap.SugaredLogger
}

func NewLogger(i do.Injector) (*Logger, error) {
	cfg := zap.Config{
		Level:            zap.NewAtomicLevelAt(zap.DebugLevel),
		Development:      true,
		Encoding:         "console",
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
		EncoderConfig: zapcore.EncoderConfig{
			TimeKey:          "time",
			LevelKey:         "level",
			NameKey:          "logger",
			CallerKey:        "caller",
			MessageKey:       "msg",
			StacktraceKey:    "stacktrace",
			LineEnding:       zapcore.DefaultLineEnding,
			EncodeLevel:      zapcore.CapitalColorLevelEncoder,
			EncodeTime:       zapcore.RFC3339TimeEncoder,
			EncodeDuration:   zapcore.StringDurationEncoder,
			EncodeCaller:     zapcore.ShortCallerEncoder,
			ConsoleSeparator: "\t",
		},
	}

	appCFG := do.MustInvoke[*config.Config](i)
	switch appCFG.AppMode {
	case "dev":
		cfg.Level.SetLevel(zapcore.DebugLevel)
	case "prod":
		cfg.Level.SetLevel(zapcore.InfoLevel)
	default:
		cfg.Level.SetLevel(zapcore.DebugLevel)
	}

	log, err := cfg.Build(zap.AddCaller(), zap.AddStacktrace(zapcore.PanicLevel))
	if err != nil {
		return nil, errors.Wrap(err, "init logger")
	}

	sugar := log.Sugar().Named("ad_market").WithOptions(zap.Hooks())

	return &Logger{sugar}, nil
}
