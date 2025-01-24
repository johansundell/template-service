package main

import (
	"embed"
	"flag"
	"log"

	"github.com/kardianos/service"
)

const (
	appVersionStr = "v0.0.1"
	nameOfService = "template-service"
)

/*var (
	routes = Routes{
		Route{
			"Index",
			"GET",
			"/",
			defaultHandler,
		},
	}
	router *mux.Router
)*/

//go:embed tmpl/*.html
var tpls embed.FS

//go:embed assets/*
var embededFiles embed.FS

func main() {
	svcFlag := flag.String("service", "", "Control the system service.")
	flag.Parse()

	svcConfig := &service.Config{
		Name:        nameOfService,
		DisplayName: nameOfService,
		Description: nameOfService,
	}

	prg := &program{}
	s, err := service.New(prg, svcConfig)
	if err != nil {
		log.Fatal(err)
	}
	errs := make(chan error, 5)
	logger, err = s.Logger(errs)
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		for {
			err := <-errs
			if err != nil {
				log.Print(err)
			}
		}
	}()

	if len(*svcFlag) != 0 {
		err := service.Control(s, *svcFlag)
		if err != nil {
			log.Printf("Valid actions: %q\n", service.ControlAction)
			log.Fatal(err)
		}
		return
	}
	err = s.Run()
	if err != nil {
		logger.Error(err)
	}
}

/*func defaultHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "<html><body>We are up and running "+nameOfService+" on port "+settings.Port+" ;)</body></html>")
}*/
