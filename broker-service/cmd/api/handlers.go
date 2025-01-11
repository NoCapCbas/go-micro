package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"log"
	"net/http"
)

type RequestPayload struct {
	Action string      `json:"action"`
	Auth   AuthPayload `json:"auth,omitempty"`
	// Log    Log    `json:"log,omitempty"`
	// Mail   Mail   `json:"mail,omitempty"`
}

type AuthPayload struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (app *Config) Broker(w http.ResponseWriter, r *http.Request) {
	log.Println("Broker hit")
	payload := jsonResponse{
		Error:   false,
		Message: "Hit the broker",
	}

	_ = app.writeJSON(w, http.StatusOK, payload)
}

func (app *Config) HandleSubmission(w http.ResponseWriter, r *http.Request) {
	log.Println("HandleSubmission hit")
	var requestPayload RequestPayload

	err := app.readJSON(w, r, &requestPayload)
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	switch requestPayload.Action {
	case "auth":
		app.authenticate(w, requestPayload.Auth)
		return
	default:
		app.errorJSON(w, errors.New("unknown action"))
		return
	}
}

func (app *Config) authenticate(w http.ResponseWriter, a AuthPayload) {
	log.Println("Authenticating...")
	// create some json which we'll send to the auth microservice
	jsonData, _ := json.MarshalIndent(a, "", "\t")

	// call the service
	request, err := http.NewRequest("POST", "http://authentication-service/authenticate", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Println("Error creating request")
		app.errorJSON(w, err)
		return
	}

	// create a client
	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		log.Println("Error calling auth service")
		app.errorJSON(w, err)
		return
	}
	defer response.Body.Close()

	// make sure we get back the correct status code
	if response.StatusCode == http.StatusUnauthorized {
		log.Println("Invalid credentials")
		app.errorJSON(w, errors.New("invalid credentials"))
		return
	} else if response.StatusCode != http.StatusAccepted {
		log.Println("Error calling auth service")
		app.errorJSON(w, errors.New("error calling auth service"))
		return
	}

	// create a variable we'll read response.Body into
	var jsonFromService jsonResponse

	// decode the json from the auth service
	err = json.NewDecoder(response.Body).Decode(&jsonFromService)
	if err != nil {
		log.Println("Error decoding json from auth service")
		app.errorJSON(w, err)
		return
	}

	log.Println("jsonFromService", jsonFromService)

	if jsonFromService.Error {
		log.Println("Error in jsonFromService")
		app.errorJSON(w, errors.New(jsonFromService.Message), http.StatusUnauthorized)
		return
	}

	log.Println("jsonFromService", jsonFromService)

	var payload jsonResponse
	payload.Error = false
	payload.Message = "Authenticated!"
	payload.Data = jsonFromService.Data

	app.writeJSON(w, http.StatusAccepted, payload)
}
