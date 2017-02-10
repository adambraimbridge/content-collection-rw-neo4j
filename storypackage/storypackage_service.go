package storypackage

import (
	"encoding/json"
	"fmt"
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
		"Curation": "uuid"})
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
		Statement: `MATCH (n:Curation {uuid:{uuid}}) 
				RETURN n.uuid as uuid, n.publishReference as publishReference, n.lastModified as lastModified`,
		Parameters: map[string]interface{}{
			"uuid": uuid,
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
	/*	sp := thing.(storyPackage)
		params := map[string]interface{}{
			"uuid": sp.UUID,
		} */

	return nil
}

func addStoryPackageItemQuery(itemUuid string) *neoism.CypherQuery {
	statement := `	`
	query := &neoism.CypherQuery{
		Statement:  statement,
		Parameters: map[string]interface{}{},
	}
}

func extractUUIDFromURI(uri string) (string, error) {
	result := uuidExtractRegex.FindStringSubmatch(uri)
	if len(result) == 2 {
		return result[1], nil
	}

	return "", fmt.Errorf("Couldn't extract uuid from uri %s", uri)
}

// DecodeJSON - Decodes JSON into story package
func (pcd service) DecodeJSON(dec *json.Decoder) (interface{}, string, error) {
	sp := storyPackage{}
	err := dec.Decode(&sp)

	return sp, sp.UUID, err
}

//Delete - Deletes a content
func (pcd service) Delete(uuid string) (bool, error) {
	return true, nil
}

// Count - Returns a count of the number of content in this Neo instance
func (pcd service) Count() (int, error) {
	log.Info("Infof - Count")
	return 34444, nil
}
