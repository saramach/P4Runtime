package main

import (
	"fmt"
	"log"
	"net"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"p4"
	p4_config "p4/config"
	"io"
	)

const (
	port = ":51977"
)

var p4Info p4_config.P4Info


type p4RuntimeServer struct {}

func (s *p4RuntimeServer) Write(ctx context.Context, wrReq *p4.WriteRequest) (*p4.WriteResponse, error) {
	fmt.Println("Write from controller")
	fmt.Printf("\nReceived: %+v\n", wrReq)
	// Demux based on the update received
	for _, singleUpdate := range wrReq.GetUpdates() {
		switch x := singleUpdate.Entity.Entity.(type) {
			case  *p4.Entity_TableEntry:
				fmt.Println("Table Entry message received")
				tableEntry := x.TableEntry
				handleTableOperation(tableEntry, singleUpdate.Type)
			case nil:
				fmt.Println("Field not set")
			default:
				fmt.Println("Unsupported entity received, Type %T", x)
		}

	}
	return &p4.WriteResponse{}, nil
}

func (s *p4RuntimeServer) Read(rdReq *p4.ReadRequest, stream p4.P4Runtime_ReadServer) error {

	return nil
}

func (s *p4RuntimeServer) SetForwardingPipelineConfig(ctx context.Context, cfgSetReq *p4.SetForwardingPipelineConfigRequest) (*p4.SetForwardingPipelineConfigResponse, error) {
	fmt.Printf("Received forwarding pipeline config for device %d, role %d %+v\n",
			 	cfgSetReq.DeviceId, cfgSetReq.RoleId, cfgSetReq.Config.P4Info)
	p4Info =  *cfgSetReq.Config.GetP4Info()
	fmt.Printf("Table info: %+v\n", p4Info.Tables)
	fmt.Println(p4Info.Tables[0].Preamble.Name)
	return &p4.SetForwardingPipelineConfigResponse{}, nil
}

func (s *p4RuntimeServer) GetForwardingPipelineConfig(ctx context.Context, cfgGetReq *p4.GetForwardingPipelineConfigRequest) (*p4.GetForwardingPipelineConfigResponse, error) {

	return &p4.GetForwardingPipelineConfigResponse{}, nil
}

func (s *p4RuntimeServer) StreamChannel(stream p4.P4Runtime_StreamChannelServer) error {
	fmt.Println("Starting bi-directional channel")
	for {
		inData, err := stream.Recv()
		if (err == io.EOF) {
			return nil
		}
		fmt.Printf("%v", inData);
	}

	return nil
}

func main() {
	fmt.Println("Runtime agent program")
	lis, err := net.Listen("tcp", port)
	if (err != nil) {
		log.Fatalf("Error setting up runtime agent(Lis): %v", err)
	}
	fmt.Println("Listening in port ", port)
	serv := grpc.NewServer()
	p4.RegisterP4RuntimeServer(serv, &p4RuntimeServer{})
	reflection.Register(serv)
	err = serv.Serve(lis)
	if (err != nil) {
		log.Fatalf("Error setting up runtime agent(Srv): %v", err)
	}
}

