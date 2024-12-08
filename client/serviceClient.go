package client

import (
	"context"
	"time"

	"github.com/ispapp/Microservices-Broker/base/pb"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type ServiceClient struct {
	serviceName string
	client      pb.BrokerClient
}

func NewServiceClient(serviceName, serviceUrl string) (*ServiceClient, error) {
	conn, err := grpc.NewClient(serviceUrl, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	client := pb.NewBrokerClient(conn)
	return &ServiceClient{serviceName: serviceName, client: client}, nil
}

func (c *ServiceClient) Ping() (*pb.Status, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	return c.client.Ping(ctx, &pb.Identity{From: c.serviceName})
}

func (c *ServiceClient) SendMessage(to string, data []byte, msgType pb.Type, queue bool) (*pb.Status, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	message := &pb.Message{
		From:  c.serviceName,
		To:    to,
		Data:  data,
		Type:  msgType,
		Queue: queue,
	}
	return c.client.Send(ctx, message)
}

func (c *ServiceClient) ReceiveMessages(callback func(*pb.Message)) error {
	stream, err := c.client.Receive(context.Background(), &pb.Identity{From: c.serviceName})
	if err != nil {
		return err
	}
	for {
		msg, err := stream.Recv()
		if err != nil {
			return err
		}
		callback(msg)
	}
}

func (c *ServiceClient) Cleanup() (*pb.Status, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	return c.client.Cleanup(ctx, &pb.Identity{From: c.serviceName})
}
