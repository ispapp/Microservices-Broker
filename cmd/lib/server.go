package lib

import (
	"Microservices-Broker/base/pb"
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"go.mills.io/bitcask/v2"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Server struct {
	pb.UnimplementedBrokerServer
	db           *bitcask.Bitcask
	mu           sync.Mutex
	tickeSeconds int16
	maxAge       time.Duration
	maxStored    int32
	clients      sync.Map // Changed to sync.Map for atomic operations
}

var Utils = utils{}

func NewServer(dbPath string, TickeSeconds int16, MaxStored int32, MaxAge time.Duration) (*Server, error) {
	db, err := bitcask.Open(dbPath, bitcask.WithAutoRecovery(false), bitcask.WithDirMode(0700), bitcask.WithFileMode(0600))
	if err != nil {
		return nil, err
	}
	s := &Server{
		db:           db,
		tickeSeconds: TickeSeconds,
		maxAge:       MaxAge,
		maxStored:    MaxStored,
		clients:      sync.Map{},
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
	if !s.mu.TryLock() {
		return
	}
	defer s.mu.Unlock()
	err := s.db.Scan(nil, bitcask.KeyFunc(func(key bitcask.Key) error {
		value, err := s.db.Get(key)
		if err != nil {
			return err
		}
		var msg pb.Message
		if err := proto.Unmarshal(value, &msg); err != nil {
			return err
		}
		if time.Since(msg.Seq.AsTime()) > s.maxAge {
			if err := s.db.Delete(key); err != nil {
				return err
			}
			log.Printf("Deleted expired message %s", key)
		}
		return nil
	}))
	if err != nil {
		log.Printf("Error during message cleanup: %v", err)
	}
}

func (s *Server) Ping(ctx context.Context, identity *pb.Identity) (*pb.Status, error) {
	return &pb.Status{Message: "Pong", Success: true, Error: pb.Error_NONE}, nil
}

func (s *Server) Send(ctx context.Context, msg *pb.Message) (*pb.Status, error) {
	if msg.Data == nil || msg.From == "" || msg.To == "" {
		return &pb.Status{Message: "Invalid message", Success: false, Error: pb.Error_INVALID_REQUEST}, nil
	}
	log.Printf("Received message from %s to %s", msg.From, msg.To)
	// Check if recipient exists in clients map and send the message
	if !s.mu.TryLock() {
		return &pb.Status{Message: "Server busy", Success: false, Error: pb.Error_SERVER_ERROR}, nil
	}
	defer s.mu.Unlock()
	if clientStream, exists := s.clients.Load(msg.To); exists {
		// does not exist at the moment
		log.Printf("Sending message to %s", msg.To)
		if err := clientStream.(pb.Broker_ReceiveServer).Send(msg); err != nil {
			log.Printf("Failed to send message to %s: %v", msg.To, err)
			return &pb.Status{Message: err.Error(), Success: false, Error: pb.Error_SERVER_ERROR}, err
		}
		return &pb.Status{Message: "Message sent", Success: true, Error: pb.Error_NONE}, nil
	} else if msg.Queue {
		log.Printf("Recipient %s not found, queuing message", msg.To)
		// If recipient does not exist and message is marked for queue, store it
		err := s.storeMessage(msg.To, msg)
		if err != nil {
			log.Printf("Failed to store queued message for %s: %v", msg.To, err)
			return &pb.Status{Message: err.Error(), Success: false, Error: pb.Error_SERVER_ERROR}, err
		}
		return &pb.Status{Message: "Message queued", Success: true, Error: pb.Error_NONE}, nil
	}
	return &pb.Status{Message: "Recipient not found", Success: false, Error: pb.Error_NONE}, nil
}

func (s *Server) Receive(identity *pb.Identity, stream pb.Broker_ReceiveServer) error {
	log.Printf("Client %s connected", identity.From)
	if _, exists := s.clients.Load(identity.From); exists {
		s.clients.Store(identity.From, stream)
	}
	for {
		// Keep the connection alive
		select {
		case <-stream.Context().Done():
			log.Printf("Client %s disconnected", identity.From)
			s.clients.Delete(identity.From)

			return nil
		default:
			err := s.GetMessages(identity, stream)
			if err != nil {
				log.Printf("Failed to get messages for %s: %v", identity.From, err)
				stream.Send(&pb.Message{
					Data: []byte(err.Error()),
					Type: pb.Type_TEXT,
					Seq:  timestamppb.Now(),
					From: "broker", To: identity.From,
					Event: pb.Event_ERROR})
				return err
			}
			time.Sleep(time.Second)
		}
	}
}

func (s *Server) GetMessages(identity *pb.Identity, stream pb.Broker_ReceiveServer) error {
	serviceName := identity.From
	if serviceName == "" {
		return stream.Send(&pb.Message{Data: []byte("missing service name"), Type: pb.Type_TEXT, Seq: timestamppb.Now(), From: "broker", To: identity.From, Event: pb.Event_ERROR})
	}
	// // Check for existing messages in the database
	// if !s.mu.TryLock() {
	// 	return fmt.Errorf("Server busy")
	// }
	// defer s.mu.Unlock()
	err := s.db.Scan(bitcask.Key(serviceName+"_"), bitcask.KeyFunc(func(key bitcask.Key) error {
		value, err := s.db.Get(key)
		if err != nil {
			return err
		}
		var msg pb.Message
		if err := proto.Unmarshal(value, &msg); err != nil {
			return err
		}
		if err := stream.Send(&msg); err != nil {
			return err
		} else {
			// Delete message from database after sending
			if err := s.db.Delete(key); err != nil {
				return err
			}
			log.Printf("deleted message %s", key)
		}
		return nil
	}))
	if err != nil {
		return err
	}
	// Remove client from map when done
	defer func() {
		s.clients.Delete(serviceName)
	}()
	return nil
}

func (s *Server) Cleanup(ctx context.Context, identity *pb.Identity) (*pb.Status, error) {
	// Implement cleanup logic
	if !s.mu.TryLock() {
		return &pb.Status{Message: "Server busy", Success: false, Error: pb.Error_SERVER_ERROR}, nil
	}
	defer s.mu.Unlock()
	serviceName := identity.From
	if serviceName == "" {
		return &pb.Status{Message: "missing service name", Success: false, Error: pb.Error_INVALID_REQUEST}, nil
	}
	var count int
	err := s.db.Scan(bitcask.Key(serviceName+"_"), bitcask.KeyFunc(func(key bitcask.Key) error {
		count++
		return s.db.Delete(key)
	}))
	if err != nil {
		return &pb.Status{Message: err.Error(), Success: false, Error: pb.Error_SERVER_ERROR}, err
	}
	return &pb.Status{Message: fmt.Sprintf("Cleanup completed (%d)", count), Success: true, Error: pb.Error_NONE}, nil
}

func (s *Server) storeMessage(serviceName string, msg *pb.Message) error {
	// Store message in Bitcast DB
	key := bitcask.Key(serviceName + "_" + Utils.uid(16))
	_msg := &pb.Message{
		Data:  msg.Data,
		Type:  msg.Type,
		From:  msg.From,
		To:    msg.To,
		Event: pb.Event_MESSAGE,
		Seq:   timestamppb.Now(),
	}
	value, _err := proto.Marshal(_msg)
	if _err != nil {
		return _err
	}
	if s.db != nil {
		if err := s.db.Put(key, value); err != nil {
			return err
		}
		s.db.Sync()
	} else {
		log.Printf("Database not initialized")
	}
	log.Printf("Message queued for %s", serviceName)
	return nil
}
