package main

import (
	"context"
	"io"
	"log"
	"sync"

	pb "github.com/57ajay/collabwrite-server/proto"
)

type documentStreams struct {
	sync.RWMutex
	streams map[string]map[string]pb.DocumentService_DocumentStreamServer

	// {
	//		docID: {
	//				clientID1: pb.DocumentService_DocumentStreamServer
	//				clientID2: pb.DocumentService_DocumentStreamServer
	// 				...	 more clients
	//				}
	//
	//}
}

type grpcServer struct {
	pb.UnimplementedDocumentServiceServer
	kafkaProducer *KafkaProducer
}

func newGrpcServer(producer *KafkaProducer) *grpcServer {
	return &grpcServer{
		kafkaProducer: producer,
	}
}

func (s *grpcServer) DocumentStream(stream pb.DocumentService_DocumentStreamServer) error {

	// now this main loop is to receive a message and produce it to Kafka.
	for {

		req, err := stream.Recv()
		if err == io.EOF {
			log.Println("Client closed the stream")
			return nil
		}

		if err != nil {
			log.Printf("Error receiving message: %v", err)
			return err
		}

		msg := DocumentUpdateMessage{
			DocumentID: req.GetDocumentId(),
			ClientID:   req.GetClientId(),
			Content:    req.GetContent(),
		}

		if err := s.kafkaProducer.ProduceMessage(context.Background(), msg); err != nil {
			// will try to send the error back to client later, maybe
			log.Printf("Failed to produce message to Kafka: %v", err)
		}

	}

}
