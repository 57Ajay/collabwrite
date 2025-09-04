package main

import (
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
	docStreams *documentStreams
}

func newGrpcServer() *grpcServer {
	return &grpcServer{
		docStreams: &documentStreams{
			streams: make(map[string]map[string]pb.DocumentService_DocumentStreamServer),
		},
	}
}

func (s *grpcServer) DocumentStream(stream pb.DocumentService_DocumentStreamServer) error {

	// this is gonna be the first message from the client, and this will contain clientID and docID
	initialMsg, err := stream.Recv()
	if err != nil {
		log.Printf("Error receiving initial message: %v", err)
		return err
	}

	// this will add client's stream to our map
	docID := initialMsg.GetDocumentId()
	clientID := initialMsg.ClientId
	log.Printf("Client %s connected to document %s", clientID, docID)

	s.docStreams.Lock()
	if _, ok := s.docStreams.streams[docID]; !ok {
		s.docStreams.streams[docID] = make(map[string]pb.DocumentService_DocumentStreamServer)
	}
	s.docStreams.streams[docID][clientID] = stream
	s.docStreams.Unlock()

	// in case client disconnects
	defer func() {
		log.Printf("Client %s disconnected from document %s", clientID, docID)
		s.docStreams.Lock()
		delete(s.docStreams.streams[docID], clientID)
		if len(s.docStreams.streams[docID]) == 0 {
			delete(s.docStreams.streams, docID)
		}
	}()

	// now this is main loop where we receive message from this ClientID and bradcast them
	for {
		req, err := stream.Recv()
		if err == io.EOF {
			return nil // this means client closed the stream
		}
		if err != nil {
			log.Printf("Error receiving message from %s: %v", clientID, err)
			return err
		}

		// now we will broadcast the same message to all the client in the DocumentStream
		s.docStreams.RLock()

		for id, clientStream := range s.docStreams.streams[docID] {

			if id != clientID { // ignore the original client who made changes
				if err := clientStream.Send(&pb.DocumentUpdateResponse{
					DocumentId: req.GetDocumentId(),
					Content:    req.GetContent(),
					ClientId:   req.GetClientId(),
				}); err != nil {
					log.Printf("Error sending message to client %s: %v", id, err)
				}
			}

		}
		s.docStreams.RUnlock()
	}
}
