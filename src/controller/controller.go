// controller.go
// This is a temporary controller program to drive the runtime
// agent in the switch. This controller has very limited functionality
// and is used only for test purposes.
package main

import (
	"flag"
	"fmt"
	"github.com/golang/protobuf/jsonpb"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"io"
	"log"
	"os"
	"p4"
	p4_config "p4/config"
	"sync"
)

const (
	p4SrvAddr = "127.0.0.1:51977"
	deviceId  = 1
	roleId    = 2
)

var client p4.P4RuntimeClient
var p4Info p4_config.P4Info

func main() {
	fmt.Println("Controller program")
	p4InfoPtr := flag.String("p4info",
		"/home/saramach/P4Runtime/simple_router-p4info.json",
		"full path to the p4info json file")
	opInfoPtr := flag.String("operations",
		"/home/saramach/P4Runtime/operations.json",
		"full path to the operations json file")

	flag.Parse()
	fmt.Println("Using P4Info file: ", *p4InfoPtr)
	fmt.Println("Using operations file:", *opInfoPtr)

	conn, err := grpc.Dial(p4SrvAddr, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("cannot connect to p4rt agent")
	}
	defer conn.Close()
	client = p4.NewP4RuntimeClient(conn)
	fmt.Println("Connected to rtagent..")

	// Need to open an bidirectional stream to the server
	stream, sErr := client.StreamChannel(context.Background())
	if sErr != nil {
		log.Fatalf("cannot open stream channel with the server")
	}
	fmt.Println("Bidirectional stream opened")
	var waitg sync.WaitGroup
	waitg.Add(1)

	go func() {
		defer waitg.Done()
		for {
			inData, err := stream.Recv()
			if err == io.EOF {
				return
			}
			fmt.Printf("%v", inData)
			// Act on the received message
		}
	}()
	p4infoFile, ferr := os.Open(*p4InfoPtr)
	if ferr != nil {
		log.Fatalf("Cannot find the p4info %s for the P4 program", *p4InfoPtr)
		os.Exit(3)
	}
	// Unmarshall the Json data into a structure that can be sent
	// over in a protobuf message to the RT agent running in the
	// switch
	jerr := jsonpb.Unmarshal(p4infoFile, &p4Info)
	if jerr != nil {
		log.Fatalf("Error parsing p4info json file")
	}
	// Send the device ID and the P4Info to the RT agent
	fwdPlCfg := &p4.SetForwardingPipelineConfigRequest{DeviceId: deviceId, RoleId: roleId,
		Config: &p4.ForwardingPipelineConfig{P4Info: &p4Info}}
	_, err1 := client.SetForwardingPipelineConfig(context.Background(),
		fwdPlCfg)
	if err1 != nil {
		log.Fatalf("Got error from setting pipeline config")
	}

	// Read the json file with the operations we need to perform
	playTableOperations(*opInfoPtr)

	waitg.Wait()
}
