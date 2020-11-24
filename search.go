package main

import (
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
)

// SearchResult is a struct that models a result from a search
type SearchResult struct {
	SongID   int64  `json:"song_id" db:"song_id"`
	SongName string `json:"song_name" db:"song_name"`
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

		query := strings.Join(strings.Fields(rawQuery), " & ")
		log.Printf("Result: '%v'\n", query)

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
	}

}
