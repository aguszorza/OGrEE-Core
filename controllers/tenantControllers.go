package controllers

import (
	"encoding/json"
	"net/http"
	"p3/models"
	u "p3/utils"
	"strconv"

	"github.com/gorilla/mux"
)

// swagger:operation POST /api/user/tenants tenants CreateTenant
// Creates a Tenant in the system.
// ---
// produces:
// - application/json
// parameters:
// - name: Name
//   in: query
//   description: Name of tenant
//   required: true
//   type: string
//   default: "Nissan"
// - name: Category
//   in: query
//   description: Category of Tenant (ex. Consumer Electronics, Medical)
//   required: true
//   type: string
//   default: "Auto"
// - name: Description
//   in: query
//   description: 'Description of the Tenant'
//   required: false
//   type: string[]
//   example: ["Worldwide automotive company"]
// - name: Domain
//   description: 'Domain of the Tenant'
//   required: true
//   type: string
//   default: "Auto"
// - name: Color
//   in: query
//   description: Color of Tenant (useful for 3D rendering)
//   required: true
//   type: string
//   default: "Silver"
// - name: MainContact
//   in: query
//   description: The main method of contact for Tenant
//   required: false
//   type: string
//   default: "Website"
// - name: MainPhone
//   in: query
//   description: Main Phone # of Tenant
//   required: false
//   type: string
//   default: "000"
// - name: MainEmail
//   in: query
//   description: Main Email Address of Tenant
//   required: false
//   type: string
//   default: "nissan@nissan.com"
// responses:
//     '201':
//         description: Created
//     '400':
//         description: Bad request

var CreateTenant = func(w http.ResponseWriter, r *http.Request) {
	tenant := &models.Tenant{}

	err := json.NewDecoder(r.Body).Decode(tenant)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		u.Respond(w, u.Message(false, "Error while decoding request body"))
		u.ErrLog("Error while decoding request body", "CREATE TENANT", "")

		return
	}

	resp, e := tenant.Create()

	switch e {
	case "validate":
		w.WriteHeader(http.StatusBadRequest)
		u.ErrLog("Error while creating tenant", "CREATE TENANT", e)
	case "":
		w.WriteHeader(http.StatusCreated)
		u.ErrLog("Error while creating tenant", "CREATE TENANT", e)
	default:
		w.WriteHeader(http.StatusInternalServerError)
		u.ErrLog("Error while creating tenant", "CREATE TENANT", e)
	}

	u.Respond(w, resp)
}

// swagger:operation GET /api/user/tenants/{id} tenants GetTenant
// Gets a Tenant(s) from the system.
// The ID must be provided in the URL parameter
// ---
// produces:
// - application/json
// parameters:
// - name: ID
//   in: path
//   description: ID of desired tenant
//   required: true
//   type: int
//   default: 999
// responses:
//     '200':
//         description: Found
//     '400':
//         description: Bad request
//     '404':
//         description: Not Found
var GetTenantFor = func(w http.ResponseWriter, r *http.Request) {
	var resp map[string]interface{}
	id, err := strconv.Atoi(mux.Vars(r)["id"])

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		u.Respond(w, u.Message(false, "Error while extracting from path parameters"))
		u.ErrLog("Error while extracting from path parameters", "GET TENANT", "")

	}

	data, e := models.GetTenant(uint(id))

	if e != "" {
		resp = u.Message(false, "Error: "+e)
		u.ErrLog("Error while getting tenant", "GET TENANT", e)

		switch e {
		case "record not found":
			w.WriteHeader(http.StatusNotFound)
		default:
		}

	} else {
		resp = u.Message(true, "success")
	}

	resp["data"] = data
	u.Respond(w, resp)
}

// swagger:operation GET /api/user/tenants tenants GetAllTenants
// Obtain all tenants in the system.
// Returns JSON body with all tenants and their IDs
// ---
// responses:
//     '200':
//         description: Found
//     '404':
//         description: Nothing Found
var GetAllTenants = func(w http.ResponseWriter, r *http.Request) {

	data, e := models.GetAllTenants()
	resp := u.Message(true, "success")

	if len(data) == 0 {
		resp = u.Message(false, "Error: "+e)
		u.ErrLog("Error while getting tenants", "GET ALL TENANTS", e)

		switch e {
		case "validate":

		case "":
			w.WriteHeader(http.StatusNotFound)
			//
		default:
		}

	} else {
		resp = u.Message(true, "success")
	}

	resp["data"] = data
	u.Respond(w, resp)
}

