package util

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"
	"github.com/go-sql-driver/mysql"
	"github.com/testcontainers/testcontainers-go"
	mysqlContainer "github.com/testcontainers/testcontainers-go/modules/mysql"
	"github.com/testcontainers/testcontainers-go/network"
	"github.com/testcontainers/testcontainers-go/wait"
	mysql2 "gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var (
	dbContainerName = "mysqldb"
	dbPort          = 3306
	dbPortNat       = nat.Port("3306/tcp")
	mysqlImage      = "mysql:8.0"
	flywayImage     = "flyway/flyway:10.17.1"
)

func NewTestDB(ctx context.Context) (*gorm.DB, func()) {
	nw, err := network.New(ctx)
	if err != nil {
		panic(err)
	}

	mysqlC, cleanupFunc, err := createMySQLContainer(ctx, nw)
	if err != nil {
		panic(err)
	}

	if err = execFlywayContainer(ctx, nw.Name); err != nil {
		panic(err)
	}

	db, err := createDBConnection(ctx, mysqlC)
	if err != nil {
		panic(err)
	}

	return db, cleanupFunc
}

func createMySQLContainer(ctx context.Context, nw *testcontainers.DockerNetwork) (testcontainers.Container, func(), error) {
	mysqlC, err := mysqlContainer.Run(ctx, mysqlImage,
		testcontainers.WithHostConfigModifier(func(c *container.HostConfig) {
			c.Tmpfs = map[string]string{"/var/lib/mysql": "rw"}
		}),
		network.WithNetwork([]string{dbContainerName}, nw),
	)
	if err != nil {
		return nil, nil, err
	}

	cleanupFunc := func() {
		if mysqlC.IsRunning() {
			if err = mysqlC.Terminate(ctx); err != nil {
				panic(err)
			}
		}
	}
	return mysqlC, cleanupFunc, nil
}

func execFlywayContainer(ctx context.Context, networkName string) error {
	mysqlDBUrl := fmt.Sprintf("-url=jdbc:mysql://%s:%d/test?allowPublicKeyRetrieval=true", dbContainerName, dbPort)
	flywayC, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image: flywayImage,
			Cmd: []string{
				mysqlDBUrl, "-user=test", "-password=test",
				"baseline", "-baselineVersion=0.0",
				"-locations=filesystem:/flyway", "-validateOnMigrate=false", "migrate"},
			Networks: []string{networkName},
			Files: []testcontainers.ContainerFile{
				{
					HostFilePath:      "../migrations",
					ContainerFilePath: "/flyway/sql",
					FileMode:          644,
				},
			},
			WaitingFor: wait.ForLog("Successfully applied").AsRegexp(),
		},
		Started: true,
	})
	if err != nil {
		return err
	}

	defer func() {
		if flywayC.IsRunning() {
			if err = flywayC.Terminate(ctx); err != nil {
				panic(err)
			}
		}
	}()
	return err
}

func createDBConnection(ctx context.Context, mysqlC testcontainers.Container) (*gorm.DB, error) {
	host, err := mysqlC.Host(ctx)
	if err != nil {
		return nil, err
	}
	port, err := mysqlC.MappedPort(ctx, dbPortNat)
	if err != nil {
		return nil, err
	}
	cfg := mysql.Config{
		DBName:               "test",
		User:                 "test",
		Passwd:               "test",
		Addr:                 fmt.Sprintf("%s:%d", host, port.Int()),
		Net:                  "tcp",
		ParseTime:            true,
		AllowNativePasswords: true,
	}
	db, err := gorm.Open(mysql2.Open(cfg.FormatDSN()))
	if err != nil {
		return nil, err
	}
	db.Logger = logger.Discard
	return db, nil
}
