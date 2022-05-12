package models

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"time"
)

//Function helps with API Requests
func Send(method, URL, key string, data map[string]interface{}) (*http.Response,
	error) {
	//Loop because sometimes a
	//Stream Error occurs
	//thus give max 200 attempts before returning error
	sender := func(method, URL, key string, data map[string]interface{}) (*http.Response, error) {
		client := &http.Client{}
		dataJSON, _ := json.Marshal(data)

		req, _ := http.NewRequest(method, URL, bytes.NewBuffer(dataJSON))
		req.Header.Set("Authorization", "Bearer "+key)
		return client.Do(req)
	}

	for i := 0; ; i++ {
		r, e := sender(method, URL, key, data)
		if e == nil {
			return r, e
		}

		if i == 200 {
			return r, e
		}
	}

}

//Function communicates with Unity
func ContactUnity(method, URL string, data map[string]interface{}) error {
	dataJSON, _ := json.Marshal(data)

	// Connect to a server
	//println(URL)
	conn, e := net.Dial("tcp", URL)
	if e != nil {
		return e
	}

	for {
		_, q := conn.Write(dataJSON)
		if q != nil {
			return q
		}
		defer conn.Close()

		time.Sleep(time.Duration(1) * time.Second)
		reply := make([]byte, 1024)
		_, err := conn.Read(reply)
		if err != nil {
			return err
		}
		fmt.Printf("received from server: [%s]\n", string(reply))
		return nil
	}

}
