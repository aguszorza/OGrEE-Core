package models

import (
	"context"
	"fmt"
	u "p3/utils"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Function will recursively iterate through nested obj
// and accumulate whatever is found into category arrays
func parseDataForNonStdResult(ent string, eNum, end int, data map[string]interface{}) map[string][]map[string]interface{} {
	var nxt string
	ans := map[string][]map[string]interface{}{}
	add := data[u.EntityToString(eNum+1)+"s"].([]map[string]interface{})

	//NEW REWRITE
	for i := eNum; i+2 < end; i++ {
		idx := u.EntityToString(i + 1)
		//println("trying IDX: ", idx)
		firstArr := add

		ans[idx+"s"] = firstArr

		for q := range firstArr {
			nxt = u.EntityToString(i + 2)
			println("NXT: ", nxt)
			ans[nxt+"s"] = append(ans[nxt+"s"],
				ans[idx+"s"][q][nxt+"s"].([]map[string]interface{})...)
		}
		add = ans[nxt+"s"]

	}

	return ans
}

func CreateEntity(entity int, t map[string]interface{}) (map[string]interface{}, string) {
	message := ""
	if resp, ok := ValidateEntity(entity, t); !ok {
		return resp, "validate"
	}

	//Set timestamp
	t["createdDate"] = primitive.NewDateTimeFromTime(time.Now())
	t["lastUpdated"] = t["createdDate"]

	ctx, cancel := u.Connect()
	entStr := u.EntityToString(entity)
	res, e := GetDB().Collection(entStr).InsertOne(ctx, t)
	if e != nil {
		if strings.Contains(e.Error(), "E11000") {
			return u.Message(false,
					"Error while creating "+entStr+": Duplicates not allowed"),
				"duplicate"
		}
		return u.Message(false,
				"Internal error while creating "+entStr+": "+e.Error()),
			e.Error()
	}
	defer cancel()

	//Remove _id
	t["id"] = res.InsertedID
	//t = fixID(t)

	switch entity {
	case u.ROOMTMPL:
		message = "successfully created room_template"
	case u.OBJTMPL:
		message = "successfully created obj_template"
	case u.BLDGTMPL:
		message = "successfully created bldg_template"
	default:
		message = "successfully created object"
	}

	resp := u.Message(true, message)
	resp["data"] = t
	return resp, ""
}

func GetEntity(req bson.M, ent string) (map[string]interface{}, string) {
	t := map[string]interface{}{}

	ctx, cancel := u.Connect()
	e := GetDB().Collection(ent).FindOne(ctx, req).Decode(&t)
	if e != nil {
		return nil, e.Error()
	}
	defer cancel()
	//Remove _id
	t = fixID(t)

	//If entity has '_' remove it
	if strings.Contains(ent, "_") {
		FixUnderScore(t)
	}
	return t, ""
}

func GetManyEntities(ent string, req bson.M, opts *options.FindOptions) ([]map[string]interface{}, string) {
	data := make([]map[string]interface{}, 0)
	ctx, cancel := u.Connect()
	c, err := GetDB().Collection(ent).Find(ctx, req, opts)
	if err != nil {
		fmt.Println(err)
		return nil, err.Error()
	}
	defer cancel()

	data, e1 := ExtractCursor(c, ctx)
	if e1 != "" {
		fmt.Println(e1)
		return nil, e1
	}

	//Remove underscore If the entity has '_'
	if strings.Contains(ent, "_") == true {
		for i := range data {
			FixUnderScore(data[i])
		}
	}

	return data, ""
}

func GetSiteParentTempUnit(id string) (string, string) {
	data := map[string]interface{}{}

	// Get all collections names
	ctx, cancel := u.Connect()
	db := GetDB()
	collNames, err := db.ListCollectionNames(ctx, bson.D{})
	if err != nil {
		fmt.Println(err.Error())
		return "", err.Error()
	}

	// Search all collections for given object and its parents until site found
	searchId := id
out:
	for {
		objID, _ := primitive.ObjectIDFromHex(searchId)
		searchId = ""
		for _, collName := range collNames {
			err := db.Collection(collName).FindOne(ctx, bson.M{"_id": objID}).Decode(&data)
			if err == nil {
				if data["category"].(string) == "site" {
					// found site, break all loops
					break out
				} else {
					if data["parentId"] == nil {
						// got to tenant or object without parentId
						return "", "Could not find parent site for given object"
					} else {
						// current object is not a site, search its parent next
						searchId = data["parentId"].(string)
						break
					}
				}
			}
		}
		if searchId == "" {
			// id not found in any collection
			return "", "Could not find parent site for given object"
		}

	}

	defer cancel()

	if tempUnit := data["attributes"].(map[string]interface{})["temperatureUnit"]; tempUnit == nil {
		return "", "Parent site has no temperatureUnit in attributes"
	} else {
		return tempUnit.(string), ""
	}
}

func GetEntityCount(entity int) int64 {
	ent := u.EntityToString(entity)
	ctx, cancel := u.Connect()
	ans, e := GetDB().Collection(ent).CountDocuments(ctx, bson.M{}, nil)
	if e != nil {
		println(e.Error())
		return -1
	}
	defer cancel()
	return ans
}

func CommandRunner(cmd interface{}) *mongo.SingleResult {
	ctx, cancel := u.Connect()
	result := GetDB().RunCommand(ctx, cmd, nil)
	defer cancel()
	return result
}

func GetStats() map[string]interface{} {
	ans := map[string]interface{}{}
	t := map[string]interface{}{}
	t2 := map[string]interface{}{}

	for i := 0; i <= u.STRAYSENSOR; i++ {
		num := GetEntityCount(i)
		if num == -1 {
			num = 0
		}

		ans["Number of "+u.EntityToString(i)+"s:"] = num
	}

	cmd := bson.D{{"dbStats", 1}, {"scale", 1024}}
	cmd2 := bson.D{{"serverStatus", 1}} //This cmd gives too much info
	//logicalSessionRecordCache,lastSessionsCollectionJobTimestamp

	if e := CommandRunner(cmd).Decode(&t); e != nil {
		println(e.Error())
		return nil
	}
	if e := CommandRunner(cmd2).Decode(&t2); e != nil {
		println(e.Error())
		return nil
	}

	ans["Number of Hierarchal Objects"] = t["collections"]
	ans["Last Job Timestamp"] =
		t2["logicalSessionRecordCache"].(map[string]interface{})["lastTransactionReaperJobTimestamp"]

	return ans
}

func DeleteEntityManual(entity string, req bson.M) (map[string]interface{}, string) {
	//Finally delete the Entity
	ctx, cancel := u.Connect()
	c, _ := GetDB().Collection(entity).DeleteOne(ctx, req)
	if c.DeletedCount == 0 {
		return u.Message(false, "There was an error in deleting the entity"), "not found"
	}
	defer cancel()

	return u.Message(true, "success"), ""
}

func DeleteEntity(entity string, id primitive.ObjectID) (map[string]interface{}, string) {
	var t map[string]interface{}
	var e string
	eNum := u.EntityStrToInt(entity)
	if eNum > u.DEVICE {
		//Delete the non hierarchal objects
		t, e = GetEntityHierarchy(id, entity, eNum, eNum+eNum)
	} else {
		t, e = GetEntityHierarchy(id, entity, eNum, u.AC)
	}

	if e != "" {
		return u.Message(false,
			"There was an error in deleting the entity: "+e), "not found"
	}

	return deleteHelper(t, eNum)
}

func deleteHelper(t map[string]interface{}, ent int) (map[string]interface{}, string) {
	if t != nil {

		if v, ok := t["children"]; ok {
			if x, ok := v.([]map[string]interface{}); ok {
				for i := range x {
					if ent == u.STRAYDEV {
						deleteHelper(x[i], ent)
					} else {
						deleteHelper(x[i], ent+1)
					}

				}
			}
		}

		println("So we got: ", ent)

		if ent == u.RACK {
			ctx, cancel := u.Connect()
			GetDB().Collection("sensor").DeleteMany(ctx,
				bson.M{"parentId": t["id"].(primitive.ObjectID).Hex()})

			GetDB().Collection("group").DeleteMany(ctx,
				bson.M{"parentId": t["id"].(primitive.ObjectID).Hex()})
			defer cancel()
		}

		//Delete associated non hierarchal objs
		if ent == u.ROOM {
			//ITER Through all nonhierarchal objs
			ctx, cancel := u.Connect()
			for i := u.AC; i < u.GROUP+1; i++ {
				ent := u.EntityToString(i)
				GetDB().Collection(ent).DeleteMany(ctx, bson.M{"parentId": t["id"].(primitive.ObjectID).Hex()})
			}
			defer cancel()
		}

		//Delete hierarchy under stray-device
		if ent == u.STRAYDEV {
			ctx, cancel := u.Connect()
			entity := u.EntityToString(u.STRAYSENSOR)
			GetDB().Collection(entity).DeleteMany(ctx, bson.M{"parentId": t["id"].(primitive.ObjectID).Hex()})
			defer cancel()
		}

		if ent == u.DEVICE {
			DeleteDeviceF(t["id"].(primitive.ObjectID))
		} else {
			ctx, cancel := u.Connect()
			entity := u.EntityToString(ent)
			println(entity)
			c, _ := GetDB().Collection(entity).DeleteOne(ctx, bson.M{"_id": t["id"].(primitive.ObjectID)})
			if c.DeletedCount == 0 {
				return u.Message(false, "There was an error in deleting the entity"), "not found"
			}
			defer cancel()

		}
	}
	return nil, ""
}

func UpdateEntity(ent string, req bson.M, t *map[string]interface{}, isPatch bool) (map[string]interface{}, string) {
	var e *mongo.SingleResult
	updatedDoc := bson.M{}
	retDoc := options.ReturnDocument(options.After)

	//Update timestamp requires first obj retrieval
	//there isn't any way for mongoDB to make a field
	//immutable in a document
	oldObj, e1 := GetEntity(req, ent)
	if e1 != "" {
		return u.Message(false, "Error: "+e1), e1
	}
	(*t)["lastUpdated"] = primitive.NewDateTimeFromTime(time.Now())
	(*t)["createdDate"] = oldObj["createdDate"]

	ctx, cancel := u.Connect()
	if isPatch == true {

		msg, ok := ValidatePatch(u.EntityStrToInt(ent), *t)
		if !ok {
			return msg, "invalid"
		}
		e = GetDB().Collection(ent).FindOneAndUpdate(ctx,
			req, bson.M{"$set": *t},
			&options.FindOneAndUpdateOptions{ReturnDocument: &retDoc})
		if e.Err() != nil {
			return u.Message(false, "failure: "+e.Err().Error()), e.Err().Error()
		}
	} else {

		//Ensure that the update will be valid
		println("NOT A PATCH")
		msg, ok := ValidateEntity(u.EntityStrToInt(ent), *t)
		if !ok {
			return msg, "invalid"
		}

		e = GetDB().Collection(ent).FindOneAndReplace(ctx,
			req, *t,
			&options.FindOneAndReplaceOptions{ReturnDocument: &retDoc})

		if e.Err() != nil {
			return u.Message(false, "failure: "+e.Err().Error()), e.Err().Error()
		}
	}

	//Obtain new document then
	//Fix the _id / id discrepancy
	e.Decode(&updatedDoc)
	updatedDoc = fixID(updatedDoc)

	//Response Message
	message := ""
	switch u.EntityStrToInt(ent) {
	case u.ROOMTMPL:
		message = "successfully updated room_template"
	case u.OBJTMPL:
		message = "successfully updated obj_template"
	case u.BLDGTMPL:
		message = "successfully created bldg_template"
	default:
		message = "successfully updated object"
	}

	defer cancel()
	resp := u.Message(true, message)
	resp["data"] = updatedDoc
	return resp, ""
}

func GetEntityHierarchy(ID primitive.ObjectID, ent string, start, end int) (map[string]interface{}, string) {
	var childEnt string
	if start < end {
		top, e := GetEntity(bson.M{"_id": ID}, ent)
		if top == nil {
			return nil, e
		}
		top = fixID(top)

		children := []map[string]interface{}{}
		pid := ID.Hex()
		//Get sensors & groups
		if ent == "rack" || ent == "device" {
			//Get sensors
			sensors, _ := GetManyEntities("sensor", bson.M{"parentId": pid}, nil)

			//Get groups
			groups, _ := GetManyEntities("group", bson.M{"parentId": pid}, nil)

			if sensors != nil {
				children = append(children, sensors...)
			}
			if groups != nil {
				children = append(children, groups...)
			}
		}

		if ent == "device" || ent == "stray_device" {
			childEnt = ent
		} else {
			childEnt = u.EntityToString(start + 1)
		}

		subEnts, _ := GetManyEntities(childEnt, bson.M{"parentId": pid}, nil)

		for idx := range subEnts {
			tmp, _ := GetEntityHierarchy(subEnts[idx]["id"].(primitive.ObjectID), childEnt, start+1, end)
			if tmp != nil {
				subEnts[idx] = tmp
			}
		}

		if subEnts != nil {
			children = append(children, subEnts...)
		}

		if ent == "room" {
			for i := u.AC; i < u.CABINET+1; i++ {
				roomEnts, _ := GetManyEntities(u.EntityToString(i), bson.M{"parentId": pid}, nil)
				if roomEnts != nil {
					children = append(children, roomEnts...)
				}
			}
			for i := u.PWRPNL; i < u.SENSOR+1; i++ {
				roomEnts, _ := GetManyEntities(u.EntityToString(i), bson.M{"parentId": pid}, nil)
				if roomEnts != nil {
					children = append(children, roomEnts...)
				}
			}
			roomEnts, _ := GetManyEntities(u.EntityToString(u.CORIDOR), bson.M{"parentId": pid}, nil)
			if roomEnts != nil {
				children = append(children, roomEnts...)
			}
			roomEnts, _ = GetManyEntities(u.EntityToString(u.GROUP), bson.M{"parentId": pid}, nil)
			if roomEnts != nil {
				children = append(children, roomEnts...)
			}
		}

		if ent == "stray_device" {
			sSensors, _ := GetManyEntities("stray_sensor", bson.M{"parentId": pid}, nil)
			if sSensors != nil {
				children = append(children, sSensors...)
			}
		}

		if children != nil && len(children) > 0 {
			top["children"] = children
		}

		return top, ""
	}
	return nil, ""
}

func GetEntitiesUsingAncestorNames(ent string, id primitive.ObjectID, ancestry []map[string]string) ([]map[string]interface{}, string) {
	top, e := GetEntity(bson.M{"_id": id}, ent)
	if e != "" {
		return nil, e
	}

	//Remove _id
	top = fixID(top)

	pid := (top["id"].(primitive.ObjectID)).Hex()

	var x map[string]interface{}
	var e1 string
	for i := range ancestry {
		for k, v := range ancestry[i] {

			println("KEY:", k, " VAL:", v)

			if v == "all" {
				println("K:", k)
				//println("ID", x["_id"].(primitive.ObjectID).String())
				/*if k == "device" {
					return GetDeviceFByParentID(pid) nil, ""
				}*/
				return GetManyEntities(k, bson.M{"parentId": pid}, nil)
			}

			x, e1 = GetEntity(bson.M{"parentId": pid, "name": v}, k)
			if e1 != "" {
				println("Failing here")
				return nil, ""
			}
			pid = (x["id"].(primitive.ObjectID)).Hex()
		}
	}

	return nil, ""
}

func GetEntityUsingAncestorNames(ent string, id primitive.ObjectID, ancestry []map[string]string) (map[string]interface{}, string) {
	top, e := GetEntity(bson.M{"_id": id}, ent)
	if e != "" {
		return nil, e
	}

	//Remove _id
	top = fixID(top)

	pid := (top["id"].(primitive.ObjectID)).Hex()

	var x map[string]interface{}
	var e1 string
	for i := range ancestry {
		for k, v := range ancestry[i] {

			println("KEY:", k, " VAL:", v)

			x, e1 = GetEntity(bson.M{"parentId": pid, "name": v}, k)
			if e1 != "" {
				println("Failing here")
				return nil, ""
			}
			pid = (x["id"].(primitive.ObjectID)).Hex()
		}
	}

	//Remove _id
	x = fixID(x)
	return x, ""
}

func GetHierarchyByName(entity, name string, entnum, end int) (map[string]interface{}, string) {

	t, e := GetEntity(bson.M{"name": name}, entity)
	if e != "" {
		fmt.Println(e)
		return nil, e
	}

	//Remove _id
	t = fixID(t)

	var subEnt string
	if entnum == u.STRAYDEV {
		subEnt = "stray_device"
	} else {
		subEnt = u.EntityToString(entnum + 1)
	}

	tid := t["id"].(primitive.ObjectID).Hex()

	//Get immediate children
	children, e1 := GetManyEntities(subEnt, bson.M{"parentId": tid}, nil)
	if e1 != "" {
		println("Are we here")
		println("SUBENT: ", subEnt)
		println("PID: ", tid)
		return nil, e1
	}
	t["children"] = children

	//Get the rest of hierarchy for children
	for i := range children {
		var x map[string]interface{}
		var subIdx string
		if subEnt == "stray_device" { //only set entnum+1 for tenants
			subIdx = subEnt
		} else {
			subIdx = u.EntityToString(entnum + 1)
		}
		subID := (children[i]["id"].(primitive.ObjectID))
		x, _ =
			GetEntityHierarchy(subID, subIdx, entnum+1, end)
		if x != nil {
			children[i] = x
		}
	}

	return t, ""

}

func GetEntitiesUsingTenantAsAncestor(ent, id string, ancestry []map[string]string) ([]map[string]interface{}, string) {
	top, e := GetEntity(bson.M{"name": id}, ent)
	if e != "" {
		return nil, e
	}

	//Remove _id
	top = fixID(top)

	pid := (top["id"].(primitive.ObjectID)).Hex()

	var x map[string]interface{}
	var e1 string
	println("ANCS-LEN: ", len(ancestry))
	for i := range ancestry {
		for k, v := range ancestry[i] {

			println("KEY:", k, " VAL:", v)

			if v == "all" {
				println("K:", k)
				return GetManyEntities(k, bson.M{"parentId": pid}, nil)
			}

			x, e1 = GetEntity(bson.M{"parentId": pid, "name": v}, k)
			if e1 != "" {
				println("Failing here")
				println("E1: ", e1)
				return nil, ""
			}
			pid = (x["id"].(primitive.ObjectID)).Hex()
		}
	}

	return nil, ""
}

func GetEntityUsingTenantAsAncestor(ent, id string, ancestry []map[string]string) (map[string]interface{}, string) {
	top, e := GetEntity(bson.M{"name": id}, ent)
	if e != "" {
		return nil, e
	}

	pid := (top["id"].(primitive.ObjectID)).Hex()

	var x map[string]interface{}
	var e1 string
	for i := range ancestry {
		for k, v := range ancestry[i] {

			println("KEY:", k, " VAL:", v)

			x, e1 = GetEntity(bson.M{"parentId": pid, "name": v}, k)
			if e1 != "" {
				println("Failing here")
				return nil, ""
			}
			pid = (x["id"].(primitive.ObjectID)).Hex()
		}
	}

	return x, ""
}

func GetEntitiesOfAncestor(id interface{}, ent int, entStr, wantedEnt string) ([]map[string]interface{}, string) {
	var ans []map[string]interface{}
	var t map[string]interface{}
	var e, e1 string
	if ent == u.TENANT {

		t, e = GetEntity(bson.M{"name": id}, "tenant")
		if e != "" {
			return nil, e
		}

	} else {
		ID, _ := primitive.ObjectIDFromHex(id.(string))
		t, e = GetEntity(bson.M{"_id": ID}, entStr)
		if e != "" {
			return nil, e
		}
	}

	sub, e1 := GetManyEntities(u.EntityToString(ent+1),
		bson.M{"parentId": t["id"].(primitive.ObjectID).Hex()}, nil)
	if e1 != "" {
		return nil, e1
	}

	if wantedEnt == "" {
		wantedEnt = u.EntityToString(ent + 2)
	}

	for i := range sub {
		x, _ := GetManyEntities(wantedEnt,
			bson.M{"parentId": sub[i]["id"].(primitive.ObjectID).Hex()}, nil)
		ans = append(ans, x...)
	}
	return ans, ""
}

//DEV FAMILY FUNCS

func DeleteDeviceF(entityID primitive.ObjectID) (map[string]interface{}, string) {
	//var deviceType string

	t, e := GetEntityHierarchy(entityID, "device", 0, 999)
	if e != "" {
		return u.Message(false,
			"There was an error in deleting the entity"), "not found"
	}

	return deleteDeviceHelper(t)
}

func deleteDeviceHelper(t map[string]interface{}) (map[string]interface{}, string) {
	println("entered ddH")
	if t != nil {

		if v, ok := t["children"]; ok {
			if x, ok := v.([]map[string]interface{}); ok {
				for i := range x {
					deleteDeviceHelper(x[i])
				}
			}
		}

		ctx, cancel := u.Connect()
		//Delete relevant non hierarchal objects
		GetDB().Collection("sensor").DeleteMany(ctx,
			bson.M{"parentId": t["id"].(primitive.ObjectID).Hex()})

		GetDB().Collection("group").DeleteMany(ctx,
			bson.M{"parentId": t["id"].(primitive.ObjectID).Hex()})

		c, _ := GetDB().Collection("device").DeleteOne(ctx, bson.M{"_id": t["id"].(primitive.ObjectID)})
		if c.DeletedCount == 0 {
			return u.Message(false, "There was an error in deleting the entity"), "not found"
		}
		defer cancel()

	}
	return nil, ""
}

func ExtractCursor(c *mongo.Cursor, ctx context.Context) ([]map[string]interface{}, string) {
	ans := []map[string]interface{}{}
	for c.Next(ctx) {
		x := map[string]interface{}{}
		err := c.Decode(x)
		if err != nil {
			fmt.Println(err)
			return nil, err.Error()
		}
		//Remove _id
		x = fixID(x)
		ans = append(ans, x)
	}
	return ans, ""
}