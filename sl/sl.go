package sl

import (
	"log/slog"

	"github.com/google/uuid"
	"github.com/jackc/pgx"
)

func Err(e error) slog.Attr {
	return slog.String("err", e.Error())
}

func Module(module string) slog.Attr {
	return slog.Attr{
		Key:   "module",
		Value: slog.StringValue(module),
	}
}

func PgErr(e pgx.PgError) slog.Attr {
	return slog.Group("pgError",
		slog.String("message", e.Message),
		slog.String("detail", e.Detail),
		slog.String("hint", e.Hint),
		slog.String("code", e.Code),
		slog.String("severity", e.Severity),
	)
}

func Error(err error) slog.Attr {
	return slog.Any("error", err)
}

func Method(method string) slog.Attr {
	return slog.Attr{
		Key:   "method",
		Value: slog.StringValue(method),
	}
}

func Query(q string) slog.Attr {
	return slog.Attr{
		Key:   "query",
		Value: slog.StringValue(q),
	}
}

func Args(args []interface{}) slog.Attr {
	return slog.Any("args", args)
}

func UUID(key string, id uuid.UUID) slog.Attr {
	return slog.Attr{
		Key:   key,
		Value: slog.StringValue(id.String()),
	}
}
