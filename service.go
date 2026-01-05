package main

import (
	"context"
	"database/sql"
	"log"
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

func (p *program) Start(s service.Service) error {
	if service.Interactive() {
		logger.Info("Running in terminal.")
	} else {
		logger.Info("Running under service manager.")
	}
	p.exit = make(chan struct{})

	// Start should not block. Do the actual work async.
	go p.run()
	return nil
}

func (p *program) run() error {
	logger.Infof("I'm running %v, with version %v.", service.Platform(), Version)

	var mydb *sql.DB
	var err error

	if settings.UseMySQL {
		cfg := mysql.Config{
			User:                 settings.MySqlSettings.Username,
			Passwd:               settings.MySqlSettings.Password,
			Net:                  "tcp",
			Addr:                 settings.MySqlSettings.Host + ":3306",
			DBName:               settings.MySqlSettings.Database,
			AllowNativePasswords: true,
			ParseTime:            true,
		}
		mydb, err = store.NewMySQLStorage(cfg)
		if err != nil {
			log.Fatal(err)
		}
	} else if settings.UseSqlite {
		mydb, err = store.NewSqliteDatabase("test.db")
		if err != nil {
			log.Fatal(err)
		}
	}
	if err := mydb.Ping(); err != nil {
		log.Fatal(err)
	}

	if !settings.Debug && settings.AuthToken == "" {
		log.Println("WARNING: AUTH_TOKEN is not set in non-debug mode. Security is disabled.")
	}

	store := store.NewStorage(mydb)
	handler := handlers.NewHandler(store, settings.UseFileSystem, tpls, nameOfService, Version)

	router := NewRouter(handler, store, settings)
	srv := &http.Server{
		Handler: http.TimeoutHandler(router, time.Duration(settings.Timeout)*time.Second, "Timeout"),
		Addr:    settings.Port,
	}

	go func() {
		log.Println(srv.ListenAndServe())
	}()

	<-p.exit
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	srv.Shutdown(ctx)
	return nil
}

func (p *program) Stop(s service.Service) error {
	// Any work in Stop should be quick, usually a few seconds at most.
	logger.Info("I'm Stopping!")
	close(p.exit)
	return nil
}
