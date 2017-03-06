package collection

import (
	"fmt"
	"github.com/Financial-Times/neo-utils-go/neoutils"
	"github.com/jmcvetta/neoism"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

const (
	uuid = "sp-12345"
	collectionType = "StoryPackage"
	relationType = "SELECTS"
)

func TestWrite(t *testing.T) {
	assert := assert.New(t)
	db := getDatabaseConnectionAndCheckClean(t, assert)
	testService := getContentCollectionService(db)
	defer cleanDB(db, assert)

	err := testService.Write(createContentCollection(2), collectionType, relationType)
	assert.NoError(err)

	result, found, err := testService.Read(uuid, collectionType, relationType);
	validateResult(assert, result, found, err, 2)
}

func TestUpdate(t *testing.T) {
	assert := assert.New(t)
	db := getDatabaseConnectionAndCheckClean(t, assert)
	testService := getContentCollectionService(db)
	defer cleanDB(db, assert)

	err := testService.Write(createContentCollection(2), collectionType, relationType)
	assert.NoError(err)

	result, found, err := testService.Read(uuid, collectionType, relationType);
	validateResult(assert, result, found, err, 2)

	err = testService.Write(createContentCollection(3), collectionType, relationType)
	assert.NoError(err)

	result, found, err = testService.Read(uuid, collectionType, relationType);
	validateResult(assert, result, found, err, 3)
}

func TestDelete(t *testing.T) {
	assert := assert.New(t)
	db := getDatabaseConnectionAndCheckClean(t, assert)
	testService := getContentCollectionService(db)
	defer cleanDB(db, assert)

	err := testService.Write(createContentCollection(2), collectionType, relationType)
	assert.NoError(err)

	result, found, err := testService.Read(uuid, collectionType, relationType);
	validateResult(assert, result, found, err, 2)

	deleted, err := testService.Delete(uuid, relationType)
	assert.NoError(err)
	assert.Equal(true, deleted)

	result, found, err = testService.Read(uuid, collectionType, relationType);
	assert.NoError(err)
	assert.False(found)
	assert.Equal(contentCollection{}, result.(contentCollection))
}

func createContentCollection(itemCount int) contentCollection {
	items := []item {}
	for count := 0; count < itemCount; count ++ {
		items = append(items, item { fmt.Sprint("Item", count) } );
	}

	c := contentCollection{
		UUID:             uuid,
		PublishReference: "test12345",
		LastModified:     "2016-08-25T06:06:23.532Z",
		Items:            items,
	}

	return c
}

func validateResult(assert *assert.Assertions, result interface{}, found bool, err error, itemCount int) {
	assert.NoError(err);
	assert.True(found);

	collection := result.(contentCollection)
	assert.Equal(uuid, collection.UUID)
	assert.Equal(itemCount, len(collection.Items))
}

func getDatabaseConnectionAndCheckClean(t *testing.T, assert *assert.Assertions) neoutils.NeoConnection {
	db := getDatabaseConnection(assert)
	cleanDB(db, assert)
	checkDbClean(db, t)
	return db
}

func getDatabaseConnection(assert *assert.Assertions) neoutils.NeoConnection {
	url := os.Getenv("NEO4J_TEST_URL")
	if url == "" {
		url = "http://neo4j:foobar@localhost:7474/db/data"
	}

	conf := neoutils.DefaultConnectionConfig()
	conf.Transactional = false
	db, err := neoutils.Connect(url, conf)
	assert.NoError(err, "Failed to connect to Neo4j")
	return db
}

func cleanDB(db neoutils.CypherRunner, assert *assert.Assertions) {
	qs := []*neoism.CypherQuery{
		{
			Statement: fmt.Sprintf("MATCH (mc:Thing {uuid: '%v'}) DETACH DELETE mc", uuid),
		},
	}

	err := db.CypherBatch(qs)
	assert.NoError(err)
}

func checkDbClean(db neoutils.CypherRunner, t *testing.T) {
	assert := assert.New(t)

	result := []struct {
		Uuid string `json:"uuid"`
	}{}

	checkGraph := neoism.CypherQuery{
		Statement: `MATCH (n:Thing) WHERE n.uuid in {uuids} RETURN n.uuid`,
		Parameters: neoism.Props{
			"uuids": []string{uuid},
		},
		Result: &result,
	}
	err := db.CypherBatch([]*neoism.CypherQuery{&checkGraph})
	assert.NoError(err)
	assert.Empty(result)
}

func getContentCollectionService(db neoutils.NeoConnection) service {
	s := NewContentCollectionService(db)
	s.Initialise()
	return s
}
