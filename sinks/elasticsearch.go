package sinks

import (
	"context"
	"fmt"
	"time"

	"github.com/olivere/elastic"
	"github.com/sirupsen/logrus"
)

// ElasticsearchSink represents an Elasticsearch sink
type ElasticsearchSink struct {
	client    *elastic.Client
	ctx       context.Context
	Address   string
	IndexName string
}

// NewElasticsearchSink creates a ElasticsearchSink instance
func NewElasticsearchSink(address, indexName string) Sink {
	ctx := context.Background()

	es := ElasticsearchSink{
		Address:   address,
		ctx:       ctx,
		IndexName: indexName,
	}

	client, err := elastic.NewClient(elastic.SetURL(es.Address))
	if err != nil {
		logrus.Panicf("Error connecting to Elasticsearch (%s): %v", es.Address, err)
		panic(err)
	}

	es.client = client

	info, code, err := es.client.Ping(es.Address).Do(ctx)
	if err != nil {
		logrus.Panicf("Error pinging Elasticsearch: %v", err)
		panic(err)
	}
	logrus.Debugf("Elasticsearch returned with code %d and version %s", code, info.Version.Number)

	mapping := fmt.Sprintf(`{"template":"%s-*"}`, es.IndexName)

	exists, err := client.IndexTemplateExists(es.IndexName).Do(ctx)
	if err != nil {
		logrus.Panicf("Error checking if index template exists: %v", err)
		panic(err)
	}
	if !exists {
		createIndex, err := client.IndexPutTemplate(es.IndexName).BodyString(mapping).Do(ctx)
		if err != nil {
			logrus.Panicf("Error creating index template %s: %v", es.IndexName, err)
			panic(err)
		}
		if !createIndex.Acknowledged {
			logrus.Panicf("Error acknowledging index template %s: %v", es.IndexName, err)
			panic(err)
		}
	}

	return Sink(&es)
}

// Write writes to an Elasticsearch server
func (es *ElasticsearchSink) Write(output []byte) error {
	log, err := es.client.Index().
		Index(es.getCurrentDayIndex()).
		Type("log").
		BodyJson(string(output)).
		Do(es.ctx)
	if err != nil {
		return err
	}

	logrus.Debugf("Indexed log %s to index %s, type %s\n", log.Id, log.Index, log.Type)
	return nil
}

func (es *ElasticsearchSink) getCurrentDayIndex() string {
	t := time.Now()
	return fmt.Sprintf("%s-%s", es.IndexName, t.Format("2006-01-02"))
}
