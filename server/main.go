package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	pb "github.com/57ajay/collabwrite-server/proto"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc"
)

type apiConfig struct {
	DB *pgxpool.Pool
}

func startGrpcServer(producer *KafkaProducer) {
	lis, err := net.Listen("tcp", ":9090")
	if err != nil {
		log.Fatalf("Failed to listen on port 9090: %v", err)
	}
	s := grpc.NewServer()
	pb.RegisterDocumentServiceServer(s, newGrpcServer(producer))
	log.Println("gRPC server is running on port 9090")
	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to start gRPC server: %v", err)
	}
}

func main() {
	connStr := "postgres://ajay:57ajay@localhost:5432/collabwrite?sslmode=disable"

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	dbPool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer dbPool.Close()

	err = dbPool.Ping(ctx)
	if err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}
	fmt.Println("Successfully connected to the database!")

	// kafka producer
	kafkaBrokers := []string{"localhost:9092"}
	kafkaTopic := "document-updates"
	kafkaProducer := NewKafkaProducer(kafkaBrokers, kafkaTopic)
	defer kafkaProducer.Close()

	// kafka consumer
	consumerGroupID := "persister-group"
	kafkaConsumer := NewKafkaConsumer(kafkaBrokers, kafkaTopic, consumerGroupID, dbPool)
	defer kafkaConsumer.Close()
	go kafkaConsumer.Run(context.Background())

	go startGrpcServer(kafkaProducer)

	apiCfg := &apiConfig{
		DB: dbPool,
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
	ctx := r.Context()
	var doc Document

	err := json.NewDecoder(r.Body).Decode(&doc)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	query := `INSERT INTO documents (title, content) VALUES ($1, $2) RETURNING id, created_at, updated_at`
	err = api.DB.QueryRow(ctx, query, doc.Title, doc.Content).Scan(&doc.ID, &doc.CreatedAt, &doc.UpdatedAt)
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
	ctx := r.Context()
	docId := chi.URLParam(r, "docID")
	var doc Document

	query := `SELECT id, title, content, created_at, updated_at FROM documents WHERE id = $1`
	err := api.DB.QueryRow(ctx, query, docId).Scan(&doc.ID, &doc.Title, &doc.Content, &doc.CreatedAt, &doc.UpdatedAt)
	if err != nil {
		if err.Error() == "no rows in result set" {
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
	ctx := r.Context()
	docId := chi.URLParam(r, "docID")
	var doc Document

	err := json.NewDecoder(r.Body).Decode(&doc)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	query := `UPDATE documents SET title = $1, content = $2, updated_at = NOW() WHERE id = $3 RETURNING updated_at`
	err = api.DB.QueryRow(ctx, query, doc.Title, doc.Content, docId).Scan(&doc.UpdatedAt)
	if err != nil {
		if err.Error() == "no rows in result set" {
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
