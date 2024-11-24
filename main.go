package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/joho/godotenv"
	"gopkg.in/yaml.v3"
)

const SIZE = 500
const DATA_DIR = "./assets/data"

var logs Logs
var maxID int

// Components of struct have to be capital for templating to work
type Test struct {
	Test string
}

type Log struct {
	Id           int                 `yaml:"id"`
	Date         string              `yaml:"date"`
	Goals        []map[string]string `yaml:"goals"`
	Productivity []map[string]string `yaml:"productivity"`
	Notes        []map[string]string `yaml:"notes"`
}

type Logs struct {
	Logs []Log
}

type SearchableLogs struct {
	Logs   []Log
	Search bool
}

func parseKeyValueArray(data []interface{}) []map[string]string {
	result := []map[string]string{}

	for _, item := range data {
		if str, ok := item.(string); ok {
			result = append(result, map[string]string{"": str})
		} else if mapItem, ok := item.(map[string]interface{}); ok {
			parsed := make(map[string]string)
			for k, v := range mapItem {
				if strValue, ok := v.(string); ok {
					parsed[k] = strValue
				}
			}
			result = append(result, parsed)
		}
	}

	return result
}

func parseLogs(data []byte) (*Log, error) {
	var raw map[string]interface{}
	err := yaml.Unmarshal(data, &raw)
	if err != nil {
		return nil, err
	}

	Log := &Log{}

	// Set the id field
	if id, ok := raw["id"].(int); ok {
		Log.Id = id
	}

	// Set the date field
	if date, ok := raw["date"].(string); ok {
		Log.Date = date
	}

	// Parse and set Goals
	if goals, ok := raw["goals"].([]interface{}); ok {
		Log.Goals = parseKeyValueArray(goals)
	}

	// Parse and set Productivity
	if productivity, ok := raw["productivity"].([]interface{}); ok {
		Log.Productivity = parseKeyValueArray(productivity)
	}

	// Parse and set Notes
	if notes, ok := raw["notes"].([]interface{}); ok {
		Log.Notes = parseKeyValueArray(notes)
	}

	return Log, nil
}

func getLogs(root_path string) Logs {
	entries, err := os.ReadDir(root_path)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Total file count:", len(entries))

	Logs_slice := Logs{
		Logs: make([]Log, 0, SIZE), // Use zero-length slice with a capacity
	}

	for _, e := range entries {
		filename := root_path + "/" + e.Name()
		fmt.Println("Currently evaluating:", filename)

		current_file, err := os.ReadFile(filename)
		if err != nil {
			fmt.Println(err)
			continue
		}

		Log, err := parseLogs(current_file)
		if err != nil {
			fmt.Printf("Failed to parse file %s: %v\n", filename, err)
			continue
		}

		Logs_slice.Logs = append(Logs_slice.Logs, *Log)

		// Update maxID if necessary
		if Log.Id > maxID {
			maxID = Log.Id
		}
	}

	fmt.Printf("maxID: %d\n", maxID)
	return Logs_slice
}

func containsString(log Log, searchString string) bool {
	// Easy check for data field
	if strings.Contains(log.Date, searchString) {
		return true
	}

	// Checks all of the mappings for the log we are evaluating
	checkInSliceOfMaps := func(slice []map[string]string) bool {
		// Checks each mapping
		for _, m := range slice {

			// Checks each key and value
			for key, value := range m {
				if strings.Contains(key, searchString) {
					return true
				}
				if strings.Contains(value, searchString) {
					return true
				}
			}
		}
		return false
	}

	// Check Goals, Productivity, and Notes fields
	if checkInSliceOfMaps(log.Goals) || checkInSliceOfMaps(log.Productivity) || checkInSliceOfMaps(log.Notes) {
		return true
	}

	// If not found in any field
	return false
}

func writeLog(log Log) {
	var fileName = DATA_DIR + "/" + log.Date + ".yaml"
	file, err := os.OpenFile(fileName, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		fmt.Printf("error opening/creating file: %v\n", err)
	}
	fmt.Println("file:", file)
	fmt.Println("log:", log)

	defer file.Close()
	enc := yaml.NewEncoder(file)

	err = enc.Encode(log)

	if err != nil {
		fmt.Printf("error encoding: %v", err)
	}
}

func searchLogs(keyword string) Logs {
	var log_subset Logs
	// fmt.Println(logs.Logs[0])
	log_subset.Logs = []Log{}
	for i, log := range logs.Logs {
		log = logs.Logs[i]
		fmt.Printf("Log %v: %v\n", i+1, log)

		if containsString(log, keyword) {
			log_subset.Logs = append(log_subset.Logs, log)
		}
	}
	fmt.Println(log_subset)
	return log_subset
}

func home(w http.ResponseWriter) {
	var home = "home.html"

	tmpl, err := template.ParseFiles(home)
	if err != nil {
		fmt.Println(err)
	}

	tmpl.ExecuteTemplate(w, home, SearchableLogs{Logs: logs.Logs, Search: false})
}

func search(w http.ResponseWriter, r *http.Request) {
	var search = "home.html"

	tmpl, err := template.ParseFiles(search)
	if err != nil {
		fmt.Println(err)
	}

	keyword := r.FormValue("keyword")
	fmt.Println("keyword:", keyword)

	// If they didnt search anything
	// if keyword == "" {

	// }

	var subset = searchLogs(keyword)
	fmt.Println("subet:", subset)

	tmpl.ExecuteTemplate(w, search, SearchableLogs{Logs: subset.Logs, Search: true})
}

func add(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost { // Ensure the form was submitted as POST

		// Parse and add goals, productivity, and notes from form values
		var newLog Log

		newLog.Id = maxID + 1
		maxID = maxID + 1
		fmt.Println(maxID)

		newLog.Date = r.FormValue("date")
		// Parse goals
		goalsKeys := r.Form["goals_key[]"]
		goalsValues := r.Form["goals_value[]"]
		for i := range goalsKeys {
			newLog.Goals = append(newLog.Goals, map[string]string{goalsKeys[i]: goalsValues[i]})
		}

		// Parse productivity
		productivityKeys := r.Form["productivity_key[]"]
		productivityValues := r.Form["productivity_value[]"]
		for i := range productivityKeys {
			newLog.Productivity = append(newLog.Productivity, map[string]string{productivityKeys[i]: productivityValues[i]})
		}

		// Parse notes
		notesKeys := r.Form["notes_key[]"]
		notesValues := r.Form["notes_value[]"]
		for i := range notesKeys {
			newLog.Notes = append(newLog.Notes, map[string]string{notesKeys[i]: notesValues[i]})
		}

		// Add the new log to the logs slice
		logs.Logs = append(logs.Logs, newLog)

		writeLog(newLog)

		// Redirect to the home page
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	// Render the add page for GET requests
	var add = "add.html"
	tmpl, err := template.ParseFiles(add)
	if err != nil {
		fmt.Println(err)
	}

	tmpl.ExecuteTemplate(w, add, nil)
}

func handler(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/":
		home(w)
	case "/search-logs":
		search(w, r)
	case "/add-logs":
		add(w, r)
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
	logs = getLogs(DATA_DIR)
	fmt.Println(maxID)
	fs := http.FileServer(http.Dir("assets"))
	http.Handle("/assets/", http.StripPrefix("/assets", fs))
	http.HandleFunc("/", handler)
	http.ListenAndServe(server, nil)
}
