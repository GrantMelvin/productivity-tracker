package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"gopkg.in/yaml.v2"
)

const SIZE = 500

var days Days

// Components of struct have to be capital for templating to work
type Test struct {
	Test string
}

type Day struct {
	Date         string              `yaml:"date"`
	Goals        []map[string]string `yaml:"goals"`
	Productivity []map[string]string `yaml:"productivity"`
	Notes        []map[string]string `yaml:"notes"`
}

type Days struct {
	Days []Day
}

func getLogs(root_path string) Days {

	// Gets all subfolders of the data directory
	entries, err := os.ReadDir(root_path)
	if err != nil {
		log.Fatal(err)
	}

	var total_file_count int = (len(entries))
	fmt.Println("total file count:", total_file_count)

	// Slice starting from 0 to the length of what we need
	days_slice := Days{
		Days: make([]Day, total_file_count, SIZE),
	}

	for i, e := range entries {
		filename := (root_path + "/" + e.Name())
		fmt.Println("Currently evaluating:", filename)

		var day Day
		current_file, err := os.ReadFile(filename)
		if err != nil {
			fmt.Println(err)
		}

		err = yaml.Unmarshal(current_file, &day)
		if err != nil {
			log.Fatalf("Unmarshal: %v", err)
		}

		days_slice.Days[i] = day
	}

	return days_slice

}

func home(w http.ResponseWriter, days Days) {

	var home = "home.html"

	t, err := template.ParseFiles(home)
	if err != nil {
		fmt.Println(err)
	}

	fmt.Println(days.Days[0].Goals[0])

	t.ExecuteTemplate(w, home, Days{days.Days})
}

func handler(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/":
		home(w, days)
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

	// retrieve the logs before we do anything
	days = getLogs("./data")

	http.HandleFunc("/", handler)
	http.ListenAndServe(server, nil)
}
