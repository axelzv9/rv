package example

import (
	"context"
	"log"
	"testing"

	"github.com/axelzv9/rv"
	"github.com/axelzv9/rv/example/project1/repository"
	"github.com/axelzv9/rv/example/project1/server"
	"github.com/axelzv9/rv/example/project1/service"
)

func TestProject1(t *testing.T) {
	ctx := context.Background()
	err := rv.Revolve(ctx, rv.Options(
		rv.WithLogger(rv.LogFunc(func(lvl rv.LogLevel, format string, args ...any) {
			switch lvl {
			case rv.LogLevelInfo:
				log.Printf("customLogFunc: "+format, args...)
			case rv.LogLevelDebug:
				log.Printf("customLogFunc: debug:"+format, args...)
			}
		})),
		rv.WithDuckTyping(),
		rv.Supply(ctx),
		rv.Provide(
			repository.NewOne,
			repository.NewTwo,
			service.NewOne,
			service.NewTwo,
			server.NewServer,
		),
		rv.Invoke(func(srv *server.Server) error {
			log.Println("it's working...")
			return srv.Serve()
		}),
	))
	if err != nil {
		t.Fatal(err)
	}
}
