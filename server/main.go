package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	_ "github.com/jackc/pgx/v5/stdlib"
)

type apiConfig struct {
	DB *sql.DB
}

func main() {
	connStr := "postgres://ajay:57ajay@localhost:5432/collabwrite?sslmode=disable"

	db, err := sql.Open("pgx", connStr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	err = db.Ping()
	if err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	fmt.Println("Successfully connected to the database!")

	apiCfg := &apiConfig{
		DB: db,
	}

	r := chi.NewRouter()
	r.Use(middleware.Logger)

	r.Post("/documents", apiCfg.createDocumentHandler)
	r.Get("/documents/{docID}", apiCfg.getDocumentHandler)
	r.Put("/documents/{docID}", apiCfg.updateDocumentHandler)

	port := "8080"
	log.Printf("Server is running on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, r))

}

func (api *apiConfig) createDocumentHandler(w http.ResponseWriter, r *http.Request) {
	var doc Document

	err := json.NewDecoder(r.Body).Decode(&doc)

	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	query := `INSERT INTO documents (title, content) VALUES ($1, $2) RETURNING id, created_at, updated_at`
	err = api.DB.QueryRow(query, doc.Title, doc.Content).Scan(&doc.ID, &doc.CreatedAt, &doc.UpdatedAt)

	if err != nil {
		log.Printf("Failed to insert document: %v", err)
		http.Error(w, "Failed to create document", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(doc)
}

func (api *apiConfig) getDocumentHandler(w http.ResponseWriter, r *http.Request) {
	docId := chi.URLParam(r, "docID")

	var doc Document
	query := `SELECT id, title, content, created_at, updated_at FROM documents WHERE id = $1`
	err := api.DB.QueryRow(query, docId).Scan(&doc.ID, &doc.Title, &doc.Content, &doc.CreatedAt, &doc.UpdatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Document not found", http.StatusNotFound)
			return
		}
		log.Printf("Failed to get document: %v", err)
		http.Error(w, "Failed to get document", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(doc)

}

func (api *apiConfig) updateDocumentHandler(w http.ResponseWriter, r *http.Request) {
	docId := chi.URLParam(r, "docID")

	var doc Document
	err := json.NewDecoder(r.Body).Decode(&doc)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	query := `UPDATE documents SET title = $1, content = $2, updated_at = NOW() WHERE id = $3 RETURNING updated_at`
	err = api.DB.QueryRow(query, doc.Title, doc.Content, docId).Scan(&doc.UpdatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Document not found", http.StatusNotFound)
			return
		}
		log.Printf("Failed to update document: %v", err)
		http.Error(w, "Failed to update document", http.StatusInternalServerError)
		return
	}

	doc.ID = docId
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(doc)
}
