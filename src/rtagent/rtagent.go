// rtagent.go
// This file has the gRPC server implementation
// This file needs to be as program agnostic as possible. With the
// OFA forwarding chains, we will be very restricted in the number
// of program that we can fully support. And, every program will
// have a different OFA chain mapping. Keep the mapping out of this
// file as much as possible.

package main

import (
	"fmt"
	"globals"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"io"
	"log"
	"net"
	"p4"
	"simplerouter"
)

const (
	port = ":51977"
)

var (
	implementationExists	bool	= false
	p4infoAnnotation		string
)

type p4RuntimeServer struct{}

// Look at the annotation in the p4Info and set the implementation
// status and pick up a target package accordingly.
func setImplementationStatus() {
	implementationExists = false
	for _, annotation := range globals.MyP4Info.GetPkgInfo().GetAnnotations() {
		fmt.Printf("Checking annotation: %s\n", annotation)
		switch annotation {
		case "ofa_package_simplerouter":
			implementationExists = true
			p4infoAnnotation = annotation
		default:
			// Don't know this annotation
		}
	}
}

// Execute the table operation. Based on the annotation we got in
// the p4Info, we'll pick the OFA chain implementation. If we
// don't have a package for the specific program, pass on the
// update.
func tableOperation(tableEntry *p4.TableEntry, op p4.Update_Type) {
	if implementationExists == false {
		fmt.Println("No matching implementation found for this program")
		return
	}
	fmt.Printf("Implementation tag: %s\n", p4infoAnnotation);
	switch p4infoAnnotation {
	case "ofa_package_simplerouter":
		simplerouter.HandleTableOperation(tableEntry, op)
	default:
		fmt.Println("No matching implementation for this annotation")
	}
}

// Write from the controller to the switch.
func (s *p4RuntimeServer) Write(ctx context.Context, wrReq *p4.WriteRequest) (*p4.WriteResponse, error) {
	fmt.Println("Write from controller")
	fmt.Printf("\nReceived: %+v\n", wrReq)
	// Demux based on the update received
	for _, singleUpdate := range wrReq.GetUpdates() {
		switch x := singleUpdate.Entity.Entity.(type) {
		case *p4.Entity_TableEntry:
			fmt.Println("Table Entry message received")
			tableOperation(x.TableEntry, singleUpdate.Type)
		case nil:
			fmt.Println("Field not set")
		default:
			fmt.Println("Unsupported entity received, Type %T", x)
		}

	}
	return &p4.WriteResponse{}, nil
}

// Read state from the switch/device and respond back to the controller.
func (s *p4RuntimeServer) Read(rdReq *p4.ReadRequest, stream p4.P4Runtime_ReadServer) error {

	return nil
}

// Set the forwarding pipeline config
func (s *p4RuntimeServer) SetForwardingPipelineConfig(ctx context.Context, cfgSetReq *p4.SetForwardingPipelineConfigRequest) (*p4.SetForwardingPipelineConfigResponse, error) {
	fmt.Printf("Received forwarding pipeline config for device %d, role %d %+v\n",
		cfgSetReq.DeviceId, cfgSetReq.RoleId, cfgSetReq.Config.P4Info)
	globals.MyP4Info = *cfgSetReq.Config.GetP4Info()
	fmt.Printf("Package info: %v\n", globals.MyP4Info.GetPkgInfo())
	fmt.Printf("Table info: %+v\n", globals.MyP4Info.GetTables())
	fmt.Println(globals.MyP4Info.Tables[0].Preamble.Name)

	// Look at the Package info in the p4Info and set the implementation
	// status.
	setImplementationStatus()
	return &p4.SetForwardingPipelineConfigResponse{}, nil
}

// Return the forwarding pipeline config
func (s *p4RuntimeServer) GetForwardingPipelineConfig(ctx context.Context, cfgGetReq *p4.GetForwardingPipelineConfigRequest) (*p4.GetForwardingPipelineConfigResponse, error) {

	return &p4.GetForwardingPipelineConfigResponse{}, nil
}

// Bi-directional stream channel for packet-IO
func (s *p4RuntimeServer) StreamChannel(stream p4.P4Runtime_StreamChannelServer) error {
	fmt.Println("Starting bi-directional channel")
	for {
		inData, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		fmt.Printf("%v", inData)
	}

	return nil
}

func main() {
	fmt.Println("Runtime agent program")
	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("Error setting up runtime agent(Lis): %v", err)
	}
	fmt.Println("Listening in port ", port)
	serv := grpc.NewServer()
	p4.RegisterP4RuntimeServer(serv, &p4RuntimeServer{})
	reflection.Register(serv)
	err = serv.Serve(lis)
	if err != nil {
		log.Fatalf("Error setting up runtime agent(Srv): %v", err)
	}
}
