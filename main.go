package main

import (
	"fmt"
	_ "net/http/pprof"
	"os"

	"github.com/Financial-Times/base-ft-rw-app-go/baseftrwapp"
	"github.com/Financial-Times/content-collection-rw-neo4j/collection"
	"github.com/Financial-Times/go-fthealth/v1a"
	"github.com/Financial-Times/neo-utils-go/neoutils"
	log "github.com/Sirupsen/logrus"
	"github.com/jawher/mow.cli"
)

func main() {
	app := cli.App("content-collection-rw-neo4j", "A RESTful API for managing Content Collections in neo4j")

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

	app.Action = func() {
		conf := neoutils.DefaultConnectionConfig()
		conf.BatchSize = *batchSize
		db, err := neoutils.Connect(*neoURL, conf)
		if err != nil {
			log.Errorf("Could not connect to neo4j, error=[%s]\n", err)
		}

		services := map[string]baseftrwapp.Service{
			"content-collection/story-package":   collection.NewContentCollectionService(db, "StoryPackage", "SELECTS"),
			"content-collection/content-package": collection.NewContentCollectionService(db, "ContentPackage", "CONTAINS"),
		}

		var checks []v1a.Check
		for _, service := range services {
			service.Initialise()
			checks = append(checks, makeCheck(service))
		}

		baseftrwapp.OutputMetricsIfRequired(*graphiteTCPAddress, *graphitePrefix, *logMetrics)

		baseftrwapp.RunServerWithConf(baseftrwapp.RWConf{
			Services:      services,
			HealthHandler: v1a.Handler("ft-content-collection_rw_neo4j ServiceModule", "Writes content collections to Neo4j, usually as part of a bulk upload done on a schedule", checks...),
			Port:          *port,
			ServiceName:   "content-collection-rw-neo4j",
			Env:           "local",
			EnableReqLog:  true,
		})
	}

	log.SetLevel(log.InfoLevel)
	log.Infof("Application started with args %s", os.Args)
	app.Run(os.Args)
}

func makeCheck(service baseftrwapp.Service) v1a.Check {
	return v1a.Check{
		BusinessImpact:   "Cannot read/write content via this writer",
		Name:             "Check connectivity to Neo4j",
		PanicGuide:       "https://dewey.ft.com/upp-content-collection-rw-neo4j.html",
		Severity:         1,
		TechnicalSummary: fmt.Sprintf("Service %s cannot connect to Neo4j", service),
		Checker:          func() (string, error) { return "", service.Check() },
	}
}
