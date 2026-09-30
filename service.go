package main

import (
	"context"
	"crypto/rand"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/johansundell/template-service/handlers"
	"github.com/johansundell/template-service/logqueue"
	"github.com/johansundell/template-service/router"
	"github.com/johansundell/template-service/store"
	"github.com/johansundell/template-service/types"
	"github.com/kardianos/service"
)

var logger service.Logger

type program struct {
	exit     chan struct{}
	done     chan struct{} // closed when run returns; runErr holds its result
	runErr   error
	stopOnce sync.Once
	// failed receives the error when the HTTP server stops on its own after a
	// successful start, so main can exit and let the service manager restart us.
	failed chan error
}

func newProgram() *program {
	return &program{failed: make(chan error, 1)}
}

// injectable constructors so tests can mock storage initialization and listening
var newSQLiteStore = func(path string) (store.Store, error) {
	s, err := store.NewSQLite(path)
	if err != nil {
		return nil, err
	}
	return s, nil
}
var newMySQLStore = func(cfg mysql.Config) (store.Store, error) {
	s, err := store.NewMySQL(cfg)
	if err != nil {
		return nil, err
	}
	return s, nil
}
var newFileMakerStore = func(ctx context.Context, cfg store.FileMakerConfig) (store.Store, error) {
	s, err := store.NewFileMaker(ctx, cfg)
	if err != nil {
		return nil, err
	}
	return s, nil
}
var netListen = net.Listen

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
	return p.startWorker()
}

// startWorker runs the service worker in the background. Start should not
// block while the service is serving requests, but it must report
// initialization failures to the service manager.
func (p *program) startWorker() error {
	p.exit = make(chan struct{})
	p.done = make(chan struct{})
	startup := make(chan error, 1)
	go func() {
		p.runErr = p.run(startup)
		close(p.done)
	}()
	return <-startup
}

func (p *program) run(startup chan<- error) error {
	logInfo("I'm running %v, with version %v.", service.Platform(), Version)

	st, err := openStore()
	if err != nil {
		logError("failed to initialize %s storage: %v", settings.Storage, err)
		startup <- err
		return err
	}
	defer st.Close()

	pingCtx, cancelPing := context.WithTimeout(context.Background(), 5*time.Second)
	err = st.Ping(pingCtx)
	cancelPing()
	if err != nil {
		logError("database ping failed: %v", err)
		startup <- err
		return err
	}
	ensureAuthToken()

	// Request logs are written in the background. The deferred Close runs after
	// the HTTP server has shut down, drains the queue for up to 5 more seconds
	// and runs before the store is closed.
	logQueue := logqueue.New(st, appLogger())
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		logQueue.Close(ctx)
	}()

	handler, err := handlers.NewHandler(st, settings.UseFileSystem, tpls, nameOfService, Version)
	if err != nil {
		logError("failed to create handlers: %v", err)
		startup <- err
		return err
	}

	routerEngine, err := router.NewRouter(router.Config{
		Handler:  handler,
		LogSink:  logQueue,
		Settings: settings,
		Assets:   embededFiles,
		Version:  Version,
		Logger:   appLogger(),
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
	listener, err := netListen("tcp", settings.Port)
	if err != nil {
		logError("failed to listen on %s: %v", settings.Port, err)
		startup <- err
		return err
	}
	serveErr := make(chan error, 1)
	go func() { serveErr <- srv.Serve(listener) }()
	startup <- nil

	select {
	case err := <-serveErr:
		logError("HTTP server stopped: %v", err)
		select {
		case p.failed <- err:
		default:
		}
		return err
	case <-p.exit:
	}

	// The shutdown deadline is shorter than TIMEOUT (default 15s), so a slow
	// request can still be cut off when the service stops.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		logError("HTTP server shutdown failed: %v", err)
		return err
	}
	return nil
}

// openStore opens the storage backend chosen with STORAGE.
func openStore() (store.Store, error) {
	switch settings.Storage {
	case types.StorageSQLite:
		return newSQLiteStore(settings.SqlitePath)
	case types.StorageMySQL:
		return newMySQLStore(mysql.Config{
			User:                 settings.MySqlSettings.Username,
			Passwd:               settings.MySqlSettings.Password,
			Net:                  "tcp",
			Addr:                 settings.MySqlSettings.Host + ":" + settings.MySqlSettings.Port,
			DBName:               settings.MySqlSettings.Database,
			AllowNativePasswords: true,
		})
	case types.StorageFileMaker:
		fm := settings.FileMaker
		if fm.InsecureSkipVerify {
			logWarning("FMS_INSECURE_SKIP_VERIFY is set: the FileMaker Server certificate is NOT verified, so credentials can be intercepted. Use FMS_CA_FILE instead.")
		}
		ctx, cancel := context.WithTimeout(context.Background(), fm.Timeout)
		defer cancel()
		return newFileMakerStore(ctx, store.FileMakerConfig{
			Host:               fm.Host,
			Database:           fm.Database,
			Username:           fm.Username,
			Password:           fm.Password,
			Timeout:            fm.Timeout,
			Table:              fm.LogTable,
			CAFile:             fm.CAFile,
			InsecureSkipVerify: fm.InsecureSkipVerify,
		})
	default:
		return nil, fmt.Errorf("unsupported STORAGE %q", settings.Storage)
	}
}

// Stop waits for graceful shutdown to finish: once it returns,
// kardianos/service returns from Run and the process exits.
func (p *program) Stop(s service.Service) error {
	logInfo("I'm Stopping!")
	if p.done == nil {
		return nil
	}
	p.stopOnce.Do(func() { close(p.exit) })
	<-p.done
	return p.runErr
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

// serviceLogger adapts the kardianos/service logger, whose methods return an
// error, to router.Logger.
type serviceLogger struct {
	l service.Logger
}

func (s serviceLogger) Infof(format string, v ...interface{})    { s.l.Infof(format, v...) }
func (s serviceLogger) Warningf(format string, v ...interface{}) { s.l.Warningf(format, v...) }
func (s serviceLogger) Errorf(format string, v ...interface{})   { s.l.Errorf(format, v...) }

// appLogger returns the service manager's logger, or the standard logger
// before one is set up (tests, early startup).
func appLogger() router.Logger {
	if logger == nil {
		return router.StdLogger{}
	}
	return serviceLogger{l: logger}
}

func logInfo(format string, v ...interface{}) {
	appLogger().Infof(format, v...)
}

func logWarning(format string, v ...interface{}) {
	appLogger().Warningf(format, v...)
}

func logError(format string, v ...interface{}) {
	appLogger().Errorf(format, v...)
}
