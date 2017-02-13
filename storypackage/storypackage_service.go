package storypackage

import (
	"encoding/json"
	//"fmt"
	//	"github.com/Financial-Times/neo-model-utils-go/mapper"
	"github.com/Financial-Times/neo-utils-go/neoutils"
	log "github.com/Sirupsen/logrus"
	"github.com/jmcvetta/neoism"
	"regexp"
)

var uuidExtractRegex = regexp.MustCompile(".*/([0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12})$")

// CypherDriver - CypherDriver
type service struct {
	conn neoutils.NeoConnection
}

//NewCypherDriver instantiate driver
func NewCypherStoryPackageService(cypherRunner neoutils.NeoConnection) service {
	return service{cypherRunner}
}

//Initialise initialisation of the indexes
func (cd service) Initialise() error {
	err := cd.conn.EnsureIndexes(map[string]string{
		"Identifier": "value",
	})

	if err != nil {
		return err
	}

	return cd.conn.EnsureConstraints(map[string]string{
		"StoryPackage": "uuid"})
}

// Check - Feeds into the Healthcheck and checks whether we can connect to Neo and that the datastore isn't empty
func (pcd service) Check() error {
	return neoutils.Check(pcd.conn)
}

// Read - reads a story package given a UUID
func (pcd service) Read(uuid string) (interface{}, bool, error) {
	log.Infof("Trying to read story package with uuid: %v", uuid)

	results := []struct {
		storyPackage
	}{}

	query := &neoism.CypherQuery{
		Statement: `MATCH (n {uuid:{uuid}}) WHERE {label} IN labels(n)
				RETURN n.uuid as uuid, n.publishReference as publishReference, n.lastModified as lastModified`,
		Parameters: map[string]interface{}{
			"label": "StoryPackage",
			"uuid":  uuid,
		},
		Result: &results,
	}

	err := pcd.conn.CypherBatch([]*neoism.CypherQuery{query})

	if err != nil {
		return storyPackage{}, false, err
	}

	if len(results) == 0 {
		return storyPackage{}, false, nil
	}

	result := results[0]

	storyPackageResult := storyPackage{
		UUID:             result.UUID,
		PublishReference: result.PublishReference,
		LastModified:     result.LastModified,
	}

	return storyPackageResult, true, nil
}

//Write - Writes a story package node
func (pcd service) Write(thing interface{}) error {
	log.Info("Entered write")
	sp := thing.(storyPackage)

	spDeleteRelationshipsQuery := &neoism.CypherQuery{
		Statement: `MATCH (sp:Thing {uuid: {uuid}})
			MATCH (item:Thing)<-[rel:SELECTS]-(sp) 
			DELETE rel`,
		Parameters: map[string]interface{}{
			"uuid": sp.UUID,
		},
	}

	params := map[string]interface{}{
		"uuid":             sp.UUID,
		"publishReference": sp.PublishReference,
		"lastModified":     sp.LastModified,
	}

	writeSPQuery := &neoism.CypherQuery{
		Statement: `MERGE (sp:Thing {uuid: {uuid}})
		    set sp={allprops}
		    set sp :Curation:StoryPackage`,
		Parameters: map[string]interface{}{
			"uuid":     sp.UUID,
			"allprops": params,
		},
	}

	queries := []*neoism.CypherQuery{spDeleteRelationshipsQuery, writeSPQuery}

	for i, item := range sp.Items {
		addItemQuery := addStoryPackageItemQuery(sp.UUID, item.UUID, i+1)
		queries = append(queries, addItemQuery)
	}

	return pcd.conn.CypherBatch(queries)
}

func addStoryPackageItemQuery(storyPackageUuid string, itemUuid string, order int) *neoism.CypherQuery {
	query := &neoism.CypherQuery{
		Statement: `MATCH (storyPackage:StoryPackage {uuid: {spUuid}})
			MERGE (content:Thing {uuid: {contentUuid}})
			MERGE (storyPackage)-[rel:SELECTS {order: {itemOrder}}]->(content)`,
		Parameters: map[string]interface{}{
			"spUuid":      storyPackageUuid,
			"contentUuid": itemUuid,
			"itemOrder":   order,
		},
	}

	return query
}

//Delete - Deletes a story package
func (pcd service) Delete(uuid string) (bool, error) {
	return true, nil
}

// DecodeJSON - Decodes JSON into story package
func (pcd service) DecodeJSON(dec *json.Decoder) (interface{}, string, error) {
	sp := storyPackage{}
	err := dec.Decode(&sp)

	return sp, sp.UUID, err
}

// Count - Returns a count of the number of content in this Neo instance
func (pcd service) Count() (int, error) {
	results := []struct {
		Count int `json:"c"`
	}{}

	query := &neoism.CypherQuery{
		Statement: `MATCH (n) WHERE {label} IN labels(n) RETURN count(n) as c`,
		Parameters: map[string]interface{}{
			"label": "StoryPackage",
		},
		Result: &results,
	}

	err := pcd.conn.CypherBatch([]*neoism.CypherQuery{query})
	if err != nil {
		return 0, err
	}

	return results[0].Count, nil
}
