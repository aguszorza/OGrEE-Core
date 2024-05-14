package controllers_test

import (
	"encoding/json"
	"net/http"
	"net/url"
	"p3/models"
	"p3/test/e2e"
	"p3/test/integration"
	"p3/utils"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func init() {
	integration.RequireCreateSite("site-no-temperature")
	integration.RequireCreateBuilding("site-no-temperature", "building-1")
	integration.RequireCreateBuilding("site-no-temperature", "building-2")
	integration.RequireCreateBuilding("site-no-temperature", "building-3")
	integration.RequireCreateRoom("site-no-temperature.building-1", "room-1")
	integration.RequireCreateRoom("site-no-temperature.building-1", "room-2")
	integration.RequireCreateRoom("site-no-temperature.building-2", "room-1")
	integration.RequireCreateRack("site-no-temperature.building-1.room-1", "rack-1")
	integration.RequireCreateRack("site-no-temperature.building-1.room-1", "rack-2")
	integration.RequireCreateDevice("site-no-temperature.building-1.room-1.rack-2", "device-1")
	integration.RequireCreateRack("site-no-temperature.building-1.room-2", "rack-1")
	integration.RequireCreateSite("site-with-temperature")
	integration.RequireCreateBuilding("site-with-temperature", "building-3")
	var ManagerUserRoles = map[string]models.Role{
		models.ROOT_DOMAIN: models.Manager,
	}
	temperatureData := map[string]any{
		"attributes": map[string]any{
			"temperatureUnit": "30",
		},
	}

	models.UpdateObject("site", "site-with-temperature", temperatureData, true, ManagerUserRoles, false)
	layer := map[string]any{
		"slug":          "racks-layer",
		"filter":        "category=rack",
		"applicability": "site-no-temperature.building-1.room-1",
	}
	models.CreateEntity(utils.LAYER, layer, ManagerUserRoles)
	layer2 := map[string]any{
		"slug":          "racks-1-layer",
		"filter":        "category=rack & name=rack-1",
		"applicability": "site-no-temperature.building-1.room-*",
	}
	models.CreateEntity(utils.LAYER, layer2, ManagerUserRoles)
}

func testInvalidBody(t *testing.T, httpMethod string, endpoint string) {
	e2e.TestInvalidBody(t, httpMethod, endpoint, "Error while decoding request body")
}

func TestCreateEntityInvalidBody(t *testing.T) {
	testInvalidBody(t, "POST", "/api/sites")
}

// Tests domain bulk creation (/api/domains/bulk)
func TestCreateBulkInvalidBody(t *testing.T) {
	testInvalidBody(t, "POST", "/api/domains/bulk")
}

func TestCreateBulkDomains(t *testing.T) {
	// Test create two separate domains
	requestBody := []byte(`[
		{
			"name": "domain1",
			"parentId": "",
			"color": "ffffff"
		},
		{
			"name": "domain2",
			"parentId": "",
			"description": "Domain 2"
		}
	]`)

	recorder := e2e.MakeRequest("POST", "/api/domains/bulk", requestBody)
	assert.Equal(t, http.StatusOK, recorder.Code)

	var response map[string]interface{}
	json.Unmarshal(recorder.Body.Bytes(), &response)
	message, exists := response["domain1"].(string)
	assert.True(t, exists)
	assert.Equal(t, "successfully created domain", message)

	message, exists = response["domain2"].(string)
	assert.True(t, exists)
	assert.Equal(t, "successfully created domain", message)
}

func TestCreateBulkDomainWithSubdomains(t *testing.T) {
	// Test create one domaain with a sub domain
	requestBody := []byte(`[
		{
			"name": "domain3",
			"description": "Domain 3",
			"color": "00ED00",
			"domains": [
				{
					"name": "subDomain1",
					"description": "subDomain 1",
					"color": "ffffff"
				}
			]
		},
		{
			"name": "domain4",
			"description": "Domain 4",
			"color": "00ED00",
			"domains": [
				{
					"name": "subDomain1",
					"description": "subDomain 1",
					"color": "00ED00"
				},
				{
					"name": "subDomain2",
					"description": "subDomain 2",
					"color": "ffffff"
				}
			]
		}
	]`)

	recorder := e2e.MakeRequest("POST", "/api/domains/bulk", requestBody)
	assert.Equal(t, http.StatusOK, recorder.Code)

	var response map[string]interface{}
	json.Unmarshal(recorder.Body.Bytes(), &response)
	message, exists := response["domain3"].(string)
	assert.True(t, exists)
	assert.Equal(t, "successfully created domain", message)

	message, exists = response["domain3.subDomain1"].(string)
	assert.True(t, exists)
	assert.Equal(t, "successfully created domain", message)

	message, exists = response["domain4"].(string)
	assert.True(t, exists)
	assert.Equal(t, "successfully created domain", message)

	message, exists = response["domain4.subDomain1"].(string)
	assert.True(t, exists)
	assert.Equal(t, "successfully created domain", message)

	message, exists = response["domain4.subDomain2"].(string)
	assert.True(t, exists)
	assert.Equal(t, "successfully created domain", message)
}

func TestCreateBulkDomainWithDuplicateError(t *testing.T) {
	// Test try to create a domain that already exists
	requestBody := []byte(`[
		{
			"name": "domain3",
			"description": "Domain 3",
			"color": "00ED00"
		}
	]`)

	recorder := e2e.MakeRequest("POST", "/api/domains/bulk", requestBody)
	assert.Equal(t, http.StatusOK, recorder.Code)

	var response map[string]interface{}
	json.Unmarshal(recorder.Body.Bytes(), &response)
	message, exists := response["domain3"].(string)
	assert.True(t, exists)
	assert.Equal(t, "Error while creating domain: Duplicates not allowed", message)
}

// Tests delete subdomains (/api/objects)
func TestDeleteSubdomains(t *testing.T) {
	// Test delete subdomain using a pattern
	params, _ := url.ParseQuery("id=domain3.*")

	recorder := e2e.MakeRequest("DELETE", "/api/objects?"+params.Encode(), nil)
	assert.Equal(t, http.StatusOK, recorder.Code)

	var response map[string]interface{}
	json.Unmarshal(recorder.Body.Bytes(), &response)
	message, exists := response["message"].(string)
	assert.True(t, exists)
	assert.Equal(t, "successfully deleted objects", message)

	data, exists := response["data"].([]interface{})
	assert.True(t, exists)
	assert.Equal(t, 1, len(data))
	deletedDomain := data[0].(map[string]interface{})
	id, exists := deletedDomain["id"].(string)
	assert.True(t, exists)
	assert.Equal(t, "domain3.subDomain1", id)
}

// Tests handle complex filters (/api/objects/search)
func TestComplexFilterSearchInvalidBody(t *testing.T) {
	testInvalidBody(t, "POST", "/api/objects/search")
}

func TestComplexFilterWithNoFilterInput(t *testing.T) {
	requestBody := []byte(`{}`)

	recorder := e2e.MakeRequest("POST", "/api/objects/search", requestBody)
	assert.Equal(t, http.StatusBadRequest, recorder.Code)

	var response map[string]interface{}
	json.Unmarshal(recorder.Body.Bytes(), &response)
	message, exists := response["message"].(string)
	assert.True(t, exists)
	assert.Equal(t, "Invalid body format: must contain a filter key with a not empty string as value", message)
}

func TestComplexFilterSearch(t *testing.T) {
	// Test get subdomains of domain4 with color 00ED00
	requestBody := []byte(`{
		"filter": "id=domain4.* & color=00ED00"
	}`)

	recorder := e2e.MakeRequest("POST", "/api/objects/search", requestBody)
	assert.Equal(t, http.StatusOK, recorder.Code)

	var response map[string]interface{}
	json.Unmarshal(recorder.Body.Bytes(), &response)
	message, exists := response["message"].(string)
	assert.True(t, exists)
	assert.Equal(t, "successfully processed request", message)

	data, exists := response["data"].([]interface{})
	assert.True(t, exists)
	assert.Equal(t, 1, len(data))

	domain := data[0].(map[string]interface{})
	id, exists := domain["id"].(string)
	assert.True(t, exists)
	assert.Equal(t, "domain4.subDomain1", id)
}

func TestComplexFilterSearchWithStartDateFilter(t *testing.T) {
	// Test get subdomains of domain4 with color 00ED00 and different startDate
	requestBody := []byte(`{
		"filter": "id=domain4.* & color=00ED00"
	}`)

	yesterday := time.Now().Add(-24 * time.Hour)
	tomorrow := time.Now().Add(24 * time.Hour)
	recorder := e2e.MakeRequest("POST", "/api/objects/search?startDate="+yesterday.Format("2006-01-02"), requestBody)
	assert.Equal(t, http.StatusOK, recorder.Code)

	var response map[string]interface{}
	json.Unmarshal(recorder.Body.Bytes(), &response)
	message, exists := response["message"].(string)
	assert.True(t, exists)
	assert.Equal(t, "successfully processed request", message)

	data, exists := response["data"].([]interface{})
	assert.True(t, exists)
	assert.Equal(t, 1, len(data))

	recorder = e2e.MakeRequest("POST", "/api/objects/search?startDate="+tomorrow.Format("2006-01-02"), requestBody)
	assert.Equal(t, http.StatusOK, recorder.Code)

	json.Unmarshal(recorder.Body.Bytes(), &response)
	message, exists = response["message"].(string)
	assert.True(t, exists)
	assert.Equal(t, "successfully processed request", message)

	data, exists = response["data"].([]interface{})
	assert.True(t, exists)
	assert.Equal(t, 0, len(data))
}

func TestComplexFilterSearchWithEndtDateFilter(t *testing.T) {
	// Test get subdomains of domain4 with color 00ED00 and different endDate
	requestBody := []byte(`{
		"filter": "id=domain4.* & color=00ED00"
	}`)

	yesterday := time.Now().Add(-24 * time.Hour)
	tomorrow := time.Now().Add(24 * time.Hour)
	recorder := e2e.MakeRequest("POST", "/api/objects/search?endDate="+tomorrow.Format("2006-01-02"), requestBody)
	assert.Equal(t, http.StatusOK, recorder.Code)

	var response map[string]interface{}
	json.Unmarshal(recorder.Body.Bytes(), &response)
	message, exists := response["message"].(string)
	assert.True(t, exists)
	assert.Equal(t, "successfully processed request", message)

	data, exists := response["data"].([]interface{})
	assert.True(t, exists)
	assert.Equal(t, 1, len(data))

	recorder = e2e.MakeRequest("POST", "/api/objects/search?endDate="+yesterday.Format("2006-01-02"), requestBody)
	assert.Equal(t, http.StatusOK, recorder.Code)

	json.Unmarshal(recorder.Body.Bytes(), &response)
	message, exists = response["message"].(string)
	assert.True(t, exists)
	assert.Equal(t, "successfully processed request", message)

	data, exists = response["data"].([]interface{})
	assert.True(t, exists)
	assert.Equal(t, 0, len(data))
}

// Tests handle delete with complex filters (/api/objects/search)
func TestComplexFilterDelete(t *testing.T) {
	// Test delete subdomains of domain4 with color 00ED00
	requestBody := []byte(`{
		"filter": "id=domain4.* & color=00ED00"
	}`)

	recorder := e2e.MakeRequest("DELETE", "/api/objects/search", requestBody)
	assert.Equal(t, http.StatusOK, recorder.Code)

	var response map[string]interface{}
	json.Unmarshal(recorder.Body.Bytes(), &response)
	message, exists := response["message"].(string)
	assert.True(t, exists)
	assert.Equal(t, "successfully deleted objects", message)

	data, exists := response["data"].([]interface{})
	assert.True(t, exists)
	assert.Equal(t, 1, len(data))

	domain := data[0].(map[string]interface{})
	id, exists := domain["id"].(string)
	assert.True(t, exists)
	assert.Equal(t, "domain4.subDomain1", id)
}

// Tests get different entities
func TestGetDomainEntity(t *testing.T) {
	recorder := e2e.MakeRequest("GET", "/api/domains", nil)
	assert.Equal(t, http.StatusOK, recorder.Code)

	var response map[string]interface{}
	json.Unmarshal(recorder.Body.Bytes(), &response)
	message, exists := response["message"].(string)
	assert.True(t, exists)
	assert.Equal(t, "successfully got domains", message)

	// we have multiple domains
	data, exists := response["data"].(map[string]interface{})
	assert.True(t, exists)

	objects, exists := data["objects"].([]interface{})
	assert.True(t, exists)
	assert.Equal(t, true, len(objects) > 0) // we have domains created in this file and others

	// domain3 exists but domain3.subDomain1 does not
	hasDomain3 := slices.ContainsFunc(objects, func(value interface{}) bool {
		domain := value.(map[string]interface{})
		return domain["id"].(string) == "domain3"
	})
	assert.Equal(t, true, hasDomain3)

	hasDomain3Subdomain1 := slices.ContainsFunc(objects, func(value interface{}) bool {
		domain := value.(map[string]interface{})
		return domain["id"].(string) == "domain3.subDomain1"
	})
	assert.Equal(t, false, hasDomain3Subdomain1)
}

func TestGetBuildingsEntity(t *testing.T) {
	recorder := e2e.MakeRequest("GET", "/api/buildings", nil)
	assert.Equal(t, http.StatusOK, recorder.Code)

	var response map[string]interface{}
	json.Unmarshal(recorder.Body.Bytes(), &response)
	message, exists := response["message"].(string)
	assert.True(t, exists)
	assert.Equal(t, "successfully got buildings", message)

	// we have multiple buildings
	data, exists := response["data"].(map[string]interface{})
	assert.True(t, exists)

	objects, exists := data["objects"].([]interface{})
	assert.True(t, exists)
	assert.Equal(t, true, len(objects) > 0)
}

func TestGetUnknownEntity(t *testing.T) {
	recorder := e2e.MakeRequest("GET", "/api/unknown", nil)
	assert.Equal(t, http.StatusNotFound, recorder.Code)
}

func TestGetDomainEntitiesFilteredByColor(t *testing.T) {
	recorder := e2e.MakeRequest("GET", "/api/domains?color=00ED00", nil)
	assert.Equal(t, http.StatusOK, recorder.Code)

	var response map[string]interface{}
	json.Unmarshal(recorder.Body.Bytes(), &response)
	message, exists := response["message"].(string)
	assert.True(t, exists)
	assert.Equal(t, "successfully got query for domain", message)

	// we have multiple domains
	data, exists := response["data"].(map[string]interface{})
	assert.True(t, exists)

	objects, exists := data["objects"].([]interface{})
	assert.True(t, exists)
	assert.Equal(t, 2, len(objects)) // domain3 and domain4
}

// Test get temperature unit
func TestGetTemperatureForDomain(t *testing.T) {
	recorder := e2e.MakeRequest("GET", "/api/tempunits/domain3", nil)
	assert.Equal(t, http.StatusNotFound, recorder.Code)

	var response map[string]interface{}
	json.Unmarshal(recorder.Body.Bytes(), &response)
	message, exists := response["message"].(string)
	assert.True(t, exists)
	assert.Equal(t, "Could not find parent site for given object", message)
}

func TestGetTemperatureForParentWithNoTemperature(t *testing.T) {
	recorder := e2e.MakeRequest("GET", "/api/tempunits/site-no-temperature.building-1", nil)
	assert.Equal(t, http.StatusNotFound, recorder.Code)

	var response map[string]interface{}
	json.Unmarshal(recorder.Body.Bytes(), &response)
	message, exists := response["message"].(string)
	assert.True(t, exists)
	assert.Equal(t, "Parent site has no temperatureUnit in attributes", message)
}

func TestGetTemperature(t *testing.T) {
	recorder := e2e.MakeRequest("GET", "/api/tempunits/site-with-temperature.building-3", nil)
	assert.Equal(t, http.StatusOK, recorder.Code)

	var response map[string]interface{}
	json.Unmarshal(recorder.Body.Bytes(), &response)
	message, exists := response["message"].(string)
	assert.True(t, exists)
	assert.Equal(t, "successfully got attribute from object's parent site", message)

	data, exists := response["data"].(map[string]interface{})
	assert.True(t, exists)
	temperatureUnit, exists := data["temperatureUnit"].(string)
	assert.True(t, exists)
	assert.Equal(t, "30", temperatureUnit)
}

// Tests get subentities
func TestErrorGetRoomsBuildingsInvalidHierarchy(t *testing.T) {
	recorder := e2e.MakeRequest("GET", "/api/rooms/site-no-temperature.building-1.room-1/buildings", nil)
	assert.Equal(t, http.StatusBadRequest, recorder.Code)

	var response map[string]interface{}
	json.Unmarshal(recorder.Body.Bytes(), &response)
	message, exists := response["message"].(string)
	assert.True(t, exists)
	assert.Equal(t, "Invalid set of entities in URL: first entity should be parent of the second entity", message)
}

func TestErrorGetSiteRoomsUnknownEntity(t *testing.T) {
	recorder := e2e.MakeRequest("GET", "/api/sites/unknown/rooms", nil)
	assert.Equal(t, http.StatusNotFound, recorder.Code)

	var response map[string]interface{}
	json.Unmarshal(recorder.Body.Bytes(), &response)
	message, exists := response["message"].(string)
	assert.True(t, exists)
	assert.Equal(t, "Nothing matches this request", message)
}

func TestGetSitesRooms(t *testing.T) {
	recorder := e2e.MakeRequest("GET", "/api/sites/site-no-temperature/rooms", nil)
	assert.Equal(t, http.StatusOK, recorder.Code)

	var response map[string]interface{}
	json.Unmarshal(recorder.Body.Bytes(), &response)
	message, exists := response["message"].(string)
	assert.True(t, exists)
	assert.Equal(t, "successfully got object", message)

	data, exists := response["data"].(map[string]interface{})
	assert.True(t, exists)
	objects, exists := data["objects"].([]interface{})
	assert.True(t, exists)
	assert.Equal(t, 3, len(objects))

	areRooms := true
	for _, element := range objects {
		if element.(map[string]interface{})["category"] != "room" {
			areRooms = false
			break
		}
	}
	assert.True(t, areRooms)
}

func TestGetHierarchyAttributes(t *testing.T) {
	recorder := e2e.MakeRequest("GET", "/api/hierarchy/attributes", nil)
	assert.Equal(t, http.StatusOK, recorder.Code)

	var response map[string]interface{}
	json.Unmarshal(recorder.Body.Bytes(), &response)
	message, exists := response["message"].(string)
	assert.True(t, exists)
	assert.Equal(t, "successfully got attrs hierarchy", message)

	data, exists := response["data"].(map[string]interface{})
	assert.True(t, exists)
	keys := make([]int, len(data))
	assert.True(t, len(keys) > 0)

	// we test the color attribute is present for domain1
	domain1, exists := data["domain1"].(map[string]interface{})
	assert.True(t, exists)
	color, exists := domain1["color"].(string)
	assert.True(t, exists)
	assert.Equal(t, "ffffff", color)
}

// Tests link and unlink entity
func TestErrorUnlinkWithNotAllowedAttributes(t *testing.T) {
	requestBody := []byte(`{
		"name": "StrayRoom",
		"other": "other"
	}`)

	recorder := e2e.MakeRequest("PATCH", "/api/rooms/site-no-temperature.building-2.room-1/unlink", requestBody)
	assert.Equal(t, http.StatusBadRequest, recorder.Code)
	var response map[string]interface{}
	json.Unmarshal(recorder.Body.Bytes(), &response)
	message, exists := response["message"].(string)
	assert.True(t, exists)
	assert.Equal(t, "Body must be empty or only contain valid name", message)
}

func TestUnlinkRoom(t *testing.T) {
	requestBody := []byte(`{
		"name": "StrayRoom"
	}`)

	recorder := e2e.MakeRequest("PATCH", "/api/rooms/site-no-temperature.building-2.room-1/unlink", requestBody)
	assert.Equal(t, http.StatusOK, recorder.Code)

	var response map[string]interface{}
	json.Unmarshal(recorder.Body.Bytes(), &response)
	message, exists := response["message"].(string)
	assert.True(t, exists)
	assert.Equal(t, "successfully unlinked", message)

	// We verify room-1 does not exist
	recorder = e2e.MakeRequest("GET", "/api/rooms/site-no-temperature.building-2.room-1", requestBody)
	assert.Equal(t, http.StatusNotFound, recorder.Code)
	json.Unmarshal(recorder.Body.Bytes(), &response)
	message, exists = response["message"].(string)
	assert.True(t, exists)
	assert.Equal(t, "Nothing matches this request", message)

	// We verify the StrayRoom exists
	recorder = e2e.MakeRequest("GET", "/api/stray-objects/StrayRoom", requestBody)
	assert.Equal(t, http.StatusOK, recorder.Code)
	json.Unmarshal(recorder.Body.Bytes(), &response)
	data, exists := response["data"].(map[string]interface{})
	assert.True(t, exists)
	id := data["id"].(string)
	assert.Equal(t, "StrayRoom", id)
}

func TestErrorLinkWithoutParentId(t *testing.T) {
	requestBody := []byte(`{
		"name": "room-1"
	}`)

	recorder := e2e.MakeRequest("PATCH", "/api/stray-objects/StrayRoom/link", requestBody)
	assert.Equal(t, http.StatusBadRequest, recorder.Code)
	var response map[string]interface{}
	json.Unmarshal(recorder.Body.Bytes(), &response)
	message, exists := response["message"].(string)
	assert.True(t, exists)
	assert.Equal(t, "Error while decoding request body: must contain parentId", message)
}

func TestLinkRoom(t *testing.T) {
	requestBody := []byte(`{
		"parentId": "site-no-temperature.building-2",
		"name": "room-1"
	}`)

	recorder := e2e.MakeRequest("PATCH", "/api/stray-objects/StrayRoom/link", requestBody)
	assert.Equal(t, http.StatusOK, recorder.Code)

	var response map[string]interface{}
	json.Unmarshal(recorder.Body.Bytes(), &response)
	message, exists := response["message"].(string)
	assert.True(t, exists)
	assert.Equal(t, "successfully linked", message)

	// We verify the StrayRoom  does not exist
	recorder = e2e.MakeRequest("GET", "/api/stray-objects/StrayRoom", requestBody)
	assert.Equal(t, http.StatusNotFound, recorder.Code)
	json.Unmarshal(recorder.Body.Bytes(), &response)
	message, exists = response["message"].(string)
	assert.True(t, exists)
	assert.Equal(t, "Nothing matches this request", message)

	// We verify room-1 exists again
	recorder = e2e.MakeRequest("GET", "/api/rooms/site-no-temperature.building-2.room-1", requestBody)
	assert.Equal(t, http.StatusOK, recorder.Code)
	json.Unmarshal(recorder.Body.Bytes(), &response)
	data, exists := response["data"].(map[string]interface{})
	assert.True(t, exists)
	id := data["id"].(string)
	assert.Equal(t, "site-no-temperature.building-2.room-1", id)
}

// Tests entity validation
func TestValidateInvalidBody(t *testing.T) {
	testInvalidBody(t, "POST", "/api/validate/rooms")
}

func TestValidateNonExistentEntity(t *testing.T) {
	requestBody := []byte(`{}`)

	recorder := e2e.MakeRequest("POST", "/api/validate/invalid", requestBody)
	assert.Equal(t, http.StatusNotFound, recorder.Code)
}

func TestValidateEntityWithoutAttributes(t *testing.T) {
	requestBody := []byte(`{
		"category": "room",
		"description": "room",
		"domain": "domain1",
		"name": "roomA",
		"parentId": "site-no-temperature.building-1"
	}`)

	recorder := e2e.MakeRequest("POST", "/api/validate/rooms", requestBody)
	assert.Equal(t, http.StatusBadRequest, recorder.Code)

	var response map[string]interface{}
	json.Unmarshal(recorder.Body.Bytes(), &response)
	message, exists := response["message"].(string)
	assert.True(t, exists)
	assert.Equal(t, "JSON body doesn't validate with the expected JSON schema", message)
}

func TestValidateEntityNonExistentDomain(t *testing.T) {
	requestBody := []byte(`{
		"attributes": {
			"floorUnit": "t",
			"height": 2.8,
			"heightUnit": "m",
			"axisOrientation": "+x+y",
			"rotation": -90,
			"posXY": [0, 0],
			"posXYUnit": "m",
			"size": [-13, -2.9],
			"sizeUnit": "m",
			"template": ""
		},
		"category": "room",
		"description": "room",
		"domain": "invalid",
		"name": "roomA",
		"parentId": "site-no-temperature.building-1"
	}`)

	recorder := e2e.MakeRequest("POST", "/api/validate/rooms", requestBody)
	assert.Equal(t, http.StatusNotFound, recorder.Code)

	var response map[string]interface{}
	json.Unmarshal(recorder.Body.Bytes(), &response)
	message, exists := response["message"].(string)
	assert.True(t, exists)
	assert.Equal(t, "Domain not found: invalid", message)
}

func TestValidateEntityInvalidDomain(t *testing.T) {
	requestBody := []byte(`{
		"attributes": {
			"floorUnit": "t",
			"height": 2.8,
			"heightUnit": "m",
			"axisOrientation": "+x+y",
			"rotation": -90,
			"posXY": [0, 0],
			"posXYUnit": "m",
			"size": [-13, -2.9],
			"sizeUnit": "m",
			"template": ""
		},
		"category": "room",
		"description": "room",
		"domain": "domain1",
		"name": "roomA",
		"parentId": "site-no-temperature.building-1"
	}`)

	recorder := e2e.MakeRequest("POST", "/api/validate/rooms", requestBody)
	assert.Equal(t, http.StatusBadRequest, recorder.Code)

	var response map[string]interface{}
	json.Unmarshal(recorder.Body.Bytes(), &response)
	message, exists := response["message"].(string)
	assert.True(t, exists)
	assert.Equal(t, "Object domain is not equal or child of parent's domain", message)
}

func TestValidateValidRoomEntity(t *testing.T) {
	requestBody := []byte(`{
		"attributes": {
			"floorUnit": "t",
			"height": 2.8,
			"heightUnit": "m",
			"axisOrientation": "+x+y",
			"rotation": -90,
			"posXY": [0, 0],
			"posXYUnit": "m",
			"size": [-13, -2.9],
			"sizeUnit": "m",
			"template": ""
		},
		"category": "room",
		"description": "room",
		"domain": "` + integration.TestDBName + `",
		"name": "roomA",
		"parentId": "site-no-temperature.building-1"
	}`)

	recorder := e2e.MakeRequest("POST", "/api/validate/rooms", requestBody)
	assert.Equal(t, http.StatusOK, recorder.Code)

	var response map[string]interface{}
	json.Unmarshal(recorder.Body.Bytes(), &response)
	message, exists := response["message"].(string)
	assert.True(t, exists)
	assert.Equal(t, "This object can be created", message)
}

func TestErrorValidateValidRoomEntityNotEnoughPermissions(t *testing.T) {
	requestBody := []byte(`{
		"attributes": {
			"floorUnit": "t",
			"height": 2.8,
			"heightUnit": "m",
			"axisOrientation": "+x+y",
			"rotation": -90,
			"posXY": [0, 0],
			"posXYUnit": "m",
			"size": [-13, -2.9],
			"sizeUnit": "m",
			"template": ""
		},
		"category": "room",
		"description": "room",
		"domain": "` + integration.TestDBName + `",
		"name": "roomA",
		"parentId": "site-no-temperature.building-1"
	}`)
	recorder := e2e.MakeRequestWithUser("POST", "/api/validate/rooms", requestBody, "viewer")
	assert.Equal(t, http.StatusUnauthorized, recorder.Code)

	var response map[string]interface{}
	json.Unmarshal(recorder.Body.Bytes(), &response)
	message, exists := response["message"].(string)
	assert.True(t, exists)
	assert.Equal(t, "This user does not have sufficient permissions to create this object under this domain ", message)
}

func TestGetStats(t *testing.T) {
	recorder := e2e.MakeRequest("GET", "/api/stats", nil)
	assert.Equal(t, http.StatusOK, recorder.Code)

	var response map[string]interface{}
	json.Unmarshal(recorder.Body.Bytes(), &response)
	numberOfRacks, exists := response["Number of racks:"].(float64)
	assert.True(t, exists)
	assert.True(t, numberOfRacks > 0)
}

func TestGetApiVersion(t *testing.T) {
	recorder := e2e.MakeRequest("GET", "/api/version", nil)
	assert.Equal(t, http.StatusOK, recorder.Code)

	var response map[string]interface{}
	json.Unmarshal(recorder.Body.Bytes(), &response)
	status, exists := response["status"].(bool)
	assert.True(t, exists)
	assert.True(t, status)

	data, exists := response["data"].(map[string]interface{})
	assert.True(t, exists)
	customer, exists := data["Customer"].(string)
	assert.True(t, exists)
	assert.True(t, len(customer) > 0)
}

// Tests layers objects
func TestGetLayersObjectsRootRequired(t *testing.T) {
	recorder := e2e.MakeRequest("GET", "/api/layers/racks-layer/objects", nil)
	assert.Equal(t, http.StatusBadRequest, recorder.Code)

	var response map[string]interface{}
	json.Unmarshal(recorder.Body.Bytes(), &response)
	message, exists := response["message"].(string)
	assert.True(t, exists)
	assert.Equal(t, "Query param root is mandatory", message)
}

func TestGetLayersObjectsLayerUnknown(t *testing.T) {
	recorder := e2e.MakeRequest("GET", "/api/layers/unknown/objects?root=site-no-temperature.building-1.room-1", nil)
	assert.Equal(t, http.StatusNotFound, recorder.Code)
}

// func TestGetLayersObjectsWithSimpleFilter(t *testing.T) {
// 	recorder := e2e.MakeRequest("GET", "/api/layers/racks-layer/objects?root=site-no-temperature.building-1.room-1", nil)
// 	assert.Equal(t, http.StatusOK, recorder.Code)

// 	var response map[string]interface{}
// 	json.Unmarshal(recorder.Body.Bytes(), &response)
// 	message, exists := response["message"].(string)
// 	assert.True(t, exists)
// 	assert.Equal(t, "successfully processed request", message)

// 	data, exists := response["data"].([]any)
// 	assert.True(t, exists)
// 	assert.Equal(t, 2, len(data))

// 	condition := true
// 	for _, rack := range data {
// 		condition = condition && rack.(map[string]any)["parentId"] == "site-no-temperature.building-1.room-1"
// 		condition = condition && rack.(map[string]any)["category"] == "rack"
// 	}

// 	assert.True(t, condition)
// }

// func TestGetLayersObjectsWithDoubleFilter(t *testing.T) {
// 	recorder := e2e.MakeRequest("GET", "/api/layers/racks-1-layer/objects?root=site-no-temperature.building-1.room-*", nil)
// 	assert.Equal(t, http.StatusOK, recorder.Code)

// 	var response map[string]interface{}
// 	json.Unmarshal(recorder.Body.Bytes(), &response)
// 	message, exists := response["message"].(string)
// 	assert.True(t, exists)
// 	assert.Equal(t, "successfully processed request", message)

// 	data, exists := response["data"].([]any)
// 	assert.True(t, exists)
// 	assert.Equal(t, 2, len(data))

// 	condition := true
// 	for _, rack := range data {
// 		condition = condition && strings.HasPrefix(rack.(map[string]any)["parentId"].(string), "site-no-temperature.building-1.room-")
// 		condition = condition && rack.(map[string]any)["category"] == "rack"
// 		condition = condition && rack.(map[string]any)["name"] == "rack-1"
// 	}

// 	assert.True(t, condition)
// }
