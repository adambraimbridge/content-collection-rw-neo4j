package main

import (
	"fmt"
	"github.com/Financial-Times/base-ft-rw-app-go/baseftrwapp"
	"github.com/Financial-Times/content-collection-rw-neo4j/collection"
	"github.com/Financial-Times/go-fthealth/v1a"
	"github.com/Financial-Times/http-handlers-go/httphandlers"
	"github.com/Financial-Times/neo-utils-go/neoutils"
	"github.com/Financial-Times/service-status-go/gtg"
	status "github.com/Financial-Times/service-status-go/httphandlers"
	log "github.com/Sirupsen/logrus"
	"github.com/gorilla/mux"
	"github.com/jawher/mow.cli"
	"github.com/rcrowley/go-metrics"
	"net/http"
	_ "net/http/pprof"
	"os"
)

func main() {
	app := cli.App("content-collection-rw-neo4j", "A RESTful API for managing Story Packages in neo4j")

	neoURL := app.String(cli.StringOpt{
		Name:   "neo-url",
		Value:  "http://localhost:7474/db/data",
		Desc:   "neo4j endpoint URL",
		EnvVar: "NEO_URL",
	})

	graphiteTCPAddress := app.String(cli.StringOpt{
		Name:   "graphiteTCPAddress",
		Value:  "",
		Desc:   "Graphite TCP address, e.g. graphite.ft.com:2003. Leave as default if you do NOT want to output to graphite (e.g. if running locally",
		EnvVar: "GRAPHITE_ADDRESS",
	})

	graphitePrefix := app.String(cli.StringOpt{
		Name:   "graphitePrefix",
		Value:  "",
		Desc:   "Prefix to use. Should start with content, include the environment, and the host name. e.g. coco.pre-prod.brands-rw-neo4j.1 or content.test.brands.rw.neo4j.ftaps58938-law1a-eu-t",
		EnvVar: "GRAPHITE_PREFIX",
	})

	port := app.Int(cli.IntOpt{
		Name:   "port",
		Value:  8080,
		Desc:   "Port to listen on",
		EnvVar: "APP_PORT",
	})

	batchSize := app.Int(cli.IntOpt{
		Name:   "batchSize",
		Value:  1024,
		Desc:   "Maximum number of statements to execute per batch",
		EnvVar: "BATCH_SIZE",
	})

	logMetrics := app.Bool(cli.BoolOpt{
		Name:   "logMetrics",
		Value:  false,
		Desc:   "Whether to log metrics. Set to true if running locally and you want metrics output",
		EnvVar: "LOG_METRICS",
	})

	env := app.String(cli.StringOpt{
		Name:  "env",
		Value: "local",
		Desc:  "environment this app is running in",
	})

	app.Action = func() {
		conf := neoutils.DefaultConnectionConfig()
		conf.BatchSize = *batchSize
		db, err := neoutils.Connect(*neoURL, conf)

		if err != nil {
			log.Errorf("Could not connect to neo4j, error=[%s]\n", err)
		}

		baseftrwapp.OutputMetricsIfRequired(*graphiteTCPAddress, *graphitePrefix, *logMetrics)

		if *env != "local" {
			f, err := os.OpenFile("/var/log/apps/content-collection-rw-neo4j-go-app.log", os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
			if err == nil {
				log.SetOutput(f)
				log.SetFormatter(&log.TextFormatter{DisableColors: true})
			} else {
				log.Fatalf("Failed to initialise log file, %v", err)
			}
			defer f.Close()
		}

		var m http.Handler
		m = router(db)

		m = httphandlers.HTTPMetricsHandler(metrics.DefaultRegistry, m)

		http.Handle("/", m)

		log.Printf("listening on %d", *port)
		log.Println(http.ListenAndServe(fmt.Sprintf(":%d", *port), nil).Error())
		log.Println("exiting on content-collection-rw-neo4j")

	}

	log.SetLevel(log.InfoLevel)
	log.Infof("Application started with args %s", os.Args)
	app.Run(os.Args)
}

//Router sets up the Router - extracted for testability
func router(neoConnection neoutils.NeoConnection) *mux.Router {
	healthHandler := v1a.Handler("ft-content-collection_rw_neo4j ServiceModule", "Writes 'content' to Neo4j, usually as part of a bulk upload done on a schedule", makeCheck(neoConnection))

	m := mux.NewRouter()

	gtgChecker := make([]gtg.StatusChecker, 0)

	storyHandler := collection.NewNeoHttpHandler(neoConnection, "StoryPackage", "SELECTS")
	m.HandleFunc("/content-collection/story-package/__count", storyHandler.CountHandler).Methods("GET")
	m.HandleFunc("/content-collection/story-package/{uuid}", storyHandler.GetHandler).Methods("GET")
	m.HandleFunc("/content-collection/story-package/{uuid}", storyHandler.PutHandler).Methods("PUT")
	m.HandleFunc("/content-collection/story-package/{uuid}", storyHandler.DeleteHandler).Methods("DELETE")

	contentHandler := collection.NewNeoHttpHandler(neoConnection, "ContentPackage", "CONTAINS")
	m.HandleFunc("/content-collection/content-package/__count", contentHandler.CountHandler).Methods("GET")
	m.HandleFunc("/content-collection/content-package/{uuid}", contentHandler.GetHandler).Methods("GET")
	m.HandleFunc("/content-collection/content-package/{uuid}", contentHandler.PutHandler).Methods("PUT")
	m.HandleFunc("/content-collection/content-package/{uuid}", contentHandler.DeleteHandler).Methods("DELETE")

	m.HandleFunc("/__health", healthHandler)
	// The top one of these feels more correct, but the lower one matches what we have in Dropwizard,
	// so it's what apps expect currently
	m.HandleFunc(status.PingPath, status.PingHandler)
	m.HandleFunc(status.PingPathDW, status.PingHandler)

	// The top one of these feels more correct, but the lower one matches what we have in Dropwizard,
	// so it's what apps expect currently same as ping, the content of build-info needs more definition
	m.HandleFunc(status.BuildInfoPath, status.BuildInfoHandler)
	m.HandleFunc(status.BuildInfoPathDW, status.BuildInfoHandler)

	m.HandleFunc(status.GTGPath, status.NewGoodToGoHandler(gtg.FailFastParallelCheck(gtgChecker)))
	return m
}

func makeCheck(cr neoutils.CypherRunner) v1a.Check {
	return v1a.Check{
		BusinessImpact:   "Cannot read/write content via this writer",
		Name:             "Check connectivity to Neo4j - neoUrl is a parameter in hieradata for this service",
		PanicGuide:       "TODO - write panic guide",
		Severity:         1,
		TechnicalSummary: fmt.Sprintf("Cannot connect to Neo4j instance %s with something written to it", cr),
		Checker:          func() (string, error) { return "", neoutils.Check(cr) },
	}
}
