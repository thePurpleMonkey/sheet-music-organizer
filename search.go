package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/lib/pq"
)

// SearchResult is a struct that models a result from a search
type SearchResult struct {
	SongID   int64  `json:"song_id" db:"song_id"`
	SongName string `json:"song_name" db:"song_name"`
}

// AdvancedSearchRequest models the POST request body of an advanced search request
type AdvancedSearchRequest struct {
	CollectionID int64      `json:"collection_id"`
	Tags         []int64    `json:"tags"`
	Before       *time.Time `json:"before"`
	After        *time.Time `json:"after"`
	Include      []string   `json:"include"`
	Exclude      []string   `json:"exclude"`
}

// SearchHandler handles performing a search in a collection
func SearchHandler(w http.ResponseWriter, r *http.Request) {
	session, err := store.Get(r, "session")
	if err != nil {
		log.Printf("Search handler - Unable to get session: %v\n", err)
		SendError(w, SERVER_ERROR_MESSAGE, http.StatusInternalServerError)
		return
	}

	if r.Method == "GET" {
		var rawQuery string
		if rawQuery, err = url.QueryUnescape(r.URL.Query().Get("query")); err != nil {
			log.Printf("Search GET - Unable to parse query: %v\n", err)
			SendError(w, URL_ERROR_MESSAGE, http.StatusBadRequest)
			return
		}

		log.Printf("Raw query: '%v'\n", rawQuery)
		var reg *regexp.Regexp
		if reg, err = regexp.Compile("[^a-zA-Z0-9 ']+"); err != nil {
			log.Printf("Search GET - Unable to compile search RegEx: %v\n", err)
			SendError(w, SERVER_ERROR_MESSAGE, http.StatusInternalServerError)
			return
		}
		sanitizedQuery := reg.ReplaceAllString(rawQuery, "")
		query := strings.Join(strings.Fields(sanitizedQuery), " & ")
		log.Printf("Processed query: '%v'\n", query)

		// Get URL parameter
		collectionID, err := strconv.ParseInt(mux.Vars(r)["collection_id"], 10, 64)
		if err != nil {
			log.Printf("Search GET - Unable to parse collection id: %v\n", err)
			SendError(w, URL_ERROR_MESSAGE, http.StatusBadRequest)
			return
		}

		// Call the database function
		log.Printf("Search GET - Calling database function collection_search with parameters collection_id = %v and query = %v\n", collectionID, query)
		rows, err := db.Query("SELECT * FROM search_collection($1, $2)", collectionID, query)
		if err != nil {
			log.Printf("Search GET - Unable to retrieve search results from database for user %d in collection %d: %v\n", session.Values["user_id"], collectionID, err)
			SendError(w, DATABASE_ERROR_MESSAGE, http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		// Retrieve rows from database
		results := make([]SearchResult, 0)
		for rows.Next() {
			var result SearchResult
			if err := rows.Scan(&result.SongID, &result.SongName); err != nil {
				log.Printf("Search GET - Unable to retrieve row from database result: %v\n", err)
			}
			results = append(results, result)
		}

		// Check for errors from iterating over rows.
		if err := rows.Err(); err != nil {
			log.Printf("Search GET - Error retrieving search results from database result: %v\n", err)
			SendError(w, DATABASE_ERROR_MESSAGE, http.StatusInternalServerError)
			return
		}

		// Send response
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(results)
		return
	} else if r.Method == "POST" {
		var search AdvancedSearchRequest
		err := json.NewDecoder(r.Body).Decode(&search)
		if err != nil {
			// If there is something wrong with the request body, return a 400 status
			log.Printf("Search POST - Unable to parse request body: %v\n", err)
			body, _ := ioutil.ReadAll(r.Body)
			log.Printf("Body: %s\n", body)
			SendError(w, REQUEST_ERROR_MESSAGE, http.StatusBadRequest)
			return
		}

		// Verify user is authorized to access this collection
		var userID = session.Values["user_id"].(int64)
		var authorized = false
		acceptableIDs, err := getAuthorizedCollectionIDs(userID)
		if err != nil {
			SendError(w, DATABASE_ERROR_MESSAGE, http.StatusInternalServerError)
			return
		}
		for _, id := range acceptableIDs {
			if search.CollectionID == id {
				authorized = true
				break
			}
		}
		if !authorized {
			log.Printf("%v | %v not found in authorized collection IDs: %v", session.Values["email"], search.CollectionID, acceptableIDs)
			SendError(w, "Forbidden", http.StatusForbidden)
			return
		}

		var reg *regexp.Regexp
		if reg, err = regexp.Compile("[^a-zA-Z0-9 ']+"); err != nil {
			log.Printf("Search POST - Unable to compile search RegEx: %v\n", err)
			SendError(w, SERVER_ERROR_MESSAGE, http.StatusInternalServerError)
			return
		}

		var includedKeywords, excludedKeywords []string
		var sanitizedKeyword string

		for _, keyword := range search.Include {
			sanitizedKeyword = reg.ReplaceAllString(keyword, "")
			if len(sanitizedKeyword) > 0 {
				includedKeywords = append(includedKeywords, sanitizedKeyword)
			}
			// log.Printf("Keyword: '%v' -> Sanitized keyword: '%v'\n", keyword, sanitizedKeyword)
		}

		for _, keyword := range search.Exclude {
			sanitizedKeyword = reg.ReplaceAllString(keyword, "")
			if len(sanitizedKeyword) > 0 {
				excludedKeywords = append(excludedKeywords, sanitizedKeyword)
			}
		}

		includeQuery := strings.Join(includedKeywords, " & ")
		excludeQuery := strings.Join(excludedKeywords, " & ")

		// log.Printf("Collection ID: %v\n", search.CollectionID)
		// log.Printf("Tags: %v\n", search.Tags)
		// log.Printf("Before: %v\n", search.Before)
		// log.Printf("After: %v\n", search.After)
		// log.Printf("Include: %v\n", search.Include)
		// log.Printf("Exclude: %v\n", search.Exclude)
		// log.Printf("Include Query: %v\n", includeQuery)
		// log.Printf("Exclude Query: %v\n", excludeQuery)

		// Call the database function
		rows, err := db.Query("SELECT * FROM advanced_search_collection($1, $2, $3, $4, $5, $6)",
			search.CollectionID, pq.Array(search.Tags), search.Before, search.After, includeQuery, excludeQuery)
		if err != nil {
			log.Printf("Search POST - Unable to retrieve search results from database for user %d in collection %d: %v\n", session.Values["user_id"], search.CollectionID, err)
			SendError(w, DATABASE_ERROR_MESSAGE, http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		// Retrieve rows from database
		results := make([]SearchResult, 0)
		for rows.Next() {
			var result SearchResult
			if err := rows.Scan(&result.SongID, &result.SongName); err != nil {
				log.Printf("Search POST - Unable to retrieve row from database result: %v\n", err)
			}
			results = append(results, result)
		}

		// Check for errors from iterating over rows.
		if err := rows.Err(); err != nil {
			log.Printf("Search GET - Error retrieving search results from database result: %v\n", err)
			SendError(w, DATABASE_ERROR_MESSAGE, http.StatusInternalServerError)
			return
		}

		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(results)
		return
	}

}
