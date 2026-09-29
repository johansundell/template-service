package main

import (
	"context"
	"crypto/rand"
	"database/sql"
	"log"
	"net"
	"net/http"
	"path/filepath"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/johansundell/template-service/handlers"
	"github.com/johansundell/template-service/router"
	"github.com/johansundell/template-service/store"
	"github.com/johansundell/template-service/utils"
	"github.com/kardianos/service"
)

var logger service.Logger

type program struct {
	exit chan struct{}
}

// injectable constructors so tests can mock DB initialization
var newMySQLStorage = store.NewMySQLStorage
var newSqliteDatabase = store.NewSqliteDatabase

func (p *program) Start(s service.Service) error {
	loadSettings()
	if err := settings.Validate(); err != nil {
		logError("invalid configuration: %v", err)
		return err
	}
	if service.Interactive() {
		logInfo("Running in terminal.")
	} else {
		logInfo("Running under service manager.")
	}
	p.exit = make(chan struct{})

	// Start should not block while the service is serving requests, but it must
	// report initialization failures to the service manager.
	startup := make(chan error, 1)
	go p.run(startup)
	return <-startup
}

func (p *program) run(startup chan<- error) error {
	logInfo("I'm running %v, with version %v.", service.Platform(), Version)

	var mydb *sql.DB
	var err error

	if settings.UseMySQL {
		cfg := mysql.Config{
			User:                 settings.MySqlSettings.Username,
			Passwd:               settings.MySqlSettings.Password,
			Net:                  "tcp",
			Addr:                 settings.MySqlSettings.Host + ":" + settings.MySqlSettings.Port,
			DBName:               settings.MySqlSettings.Database,
			AllowNativePasswords: true,
			ParseTime:            true,
		}
		mydb, err = newMySQLStorage(cfg)
		if err != nil {
			logError("failed to initialize MySQL storage: %v", err)
			startup <- err
			return err
		}
	} else if settings.UseSqlite {
		mydb, err = newSqliteDatabase(filepath.Join(utils.GetBinaryBasePath(), nameOfService+".db"))
		if err != nil {
			logError("failed to initialize sqlite storage: %v", err)
			startup <- err
			return err
		}
	}
	if mydb != nil {
		defer mydb.Close()
	}
	if err := mydb.Ping(); err != nil {
		logError("database ping failed: %v", err)
		startup <- err
		return err
	}
	ensureAuthToken()

	store := store.NewStorage(mydb)
	handler := handlers.NewHandler(store, settings.UseFileSystem, tpls, nameOfService, Version)

	routerEngine, err := router.NewRouter(router.Config{
		Handler:  handler,
		Store:    store,
		Settings: settings,
		Assets:   embededFiles,
		Version:  Version,
		Logger:   serviceLoggerAdapter{logger: logger},
	})
	if err != nil {
		logError("failed to create router: %v", err)
		startup <- err
		return err
	}
	srv := &http.Server{
		Handler: http.TimeoutHandler(routerEngine, time.Duration(settings.Timeout)*time.Second, "Timeout"),
		Addr:    settings.Port,
	}
	listener, err := net.Listen("tcp", settings.Port)
	if err != nil {
		logError("failed to listen on %s: %v", settings.Port, err)
		startup <- err
		return err
	}
	startup <- nil

	go func() {
		if err := srv.Serve(listener); err != nil && err != http.ErrServerClosed {
			logError("HTTP server stopped: %v", err)
		}
	}()

	<-p.exit
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		logError("HTTP server shutdown failed: %v", err)
	}
	return nil
}

func (p *program) Stop(s service.Service) error {
	// Any work in Stop should be quick, usually a few seconds at most.
	logInfo("I'm Stopping!")
	close(p.exit)
	return nil
}

// ensureAuthToken generates a temporary random token when AUTH_TOKEN is not
// configured, so protected routes stay locked down. The token is logged
// because it is the only way to call those routes during this run.
func ensureAuthToken() {
	if settings.AuthToken != "" {
		return
	}
	settings.AuthToken = rand.Text()
	logWarning("AUTH_TOKEN is not set; using temporary token for this run: %s", settings.AuthToken)
}

type serviceLoggerAdapter struct {
	logger service.Logger
}

func (a serviceLoggerAdapter) Infof(format string, v ...interface{}) {
	if a.logger != nil {
		a.logger.Infof(format, v...)
	} else {
		log.Printf(format, v...)
	}
}

func (a serviceLoggerAdapter) Warningf(format string, v ...interface{}) {
	if a.logger != nil {
		a.logger.Warningf(format, v...)
	} else {
		log.Printf("WARNING: "+format, v...)
	}
}

func (a serviceLoggerAdapter) Errorf(format string, v ...interface{}) {
	if a.logger != nil {
		a.logger.Errorf(format, v...)
	} else {
		log.Printf("ERROR: "+format, v...)
	}
}

func logInfo(format string, v ...interface{}) {
	serviceLoggerAdapter{logger: logger}.Infof(format, v...)
}

func logWarning(format string, v ...interface{}) {
	serviceLoggerAdapter{logger: logger}.Warningf(format, v...)
}

func logError(format string, v ...interface{}) {
	serviceLoggerAdapter{logger: logger}.Errorf(format, v...)
}
