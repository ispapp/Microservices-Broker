package lib

import (
	"Microservices-Broker/base/pb"
	"io"
	"log"
	"sync"
	"time"

	"go.mills.io/bitcask/v2"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Server struct {
	pb.UnimplementedBidistreamerServer
	db           *bitcask.Bitcask
	mu           sync.Mutex
	tickeSeconds int16
	maxStored    int32
	clients      map[string]map[string]pb.Bidistreamer_ReceiveServer
	bidiClient   map[string]map[string]pb.Bidistreamer_ReceiveServer
}

func NewServer(dbPath string, TickeSeconds int16, MaxStored int32) (*Server, error) {
	db, err := bitcask.Open(dbPath)
	if err != nil {
		return nil, err
	}
	s := &Server{
		db:           db,
		tickeSeconds: TickeSeconds,
		maxStored:    MaxStored,
		clients:      make(map[string]map[string]pb.Bidistreamer_ReceiveServer),
		bidiClient:   make(map[string]map[string]pb.Bidistreamer_ReceiveServer),
	}
	go s.startCronJob()
	return s, nil
}

func (s *Server) startCronJob() {
	ticker := time.NewTicker(time.Duration(s.tickeSeconds) * time.Second)
	for range ticker.C {
		s.checkMessageDelivery()
	}
}

func (s *Server) checkMessageDelivery() {
	s.mu.Lock()
	defer s.mu.Unlock()
	// Implement logic to check message delivery
}

func (s *Server) Send(stream pb.Bidistreamer_SendServer) error {
	for {
		msg, err := stream.Recv()
		if err == io.EOF {
			return stream.SendAndClose(&pb.Status{Message: "All messages received", Success: true, Error: pb.Error_NONE})
		}
		if err != nil {
			return stream.SendAndClose(&pb.Status{Message: err.Error(), Success: false, Error: pb.Error_SERVER_ERROR})
		}
		log.Printf("Received message from %s to %s: %v", msg.From, msg.To, msg)
		err = s.storeMessage(msg.From, msg)
		if err != nil {
			return stream.SendAndClose(&pb.Status{Message: err.Error(), Success: false, Error: pb.Error_SERVER_ERROR})
		}
		// Check if recipient exists in clients map and send the message
		s.mu.Lock()
		if clients, exists := s.clients[msg.To]; exists {
			for _, clientStream := range clients {
				if err := clientStream.Send(msg); err != nil {
					log.Printf("Failed to send message to %s: %v", msg.To, err)
				}
			}
		}
		s.mu.Unlock()
	}
}

func (s *Server) BidiStream(stream pb.Bidistreamer_BidiStreamServer) error {

	for {
		msg, err := stream.Recv()
		md, ok := stream.Context().Value("metadata").(map[string]string)
		if !ok {
			return stream.Send(&pb.Message{Data: []byte("missing metadata"), Type: pb.Type_TEXT, Seq: timestamppb.Now(), From: "broker", To: "client"})
		}
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		log.Printf("Received message from %s to %s: %v", msg.From, msg.To, msg)
		clientID := md["X-CLIENT-ID"]
		s.mu.Lock()
		if _, exists := s.bidiClient[msg.From]; !exists {
			s.bidiClient[msg.From] = make(map[string]pb.Bidistreamer_ReceiveServer)
		}
		s.bidiClient[msg.From][clientID] = stream
		// Check if recipient exists in clients map and send the message
		if clients, exists := s.clients[msg.To]; exists {
			for _, clientStream := range clients {
				if err := clientStream.Send(msg); err != nil {
					log.Printf("Failed to send message to %s: %v", msg.To, err)
				}
			}
		} else {
			s.mu.Unlock()
			return stream.Send(&pb.Message{Data: []byte("destination client not found"), Type: pb.Type_TEXT, Seq: timestamppb.Now(), From: "broker", To: msg.From})
		}
		s.mu.Unlock()

		if err := stream.Send(msg); err != nil {
			return err
		}
	}
}

func (s *Server) Receive(empty *pb.Empty, stream pb.Bidistreamer_ReceiveServer) error {
	// Implement your logic to receive messages from the broker
	md, ok := stream.Context().Value("metadata").(map[string]string)
	if !ok {
		return stream.Send(&pb.Message{Data: []byte("missing metadata"), Type: pb.Type_TEXT, Seq: timestamppb.Now(), From: "broker", To: "client"})
	}
	serviceName, exists := md["X-SERVICE-NAME"]
	if !exists {
		return stream.Send(&pb.Message{Data: []byte("missing X-SERVICE-NAME"), Type: pb.Type_TEXT, Seq: timestamppb.Now(), From: "broker", To: "client"})
	}
	clientID := md["X-CLIENT-ID"]
	if _, exists := s.clients[serviceName]; !exists {
		s.clients[serviceName] = make(map[string]pb.Bidistreamer_ReceiveServer)
	}
	s.clients[serviceName][clientID] = stream

	// Check for existing messages in the database
	s.mu.Lock()
	defer s.mu.Unlock()
	err := s.db.Scan(bitcask.Key(serviceName+"_"), bitcask.KeyFunc(func(key bitcask.Key) error {
		value, err := s.db.Get(key)
		if err != nil {
			return err
		}
		var msg pb.Message
		if err := proto.Unmarshal(value, &msg); err != nil && !msg.Done {
			return err
		}
		if err := stream.Send(&msg); err != nil {
			return err
		} else {
			key := bitcask.Key(serviceName + "_" + msg.Seq.String())
			if err := s.db.Delete(key); err != nil {
				return err
			}
		}
		return nil
	}))
	if err != nil {
		return err
	}

	// Remove client from map when done
	defer func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		delete(s.clients[serviceName], clientID)
		if len(s.clients[serviceName]) == 0 {
			delete(s.clients, serviceName)
		}
	}()

	return nil
}

func (s *Server) storeMessage(serviceName string, msg *pb.Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	// Store message in Bitcast DB
	key := bitcask.Key(serviceName + "_" + msg.Seq.String())
	value, _err := proto.Marshal(msg)
	if _err != nil {
		log.Printf("Failed to marshal message: %v", _err)
		return _err
	}
	if err := s.db.Put(key, value); err != nil {
		log.Printf("Failed to store message: %v", err)
		return _err
	}
	return nil
}
