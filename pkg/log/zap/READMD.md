## Set environment variable

```shell
# log file name
LOG_FILENAME = your project name
# file storage directory
LOG_STORAGE_DIRECTORY = /var/log
# file storage policy
LOG_STORAGE_POLICY = debug
# file max size
LOG_MAX_SIZE = 10
# max backups
LOG_MAX_BACKUPS = 5
# file expired collection time
LOG_MAX_AGE = 30
# file compress
LOG_COMPRESS = false
```



# Example

Use in Controller operator

```go
// main.go

import (
	"github.com/nautes-labs/nautes/pkg/log/zap"
)

func main() {
  ...
  ctrl.SetLogger(zap.New())
}
```



Use in Kratos server

```go
// cmd/main.go

import (
	"github.com/nautes-labs/nautes/pkg/log/zap"
)

func main() {
  ...
  logger := log.With(zap.NewLogger(),
		"ts", log.DefaultTimestamp,
		"caller", log.DefaultCaller,
		"service.id", id,
		"service.name", Name,
		"service.version", Version,
		"trace.id", tracing.TraceID(),
		"span.id", tracing.SpanID(),
  )

}
```
