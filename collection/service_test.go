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
	storyPackageCollectionType = "StoryPackage"
	storyPackageUuid           = "sp-12345"
)

type curationResult struct{ curation contentCollection }
type curationItem struct{ cItem item }

func TestRead404NotFound(t *testing.T) {
	assert := assert.New(t)
	db := getDatabaseConnectionAndCheckClean(t, assert)
	testService := getContentCollectionService(db)
	defer cleanDB(db, assert)

	result, found, err := testService.Read(storyPackageUuid, storyPackageCollectionType)
	foundContentCollection := result.(contentCollection)

	assert.NoError(err)
	assert.Equal("", foundContentCollection.UUID, "Result shouldn't exist in DB")
	assert.Equal(false, found, "Resuld should not be found in DB")
}

func TestWriteSuccessfully(t *testing.T) {
	assert := assert.New(t)
	db := getDatabaseConnectionAndCheckClean(t, assert)
	testService := getContentCollectionService(db)
	defer cleanDB(db, assert)

	contentCollectionReceived := createStoryPackageWithItems()

	err := testService.Write(contentCollectionReceived, storyPackageCollectionType)
	assert.NoError(err)

	result, err1 := getCurationByUuid(storyPackageUuid, testService)
	itemResult, err2 := getCurationItemsById(storyPackageUuid, testService)

	assert.NoError(err1)
	assert.NoError(err2)
	assert.Equal(1, len(result), "Result should have size=2")
	assert.Equal(2, len(itemResult), "Items should have size=2")
}

func TestUpdateItemsForStoryPackage(t *testing.T) {
	assert := assert.New(t)
	db := getDatabaseConnectionAndCheckClean(t, assert)
	testService := getContentCollectionService(db)
	defer cleanDB(db, assert)

	contentCollectionReceived := createStoryPackageWithItems()

	err := testService.Write(contentCollectionReceived, storyPackageCollectionType)
	assert.NoError(err)

	contentCollectionReceived.Items = append(contentCollectionReceived.Items, item{UUID: "item3"})
	updateErr := testService.Write(contentCollectionReceived, storyPackageCollectionType)
	result, err1 := getCurationByUuid(storyPackageUuid, testService)
	itemResult, err2 := getCurationItemsById(storyPackageUuid, testService)

	assert.NoError(err1)
	assert.NoError(err2)
	assert.NoError(updateErr)
	assert.Equal(1, len(result), "Result should have size=1")
	assert.Equal(3, len(itemResult), "Items should have size=3")
}

func TestDeleteStoryPackageWithItems(t *testing.T) {
	assert := assert.New(t)
	db := getDatabaseConnectionAndCheckClean(t, assert)
	testService := getContentCollectionService(db)
	defer cleanDB(db, assert)

	contentCollectionReceived := createStoryPackageWithItems()
	err := testService.Write(contentCollectionReceived, storyPackageCollectionType)
	assert.NoError(err)

	deleted, err := testService.Delete(contentCollectionReceived.UUID)

	assert.NoError(err)
	assert.Equal(true, deleted)
}

func createStoryPackageWithItems() contentCollection {
	item1 := item{UUID: "item1"}
	item2 := item{UUID: "item2"}

	c := contentCollection{
		UUID:             storyPackageUuid,
		PublishReference: "test12345",
		LastModified:     "2016-08-25T06:06:23.532Z",
		Items:            []item{item1, item2},
	}

	return c
}

func getCurationByUuid(uuid string, s service) ([]curationResult, error) {
	result := []curationResult{}
	query := &neoism.CypherQuery{
		Statement: `MATCH (n:Curation {uuid:{uuid}}) RETURN n`,
		Parameters: map[string]interface{}{
			"uuid": uuid,
		},
		Result: &result,
	}

	err := s.conn.CypherBatch([]*neoism.CypherQuery{query})

	return result, err
}

func getCurationItemsById(uuid string, s service) ([]curationItem, error) {
	itemResult := []curationItem{}
	query := &neoism.CypherQuery{
		Statement: `MATCH (n:Curation {uuid:{uuid}})-[rel:SELECTS]->(t:Thing) RETURN t`,
		Parameters: map[string]interface{}{
			"uuid": uuid,
		},
		Result: &itemResult,
	}

	err := s.conn.CypherBatch([]*neoism.CypherQuery{query})
	return itemResult, err
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
		url = "http://localhost:7474/db/data"
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
			Statement: fmt.Sprintf("MATCH (mc:Thing {uuid: '%v'}) DETACH DELETE mc", storyPackageUuid),
		},
	}

	err := db.CypherBatch(qs)
	assert.NoError(err)
}

func checkDbClean(db neoutils.CypherRunner, t *testing.T) {
	assert := assert.New(t)

	result := []struct {
		Uuid string `json:"org.uuid"`
	}{}

	checkGraph := neoism.CypherQuery{
		Statement: `MATCH (org:Thing) WHERE org.uuid in {uuids} RETURN org.uuid`,
		Parameters: neoism.Props{
			"uuids": []string{storyPackageUuid},
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
