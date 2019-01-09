package main

import (
	"encoding/json"
	"net/http"
	"strconv"
)

const (
	errorUnauthorized  = "unauthorized"
	errorBadMethod     = "bad method"
	errorUnknownMethod = "unknown method"
)

func ResponseWrite(w http.ResponseWriter, responseCode int, errorMessage string, actionResult interface{}) {
	result := make(map[string]interface{})
	result["error"] = errorMessage
	if actionResult != nil {
		result["response"] = actionResult
	}
	response, _ := json.Marshal(result)
	w.WriteHeader(responseCode)
	w.Write(response)
}

// MyApi Model Handler

func (structure *MyApi) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/user/profile":
		structure.MyApiProfile(w, r)
	case "/user/create":
		if r.Header.Get("X-Auth") != "100500" {
			ResponseWrite(w, http.StatusForbidden, errorUnauthorized, nil)
			return
		}
		if r.Method != "POST" {
			ResponseWrite(w, http.StatusNotAcceptable, errorBadMethod, nil)
			return
		}
		structure.MyApiCreate(w, r)
	default:
		ResponseWrite(w, http.StatusNotFound, errorUnknownMethod, nil)
	}
}

// OtherApi Model Handler

func (structure *OtherApi) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/user/create":
		if r.Header.Get("X-Auth") != "100500" {
			ResponseWrite(w, http.StatusForbidden, errorUnauthorized, nil)
			return
		}
		if r.Method != "POST" {
			ResponseWrite(w, http.StatusNotAcceptable, errorBadMethod, nil)
			return
		}
		structure.OtherApiCreate(w, r)
	default:
		ResponseWrite(w, http.StatusNotFound, errorUnknownMethod, nil)
	}
}

func (structure *MyApi) MyApiProfile(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	login := r.Form.Get("login")
	if login == "" {
		ResponseWrite(w, http.StatusBadRequest, "login must be not empty", nil)
		return
	}

	// action
	actionResult, actionError := structure.Profile(r.Context(), ProfileParams{
		Login: login,
	})

	// error handle
	if actionError != nil {
		ResponseWrite(w, actionError.HTTPStatus, actionError.Error(), nil)
		return
	}

	// response
	ResponseWrite(w, http.StatusOK, "", actionResult)
}

func (structure *MyApi) MyApiCreate(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	login := r.Form.Get("login")
	if login == "" {
		ResponseWrite(w, http.StatusBadRequest, "login must be not empty", nil)
		return
	}
	if len(login) < 10 {
		ResponseWrite(w, http.StatusBadRequest, "login len must be >= 10", nil)
		return
	}

	name := r.Form.Get("full_name")

	status := r.Form.Get("status")
	if status == "" {
		status = "user"
	}
	if status != "user" && status != "moderator" && status != "admin" {
		ResponseWrite(w, http.StatusBadRequest, "status must be one of [user, moderator, admin]", nil)
		return
	}

	age, err := strconv.Atoi(r.Form.Get("age"))
	if err != nil {
		ResponseWrite(w, http.StatusBadRequest, "age must be int", nil)
		return
	}
	if age < 0 {
		ResponseWrite(w, http.StatusBadRequest, "age must be >= 0", nil)
		return
	}
	if age > 128 {
		ResponseWrite(w, http.StatusBadRequest, "age must be <= 128", nil)
		return
	}

	// action
	actionResult, actionError := structure.Create(r.Context(), CreateParams{
		Login:  login,
		Name:   name,
		Status: status,
		Age:    age,
	})

	// error handle
	if actionError != nil {
		ResponseWrite(w, actionError.HTTPStatus, actionError.Error(), nil)
		return
	}

	// response
	ResponseWrite(w, http.StatusOK, "", actionResult)
}

func (structure *OtherApi) OtherApiCreate(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	username := r.Form.Get("username")
	if username == "" {
		ResponseWrite(w, http.StatusBadRequest, "username must be not empty", nil)
		return
	}
	if len(username) < 3 {
		ResponseWrite(w, http.StatusBadRequest, "username len must be >= 3", nil)
		return
	}

	name := r.Form.Get("account_name")

	class := r.Form.Get("class")
	if class == "" {
		class = "warrior"
	}
	if class != "warrior" && class != "sorcerer" && class != "rouge" {
		ResponseWrite(w, http.StatusBadRequest, "class must be one of [warrior, sorcerer, rouge]", nil)
		return
	}

	level, err := strconv.Atoi(r.Form.Get("level"))
	if err != nil {
		ResponseWrite(w, http.StatusBadRequest, "level must be int", nil)
		return
	}
	if level < 1 {
		ResponseWrite(w, http.StatusBadRequest, "level must be >= 1", nil)
		return
	}
	if level > 50 {
		ResponseWrite(w, http.StatusBadRequest, "level must be <= 50", nil)
		return
	}

	// action
	actionResult, actionError := structure.Create(r.Context(), OtherCreateParams{
		Username: username,
		Name:     name,
		Class:    class,
		Level:    level,
	})

	// error handle
	if actionError != nil {
		ResponseWrite(w, actionError.HTTPStatus, actionError.Error(), nil)
		return
	}

	// response
	ResponseWrite(w, http.StatusOK, "", actionResult)
}
