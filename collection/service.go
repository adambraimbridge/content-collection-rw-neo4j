package collection

import (
	"encoding/json"
	"fmt"
	"github.com/Financial-Times/neo-utils-go/neoutils"
	"github.com/jmcvetta/neoism"
)

type service struct {
	conn           neoutils.NeoConnection
	collectionType string
	relationType   string
}

//instantiate service
func NewContentCollectionService(cypherRunner neoutils.NeoConnection, collectionType string, relationType string) service {
	return service{cypherRunner, collectionType, relationType}
}

//Initialise initialisation of the indexes
func (cd service) Initialise() error {
	return cd.conn.EnsureConstraints(map[string]string{
		cd.collectionType: "uuid",
	})
}

// Check - Feeds into the Healthcheck and checks whether we can connect to Neo and that the datastore isn't empty
func (pcd service) Check() error {
	return neoutils.Check(pcd.conn)
}

// Read - reads a content collection given a UUID
func (pcd service) Read(uuid string) (interface{}, bool, error) {
	results := []struct {
		contentCollection
	}{}

	query := &neoism.CypherQuery{
		Statement: fmt.Sprintf(`MATCH (n:%s {uuid:{uuid}})
				OPTIONAL MATCH (n)-[rel:%s]->(t:Thing)
				WITH n, rel, t
				ORDER BY rel.order
				RETURN n.uuid as uuid, n.publishReference as publishReference, n.lastModified as lastModified, collect({uuid:t.uuid}) as items`, pcd.collectionType, pcd.relationType),
		Parameters: map[string]interface{}{
			"uuid": uuid,
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
	if len(result.Items) == 1 && (result.Items[0].UUID == "") {
		result.Items = []item{}
	}

	contentCollectionResult := contentCollection{
		UUID:             result.UUID,
		PublishReference: result.PublishReference,
		LastModified:     result.LastModified,
		Items:            result.Items,
	}

	return contentCollectionResult, true, nil
}

//Write - Writes a content collection node
func (pcd service) Write(newThing interface{}) error {
	newContentCollection := newThing.(contentCollection)

	deleteRelationshipsQuery := &neoism.CypherQuery{
		Statement: fmt.Sprintf(`MATCH (n:Curation {uuid: {uuid}})
			OPTIONAL MATCH (item:Thing)<-[rel:%s]-(n)
			DELETE rel`, pcd.relationType),
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
		    set n :Curation:` + pcd.collectionType,
		Parameters: map[string]interface{}{
			"uuid":     newContentCollection.UUID,
			"allprops": params,
		},
	}

	queries := []*neoism.CypherQuery{deleteRelationshipsQuery, writeContentCollectionQuery}

	for i, item := range newContentCollection.Items {
		addItemQuery := addCollectionItemQuery(pcd.collectionType, pcd.relationType, newContentCollection.UUID, item.UUID, i+1)
		queries = append(queries, addItemQuery)
	}

	return pcd.conn.CypherBatch(queries)
}

func addCollectionItemQuery(contentCollectionType string, relationType string, contentCollectionUuid string, itemUuid string, order int) *neoism.CypherQuery {
	query := &neoism.CypherQuery{
		Statement: fmt.Sprintf(`MATCH (n:%s {uuid:{contentCollectionUuid}})
			MERGE (content:Thing {uuid: {contentUuid}})
			MERGE (n)-[rel:%s {order: {itemOrder}}]->(content)`, contentCollectionType, relationType),
		Parameters: map[string]interface{}{
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
		Statement: fmt.Sprintf(`MATCH (n:Thing {uuid: {uuid}})
			OPTIONAL MATCH (item:Thing)<-[rel:%s]-(n)
			DELETE rel`, pcd.relationType),
		Parameters: map[string]interface{}{
			"uuid": uuid,
		},
	}

	removeNode := &neoism.CypherQuery{
		Statement: `MATCH (n:Thing {uuid: {uuid}}) DELETE n`,
		Parameters: map[string]interface{}{
			"uuid": uuid,
		},
		IncludeStats: true,
	}

	err := pcd.conn.CypherBatch([]*neoism.CypherQuery{removeRelationships, removeNode})

	s1, err := removeNode.Stats()
	if err != nil {
		return false, err
	}

	var deleted bool
	if s1.NodesDeleted > 0 {
		deleted = true
	}

	return deleted, err
}

// DecodeJSON - Decodes JSON into a content collection
func (pcd service) DecodeJSON(dec *json.Decoder) (interface{}, string, error) {
	c := contentCollection{}
	err := dec.Decode(&c)

	return c, c.UUID, err
}

// Count - Returns a count of the number of content in this Neo instance
func (pcd service) Count() (int, error) {
	results := []struct {
		Count int `json:"c"`
	}{}

	query := &neoism.CypherQuery{
		Statement: fmt.Sprintf(`MATCH (n:%s) RETURN count(n) as c`, pcd.collectionType),
		Result:    &results,
	}

	err := pcd.conn.CypherBatch([]*neoism.CypherQuery{query})
	if err != nil {
		return 0, err
	}

	return results[0].Count, nil
}
