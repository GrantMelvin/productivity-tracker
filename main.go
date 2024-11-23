package main

import (
	"fmt"
	"html/template"
	"net/http"
	"os"

	"github.com/joho/godotenv"
)

// Components of struct have to be capital for templating to work
type Test struct {
	Test string
}

func home(w http.ResponseWriter, r *http.Request) {
	var fileName = "home.html"

	t, err := template.ParseFiles(fileName)
	if err != nil {
		fmt.Println(err)
	}

	t.ExecuteTemplate(w, fileName, Test{"Grant"})
}

func handler(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/Home":
		home(w, r)
	// case "/login-submit":
	// 	loginSubmit(w, r)
	default:
		fmt.Fprintf(w, "Hello")
	}
}

func main() {
	godotenv.Load()

	server, exists := os.LookupEnv("SERVER")
	if exists {
		fmt.Println("Server start at:", server)
	} else {
		fmt.Println("Server not found")
		return
	}

	http.HandleFunc("/", handler)
	http.ListenAndServe(server, nil)
}
