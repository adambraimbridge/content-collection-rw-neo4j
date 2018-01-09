package main

import (
	_ "net/http/pprof"
	"os"

	"github.com/Financial-Times/base-ft-rw-app-go/baseftrwapp"
	"github.com/Financial-Times/content-collection-rw-neo4j/collection"
	"github.com/Financial-Times/go-fthealth/v1_1"
	"github.com/Financial-Times/neo-utils-go/neoutils"
	log "github.com/Sirupsen/logrus"
	"github.com/jawher/mow.cli"
	"time"
)

var appDescription = "A RESTful API for managing Content Collections in neo4j"

func main() {
	app := cli.App("content-collection-rw-neo4j", "A RESTful API for managing Content Collections in neo4j")

	appName := app.String(cli.StringOpt{
		Name:   "app-name",
		Value:  "content-collection-rw-neo4j",
		Desc:   "Name of the application",
		EnvVar: "APP_NAME",
	})

	appSystemCode := app.String(cli.StringOpt{
		Name:   "app-system-code",
		Value:  "upp-content-collection-rw-neo4j",
		Desc:   "System Code of the application",
		EnvVar: "APP_SYSTEM_CODE",
	})

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

		spServiceUrl := "content-collection/story-package"
		cpServiceUrl := "content-collection/content-package"
		services := map[string]baseftrwapp.Service{
			spServiceUrl: collection.NewContentCollectionService(db, []string{"Curation", "StoryPackage"}, "SELECTS", "IS_CURATED_FOR"),
			cpServiceUrl: collection.NewContentCollectionService(db, []string{}, "CONTAINS", ""),
		}

		for _, service := range services {
			service.Initialise()
		}

		baseftrwapp.OutputMetricsIfRequired(*graphiteTCPAddress, *graphitePrefix, *logMetrics)

		checks := []v1_1.Check{checkNeo4J(services[spServiceUrl], spServiceUrl), checkNeo4J(services[cpServiceUrl], cpServiceUrl)}
		hc := v1_1.TimedHealthCheck{
			HealthCheck: v1_1.HealthCheck{
				SystemCode:  *appSystemCode,
				Name:        *appName,
				Description: appDescription,
				Checks:      checks,
			},
			Timeout: 10 * time.Second,
		}
		baseftrwapp.RunServerWithConf(baseftrwapp.RWConf{
			Services:      services,
			HealthHandler: v1_1.Handler(&hc),
			Port:          *port,
			ServiceName:   *appName,
			Env:           "local",
			EnableReqLog:  true,
		})
	}

	log.SetLevel(log.InfoLevel)
	log.Infof("Application started with args %s", os.Args)
	app.Run(os.Args)
}

func checkNeo4J(service baseftrwapp.Service, serviceUrl string) v1_1.Check {
	return v1_1.Check{
		BusinessImpact:   "Cannot read/write content via this writer",
		Name:             "Check connectivity to Neo4j",
		PanicGuide:       "https://dewey.ft.com/upp-content-collection-rw-neo4j.html",
		Severity:         1,
		TechnicalSummary: "Service mapped on URL " + serviceUrl + " cannot connect to Neo4j",
		Checker:          func() (string, error) { return "", service.Check() },
	}
}
