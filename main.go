package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
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

type SearchableLogsReport struct {
	Logs   []Log
	Search bool
	Report string
}

func parseKeyVals(data []interface{}) []map[string]string {
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

func findLogByID(id int) (*Log, bool) {
	for _, log := range logs.Logs {
		if log.Id == id {
			return &log, true
		}
	}
	return nil, false
}

func generateScrumReport(log Log) string {

	content_message := fmt.Sprintf("%v", log)

	postBody, _ := json.Marshal(map[string]interface{}{
		"model": "gpt-4o-mini",
		"messages": []map[string]string{
			{
				"role":    "system",
				"content": "You are going to parse the daily log of what I have done and create a report that I can present to my engineering team to show my progress for that day.",
			},
			{
				"role":    "user",
				"content": content_message,
			},
		},
	})

	responseBody := bytes.NewBuffer(postBody)

	key, _ := os.LookupEnv("OPENAI_API_KEY")

	bearer_token := fmt.Sprintf("Bearer %s", key)

	resp, err := http.NewRequest(http.MethodPost, "https://api.openai.com/v1/chat/completions", responseBody)
	resp.Header.Add("Content-Type", "application/json")
	resp.Header.Add("Authorization", bearer_token)

	// Handle Error
	if err != nil {
		fmt.Printf("An Error Occured %v", err)
	}

	response, err := http.DefaultClient.Do(resp)

	// Handle Error
	if err != nil {
		fmt.Printf("An Error Occured %v", err)
	}

	defer response.Body.Close()

	// Read the response body
	body, err := io.ReadAll(response.Body)
	if err != nil {
		fmt.Println(err)
	}
	// Define the structure for parsing
	var test struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	err = json.Unmarshal(body, &test)
	if err != nil {
		fmt.Println("Error unmarshaling JSON:", err)
	}
	// fmt.Println(test.Choices[0].Message.Content)

	return test.Choices[0].Message.Content
}

func setCurrentLogs(data []byte) (*Log, error) {
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
		Log.Goals = parseKeyVals(goals)
	}

	// Parse and set Productivity
	if productivity, ok := raw["productivity"].([]interface{}); ok {
		Log.Productivity = parseKeyVals(productivity)
	}

	// Parse and set Notes
	if notes, ok := raw["notes"].([]interface{}); ok {
		Log.Notes = parseKeyVals(notes)
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

		Log, err := setCurrentLogs(current_file)
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

func parseLog(log Log, searchString string) bool {
	// check date field for string
	if strings.Contains(log.Date, searchString) {
		return true
	}

	// check all the other mappings for the string
	checkInSliceOfMaps := func(slice []map[string]string) bool {
		// Checks each mapping
		for _, m := range slice {

			// check each key and val
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

func searchLogs(keyword string) Logs {
	var log_subset Logs
	// fmt.Println(logs.Logs[0])
	log_subset.Logs = []Log{}
	for i, log := range logs.Logs {
		log = logs.Logs[i]
		fmt.Printf("Log %v: %v\n", i+1, log)

		if parseLog(log, keyword) {
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

		// makes new maxID
		newLog.Id = maxID + 1
		maxID = maxID + 1

		// parse Each field from the html form
		newLog.Date = r.FormValue("date")

		goalsKeys := r.Form["goals_key[]"]
		goalsValues := r.Form["goals_value[]"]
		for i := range goalsKeys {
			newLog.Goals = append(newLog.Goals, map[string]string{goalsKeys[i]: goalsValues[i]})
		}

		productivityKeys := r.Form["productivity_key[]"]
		productivityValues := r.Form["productivity_value[]"]
		for i := range productivityKeys {
			newLog.Productivity = append(newLog.Productivity, map[string]string{productivityKeys[i]: productivityValues[i]})
		}

		notesKeys := r.Form["notes_key[]"]
		notesValues := r.Form["notes_value[]"]
		for i := range notesKeys {
			newLog.Notes = append(newLog.Notes, map[string]string{notesKeys[i]: notesValues[i]})
		}

		// Add the new log to the logs slice
		logs.Logs = append(logs.Logs, newLog)

		var fileName = DATA_DIR + "/" + newLog.Date + ".yaml"
		file, err := os.OpenFile(fileName, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
		if err != nil {
			fmt.Printf("error opening/creating file: %v\n", err)
		}
		fmt.Println("New File:", file)
		fmt.Println("New Log:", newLog)

		// fmt.Println(generateScrumReport(newLog))

		defer file.Close()
		enc := yaml.NewEncoder(file)

		err = enc.Encode(newLog)

		if err != nil {
			fmt.Printf("error encoding: %v", err)
		}

		// Go back to home page after creating a new log
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

func delete(w http.ResponseWriter, r *http.Request) {
	targetID, err := strconv.Atoi(r.URL.Query().Get("id"))
	if err != nil {
		fmt.Println(err)
	}

	targetLog, found := findLogByID(targetID)

	fmt.Println("Target ID:", targetID)
	fmt.Println("Target Log:", targetLog)

	if !found {
		fmt.Println("Log not found. We are in trouble.")
	}

	for i, log := range logs.Logs {
		if log.Id == targetID {
			fmt.Println("Removing:", log)
			// Remove the log by slicing out the element at index i
			logs.Logs = append(logs.Logs[:i], logs.Logs[i+1:]...)
		}
	}

	var filename = "./assets/data/" + targetLog.Date + ".yaml"
	fmt.Println("Filename to delete:", filename)

	os.Remove(filename)

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func edit(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		targetID, err := strconv.Atoi(r.URL.Query().Get("id"))
		if err != nil {
			fmt.Println(err)
		}

		targetLog, found := findLogByID(targetID)

		fmt.Println("Target ID:", targetID)
		fmt.Println("Target Log:", targetLog)

		if !found {
			fmt.Println("Log not found. We are in trouble.")
		}

		var path = "edit.html"
		tmpl, err := template.ParseFiles(path)
		if err != nil {
			fmt.Println(err)
		}

		fmt.Println(targetLog)

		tmpl.ExecuteTemplate(w, path, targetLog)
	}

	if r.Method == http.MethodPost {
		targetID, err := strconv.Atoi(r.URL.Query().Get("id"))
		if err != nil {
			fmt.Println(err)
		}

		targetLog, found := findLogByID(targetID)

		fmt.Println("Target ID:", targetID)
		fmt.Println("Target Log:", targetLog)

		if !found {
			fmt.Println("Log not found. We are in trouble.")
		}

		var path = "edit.html"
		tmpl, err := template.ParseFiles(path)
		if err != nil {
			fmt.Println(err)
		}

		tmpl.ExecuteTemplate(w, path, nil)
	}

}

func generate(w http.ResponseWriter, r *http.Request) {
	targetID, err := strconv.Atoi(r.URL.Query().Get("id"))
	if err != nil {
		fmt.Println(err)
	}

	targetLog, found := findLogByID(targetID)

	fmt.Println("Target ID:", targetID)
	fmt.Println("Target Log:", targetLog)

	if !found {
		fmt.Println("Log not found. We are in trouble.")
		var path = "home.html"
		tmpl, err := template.ParseFiles(path)
		if err != nil {
			fmt.Println(err)
		}

		tmpl.ExecuteTemplate(w, path, SearchableLogs{Logs: logs.Logs, Search: false})
		return
	}

	report := (generateScrumReport(*targetLog))

	var path = "home.html"
	tmpl, err := template.ParseFiles(path)
	if err != nil {
		fmt.Println(err)
	}

	tmpl.ExecuteTemplate(w, path, SearchableLogsReport{Logs: logs.Logs, Search: false, Report: report})

}

func handler(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/":
		home(w)
	case "/search-logs":
		search(w, r)
	case "/add-logs":
		add(w, r)
	case "/delete-logs":
		delete(w, r)
	case "/edit-logs":
		edit(w, r)
	case "/generate-report":
		generate(w, r)
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
		fmt.Println("Server not started at:", server)
		fmt.Println("Server not found")
	}

	// retrieve the current logs before we do anything
	logs = getLogs(DATA_DIR)

	// Strip assets prefix so that we can serve the css
	fs := http.FileServer(http.Dir("assets"))
	http.Handle("/assets/", http.StripPrefix("/assets", fs))

	http.HandleFunc("/", handler)
	http.ListenAndServe(server, nil)
}
