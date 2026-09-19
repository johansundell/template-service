package main

import (
	"context"
	"database/sql"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/johansundell/template-service/handlers"
	"github.com/johansundell/template-service/store"
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
		if logger != nil {
			logger.Errorf("invalid configuration: %v", err)
		}
		return err
	}
	if service.Interactive() {
		if logger != nil {
			logger.Info("Running in terminal.")
		} else {
			log.Printf("Running in terminal.")
		}
	} else {
		if logger != nil {
			logger.Info("Running under service manager.")
		} else {
			log.Printf("Running under service manager.")
		}
	}
	p.exit = make(chan struct{})

	// Start should not block while the service is serving requests, but it must
	// report initialization failures to the service manager.
	startup := make(chan error, 1)
	go p.run(startup)
	return <-startup
}

func (p *program) run(startup chan<- error) error {
	if logger != nil {
		logger.Infof("I'm running %v, with version %v.", service.Platform(), Version)
	} else {
		log.Printf("I'm running %v, with version %v.", service.Platform(), Version)
	}

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
			if logger != nil {
				logger.Errorf("failed to initialize MySQL storage: %v", err)
			} else {
				log.Printf("failed to initialize MySQL storage: %v", err)
			}
			startup <- err
			return err
		}
	} else if settings.UseSqlite {
		mydb, err = newSqliteDatabase("test.db")
		if err != nil {
			if logger != nil {
				logger.Errorf("failed to initialize sqlite storage: %v", err)
			} else {
				log.Printf("failed to initialize sqlite storage: %v", err)
			}
			startup <- err
			return err
		}
	}
	if mydb != nil {
		defer mydb.Close()
	}
	if err := mydb.Ping(); err != nil {
		if logger != nil {
			logger.Errorf("database ping failed: %v", err)
		} else {
			log.Printf("database ping failed: %v", err)
		}
		startup <- err
		return err
	}
	if settings.AuthToken == "" {
		if logger != nil {
			logger.Warning("AUTH_TOKEN is not set; authentication is disabled.")
		} else {
			log.Printf("AUTH_TOKEN is not set; authentication is disabled.")
		}
	}

	store := store.NewStorage(mydb)
	handler := handlers.NewHandler(store, settings.UseFileSystem, tpls, nameOfService, Version)

	router, err := NewRouter(handler, store, settings)
	if err != nil {
		if logger != nil {
			logger.Errorf("failed to create router: %v", err)
		} else {
			log.Printf("failed to create router: %v", err)
		}
		startup <- err
		return err
	}
	srv := &http.Server{
		Handler: http.TimeoutHandler(router, time.Duration(settings.Timeout)*time.Second, "Timeout"),
		Addr:    settings.Port,
	}
	listener, err := net.Listen("tcp", settings.Port)
	if err != nil {
		if logger != nil {
			logger.Errorf("failed to listen on %s: %v", settings.Port, err)
		} else {
			log.Printf("failed to listen on %s: %v", settings.Port, err)
		}
		startup <- err
		return err
	}
	startup <- nil

	go func() {
		if err := srv.Serve(listener); err != nil && err != http.ErrServerClosed {
			log.Printf("HTTP server stopped: %v", err)
		}
	}()

	<-p.exit
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	srv.Shutdown(ctx)
	return nil
}

func (p *program) Stop(s service.Service) error {
	// Any work in Stop should be quick, usually a few seconds at most.
	if logger != nil {
		logger.Info("I'm Stopping!")
	} else {
		log.Printf("I'm Stopping!")
	}
	close(p.exit)
	return nil
}