// swagger:operation PUT /api/user/tenants/{id} tenants UpdateTenant
// Changes Tenant data in the system.
// If no new or any information is provided
// an OK will still be returned
// ---
// produces:
// - application/json
// parameters:
// - name: ID
//   in: path
//   description: ID of desired tenant
//   required: true
//   type: int
//   default: 999
// - name: Name
//   in: query
//   description: Name of tenant
//   required: false
//   type: string
//   default: "INFINITI"
// - name: Category
//   in: query
//   description: Category of Tenant (ex. Consumer Electronics, Medical)
//   required: false
//   type: string
//   default: "Auto"
// - name: Description
//   in: query
//   description: Description of Tenant
//   required: false
//   type: string[]
//   default: "High End Worldwide automotive company"
// - name: Domain
//   description: 'Domain of the Tenant'
//   required: false
//   type: string
//   default: "High End Auto"
// - name: Color
//   in: query
//   description: Color of Tenant (useful for 3D rendering)
//   required: false
//   type: string
//   default: "Black"
// - name: MainContact
//   in: query
//   description: The main method of contact for Tenant
//   required: false
//   type: string
//   default: "Post"
// - name: MainPhone
//   in: query
//   description: Main Phone # of Tenant
//   required: false
//   type: string
//   default: "999"
// - name: MainEmail
//   in: query
//   description: Main Email Address of Tenant
//   required: false
//   type: string
//   default: "infiniti@nissan.com"
// responses:
//     '200':
//         description: Updated
//     '400':
//         description: Bad request
//     '404':
//         description: Not Found

var UpdateTenant = func(w http.ResponseWriter, r *http.Request) {

	id, e := strconv.Atoi(mux.Vars(r)["id"])
	if e != nil {
		w.WriteHeader(http.StatusBadRequest)
		u.Respond(w, u.Message(false, "Error while extracting from path parameters"))
		u.ErrLog("Error while extracting from path parameters", "UPDATE TENANT", "")
	}
	tenant := &models.Tenant{}

	err := json.NewDecoder(r.Body).Decode(tenant)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		u.Respond(w, u.Message(false, "Error while decoding request body"))
		u.ErrLog("Error while decoding request body", "UPDATE TENANT", "")
	}

	v, e1 := models.UpdateTenant(uint(id), tenant)

	switch e1 {

	case "validate":
		w.WriteHeader(http.StatusBadRequest)
		u.ErrLog("Error while updating tenant", "UPDATE TENANT", e1)
	case "internal":
		w.WriteHeader(http.StatusInternalServerError)
		u.ErrLog("Error while updating tenant", "UPDATE TENANT", e1)
	case "record not found":
		w.WriteHeader(http.StatusNotFound)
		u.ErrLog("Error while updating tenant", "UPDATE TENANT", e1)
	default:
	}

	u.Respond(w, v)
}

//This delete function is for 1 tenant 1 user
/*var DeleteTenant = func(w http.ResponseWriter, r *http.Request) {
	id := r.Context().Value("user").(uint)
	v := models.DeleteTenant(id)
	u.Respond(w, v)
}*/

// swagger:operation DELETE /api/user/tenants/{id} tenants DeleteTenant
// Deletes a Tenant in the system.
// ---
// produces:
// - application/json
// parameters:
// - name: ID
//   in: path
//   description: ID of desired tenant
//   required: true
//   type: int
//   default: 999
// responses:
//     '204':
//        description: Successful
//     '400':
//        description: Not found

var DeleteTenant = func(w http.ResponseWriter, r *http.Request) {

	id, e := strconv.Atoi(mux.Vars(r)["id"])
	if e != nil {
		u.Respond(w, u.Message(false, "Error while extracting from path parameters"))
		u.ErrLog("Error while extracting from path parameters", "DELETE TENANT", "")
	}

	v := models.DeleteTenant(uint(id))
	if v["status"] == false {
		w.WriteHeader(http.StatusNotFound)
		u.ErrLog("Not Found", "DELETE TENANT", "")

	} else {
		w.WriteHeader(http.StatusNoContent)
	}
	u.Respond(w, v)
}
