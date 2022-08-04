package utils

//Builds json messages and
//returns json response

//Also holds other commonly used utility functions

import (
	"encoding/json"
	"net/http"
	"net/url"
)

func Message(status bool, message string) map[string]interface{} {
	return map[string]interface{}{"status": status, "message": message}
}

func Respond(w http.ResponseWriter, data map[string]interface{}) {
	w.Header().Add("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func ParamsParse(link *url.URL) []byte {
	q, _ := url.ParseQuery(link.RawQuery)
	values := make(map[string]string)
	for key, _ := range q {
		values[key] = q.Get(key)
	}

	//If you marshal it then
	//Unmarshal it, you can parse
	//the URL into a struct of choice!
	//Note that you would have to
	//Unmarshal twice to catch the
	//inner struct
	js, err := json.Marshal(values)
	if err != nil {
		panic(err)
	}

	return js

	/*
		mydata := &models.Tenant{}
		json.Unmarshal(query, mydata)
		json.Unmarshal(query, &(mydata.Attributes))
	*/
	//return values
}
