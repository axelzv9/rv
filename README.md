# Dependency resolver for Golang

Creates instances, invokes dependent functions

### Key features
- One-function usage, just provide constructors or existing values as options
- Supports your favourite logger (```rv.WithLogger``` via ```rv.LogFunc``` or ```rv.Logger``` interface)
- Supports duck typing (with option ```rv.WithDuckTyping```)
- Supports error detection while calling constructors
- Provides informative error descriptions to find missing faster

### Only 3 key options: 
- ```rv.Supply``` - to pass existing value
- ```rv.Provide``` - to pass constructor (can returns more that one value including error)
- ```rv.Invoke``` - to call a target function, which consumes dependencies and do the work

## Usage example

```go
package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"github.com/axelzv9/rv"
	"github.com/axelzv9/rv/example/project1/repository"
	"github.com/axelzv9/rv/example/project1/server"
	"github.com/axelzv9/rv/example/project1/service"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	defer cancel()
	
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
		log.Fatal(err)
	}
}
```
