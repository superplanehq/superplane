module github.com/superplanehq/superplane/agent2

go 1.23

require (
	github.com/jackc/pgx/v5 v5.7.2
	github.com/sirupsen/logrus v1.9.3
	github.com/superplanehq/superplane v0.0.0
	google.golang.org/grpc v1.72.0
	google.golang.org/protobuf v1.36.6
)

replace github.com/superplanehq/superplane => ../
