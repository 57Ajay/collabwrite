package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	pb "github.com/57ajay/collabwrite-server/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	docID := flag.String("docID", "default-doc", "The ID of the document to connect to.")
	clientID := flag.String("clientID", "default-client", "The ID of this client")
	flag.Parse()

	conn, err := grpc.NewClient("localhost:9090", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}

	defer conn.Close()

	client := pb.NewDocumentServiceClient(conn)
	stream, err := client.DocumentStream(context.Background())

	if err != nil {
		log.Fatalf("Error on DocumentStream: %v", err)
	}

	initialReq := &pb.DocumentUpdateRequest{
		DocumentId: *docID,
		ClientId:   *clientID,
	}

	if err = stream.Send(initialReq); err != nil {
		log.Fatalf("Failed to send initial message: %v", err)
	}

	log.Printf("Connected to document %s as client %s", *docID, *clientID)

	go func() {
		for {
			res, err := stream.Recv()
			if err == io.EOF {
				log.Println("Server closed the stream")
				return
			}
			if err != nil {
				log.Fatalf("Failed to receive: %v", err)
			}

			if res.GetClientId() != *clientID {
				fmt.Printf("\n[Update from %s]: %s\n> ", res.GetClientId(), res.GetContent())
			}
		}
	}()

	scanner := bufio.NewScanner(os.Stdin)
	fmt.Print("> ")

	for scanner.Scan() {
		content := scanner.Text()
		req := &pb.DocumentUpdateRequest{
			DocumentId: *docID,
			ClientId:   *clientID,
			Content:    content,
		}

		if err := stream.Send(req); err != nil {
			log.Printf("Failed to send message: %v", err)
		}

		fmt.Print("> ")

	}
}
