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

/*type Service interface {
	Write(thing interface{}, collectionType string) error
	Read(uuid string, collectionType string) (thing interface{}, found bool, err error)
	Delete(uuid string) (found bool, err error)
	DecodeJSON(*json.Decoder) (thing interface{}, identity string, err error)
	Count(collectionType string) (int, error)
	Check() error
	Initialise() error
}*/

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

// Read - reads a content collection given a UUID
func (pcd service) Read(uuid string /*, collectionType string*/) (interface{}, bool, error) {
	collectionType := "StoryPackage"
	log.Infof("read SP with uuid: %v", uuid)
	results := []struct {
		contentCollection
	}{}

	query := &neoism.CypherQuery{
		Statement: `MATCH (n {uuid:{uuid}}) WHERE {label} IN labels(n)
				OPTIONAL MATCH (n)-[rel:SELECTS]->(t:Thing)
				WITH n, collect({uuid:t.uuid}) as items, rel
				RETURN n.uuid as uuid, n.publishReference as publishReference, n.lastModified as lastModified, items
				ORDER BY rel.order`,
		Parameters: map[string]interface{}{
			"label": collectionType,
			"uuid":  uuid,
		},
		Result: &results,
	}

	err := pcd.conn.CypherBatch([]*neoism.CypherQuery{query})

	if err != nil {
		return contentCollection{}, false, err
	}

	if len(results) == 0 {
		return contentCollection{}, false, nil
	}

	result := results[0]

	contentCollectionResult := contentCollection{
		UUID:             result.UUID,
		PublishReference: result.PublishReference,
		LastModified:     result.LastModified,
		Items:            result.Items,
	}

	return contentCollectionResult, true, nil
}

//Write - Writes a content collection node
func (pcd service) Write( thing interface{}, /*collectionType string, */) error {
	collectionType := "StoryPackage"
	newContentCollection := thing.(contentCollection)

	deleteRelationshipsQuery := &neoism.CypherQuery{
		Statement: `MATCH (n:Thing {uuid: {uuid}})
			MATCH (item:Thing)<-[rel:SELECTS]-(n) 
			DELETE rel`,
		Parameters: map[string]interface{}{
			"uuid": newContentCollection.UUID,
		},
	}

	params := map[string]interface{}{
		"uuid":             newContentCollection.UUID,
		"publishReference": newContentCollection.PublishReference,
		"lastModified":     newContentCollection.LastModified,
	}

	writeContentCollectionQuery := &neoism.CypherQuery{
		Statement: `MERGE (n:Thing {uuid: {uuid}})
		    set n={allprops}
		    set n :Curation:` + collectionType,
		Parameters: map[string]interface{}{
			"uuid":     newContentCollection.UUID,
			"allprops": params,
		},
	}

	queries := []*neoism.CypherQuery{deleteRelationshipsQuery, writeContentCollectionQuery}

	for i, item := range newContentCollection.Items {
		addItemQuery := addStoryPackageItemQuery(collectionType, newContentCollection.UUID, item.UUID, i+1)
		queries = append(queries, addItemQuery)
	}

	return pcd.conn.CypherBatch(queries)
}

func addStoryPackageItemQuery(contentCollectionType string, contentCollectionUuid string, itemUuid string, order int) *neoism.CypherQuery {
	query := &neoism.CypherQuery{
		Statement: `MATCH (n {uuid:{contentCollectionUuid}}) WHERE {label} IN labels(n)
			MERGE (content:Thing {uuid: {contentUuid}})
			MERGE (n)-[rel:SELECTS {order: {itemOrder}}]->(content)`,
		Parameters: map[string]interface{}{
			"label":                 contentCollectionType,
			"contentCollectionUuid": contentCollectionUuid,
			"contentUuid":           itemUuid,
			"itemOrder":             order,
		},
	}

	return query
}

//Delete - Deletes a content collection
func (pcd service) Delete(uuid string) (bool, error) {
	removeRelationships := &neoism.CypherQuery{
		Statement: `MATCH (n:Thing {uuid: {uuid}})
			OPTIONAL MATCH (item:Thing)<-[rel:SELECTS]-(n)
			DELETE rel`,
		Parameters: map[string]interface{}{
			"uuid": uuid,
		},
		IncludeStats: true,
	}

	removeNode := &neoism.CypherQuery{
		Statement: `MATCH (n:Thing {uuid: {uuid}}) DELETE n`,
		Parameters: map[string]interface{}{
			"uuid": uuid,
		},
	}

	err := pcd.conn.CypherBatch([]*neoism.CypherQuery{removeRelationships, removeNode})

	s1, err := removeRelationships.Stats()
	if err != nil {
		return false, err
	}

	var deleted bool
	if s1.ContainsUpdates && s1.LabelsRemoved > 0 {
		deleted = true
	}

	return deleted, err
}

// DecodeJSON - Decodes JSON into story package
func (pcd service) DecodeJSON(dec *json.Decoder) (interface{}, string, error) {
	c := contentCollection{}
	err := dec.Decode(&c)

	return c, c.UUID, err
}

// Count - Returns a count of the number of content in this Neo instance
func (pcd service) Count( /*collectionType string*/ ) (int, error) {
	collectionType := "StoryPackage"
	results := []struct {
		Count int `json:"c"`
	}{}

	query := &neoism.CypherQuery{
		Statement: `MATCH (n) WHERE {label} IN labels(n) RETURN count(n) as c`,
		Parameters: map[string]interface{}{
			"label": collectionType,
		},
		Result: &results,
	}

	err := pcd.conn.CypherBatch([]*neoism.CypherQuery{query})
	if err != nil {
		return 0, err
	}

	return results[0].Count, nil
}
