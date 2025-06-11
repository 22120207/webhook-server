package main

import (
	"net/http"
)

type handler func(w http.ResponseWriter, r *http.Request)
