// controller.go
package main

import (
	"fmt"
	"log"
	"google.golang.org/grpc"
	"golang.org/x/net/context"
	"p4"
	p4_config "p4/config"
	"github.com/golang/protobuf/jsonpb"
	"os"
	//"p4InfoUtils"
	"flag"
	"sync"
	"io"
	//"io/ioutil"
	//"encoding/json"
	)

const (
	p4SrvAddr	= "127.0.0.1:51977"
	deviceId	= 1
	roleId		= 2
)

var client p4.P4RuntimeClient
var p4Info p4_config.P4Info

/*
type P4Operations struct {
	Table []struct {
		Name    string `json:"name"`
		Op      string `json:"op"`
		Prefix  string `json:"prefix"`
		Plen    int    `json:"plen"`
		Nexthop string `json:"nexthop,omitempty"`
		Port    string `json:"port,omitempty"`
	} `json:"table"`
}

func getOperations(opFile string) P4Operations {
	data, err := ioutil.ReadFile(opFile)
	if (err != nil) {
		log.Fatalf("Error opening operations file")
		os.Exit(1)
	}
	var operations P4Operations
	json.Unmarshal(data, &operations)
	return operations
}

func deletePrefix (tableName string, prefix string, prefixLen int32) {
	tableId := p4InfoUtils.GetTableIdFromName(&p4Info, tableName)
	matchIdDestAddr := p4InfoUtils.GetMatchIdInTable(&p4Info, tableId, "ipv4.dstAddr")

	var tableEntry p4.TableEntry
	tableEntry.TableId = tableId
	matchList := make([]*p4.FieldMatch,1)
	LpmMatch := p4.FieldMatch_LPM{Value: []byte(prefix), PrefixLen:prefixLen}
	matchList[0] = &p4.FieldMatch{FieldId:matchIdDestAddr,
								FieldMatchType: &p4.FieldMatch_Lpm{&LpmMatch}}

	// Ship this table entry delete to the RT agent in the switch
	tabEntity := &p4.Entity_TableEntry{TableEntry: &tableEntry}
	entity := &p4.Entity{Entity:tabEntity}
	updates := make([]*p4.Update, 1)
	singleUpdate := &p4.Update{Type: p4.Update_DELETE, Entity: entity}
	updates[0] = singleUpdate
	_, errw := client.Write(context.Background(),
							&p4.WriteRequest{DeviceId: deviceId,
											 RoleId: roleId,
											 Updates: updates})
	if errw != nil {
		log.Fatalf("Error sending table entry to rt agent in switch")
	}
}

func insertPrefix (tableName string, prefix string, prefixLen int32, nexthop string, port string) {
	// Create a table entry ipv4_lpm table.
	// Filling the complex protobuf generated struct is not easy because of the
	// nesting.
	// Is there a better way of doing this?
	tableId := p4InfoUtils.GetTableIdFromName(&p4Info, tableName)
	matchIdDestAddr := p4InfoUtils.GetMatchIdInTable(&p4Info, tableId, "ipv4.dstAddr")
	actionSetNhId := p4InfoUtils.GetActionId(&p4Info, "set_nhop")
	actionParam1Id := p4InfoUtils.GetParamIdInAction(&p4Info, actionSetNhId, "nhop_ipv4")
	actionParam2Id := p4InfoUtils.GetParamIdInAction(&p4Info, actionSetNhId, "port")

	var tableEntry p4.TableEntry
	tableEntry.TableId = tableId
	matchList := make([]*p4.FieldMatch,1)
	LpmMatch := p4.FieldMatch_LPM{Value: []byte(prefix), PrefixLen:prefixLen}
	matchList[0] = &p4.FieldMatch{FieldId:matchIdDestAddr,
								FieldMatchType: &p4.FieldMatch_Lpm{&LpmMatch}}
	actionParam1 := p4.Action_Param{ParamId: actionParam1Id, Value: []byte(nexthop)}
	actionParam2 := p4.Action_Param{ParamId: actionParam2Id, Value: []byte(port)}
	// 2 action params
	paramList := make([]*p4.Action_Param, 2)
	paramList[0] = &actionParam1
	paramList[1] = &actionParam2
	action := p4.Action{ActionId:actionSetNhId,
						Params: paramList}
	tableAction := p4.TableAction_Action{Action:&action}
	tableEntry.Match = matchList
	tableEntry.Action = &p4.TableAction{Type: &tableAction}

	//fmt.Printf("%+v", tableEntry)

	// Ship this table entry to the RT agent in the switch
	tabEntity := &p4.Entity_TableEntry{TableEntry: &tableEntry}
	entity := &p4.Entity{Entity:tabEntity}
	updates := make([]*p4.Update, 1)
	singleUpdate := &p4.Update{Type: p4.Update_INSERT, Entity: entity}
	updates[0] = singleUpdate
	_, errw := client.Write(context.Background(),
							&p4.WriteRequest{DeviceId: deviceId,
											 RoleId: roleId,
											 Updates: updates})
	if errw != nil {
		log.Fatalf("Error sending table entry to rt agent in switch")
	}
}
*/
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

	conn, err :=grpc.Dial(p4SrvAddr, grpc.WithInsecure())
	if (err != nil) {
		log.Fatalf("cannot connect to p4rt agent")
	}
	defer conn.Close()
	client = p4.NewP4RuntimeClient(conn)
	fmt.Println("Connected to rtagent..")

	// Need to open an bidirectional stream to the server
	stream, sErr :=  client.StreamChannel(context.Background())
	if (sErr != nil) {
		log.Fatalf("cannot open stream channel with the server")
	}
	fmt.Println("Bidirectional stream opened")
	var waitg sync.WaitGroup
	waitg.Add(1)

	go func() {
		defer waitg.Done()
		for {
			inData, err := stream.Recv()
			if (err == io.EOF) {
				return
			}
			fmt.Printf("%v", inData)
			// Act on the received message
		}
	}()
	p4infoFile, ferr := os.Open(*p4InfoPtr)
	if (ferr != nil) {
		log.Fatalf("Cannot find the p4info %s for the P4 program", *p4InfoPtr)
		os.Exit(3)
	}
	// Unmarshall the Json data into a structure that can be sent
	// over in a protobuf message to the RT agent running in the 
	// switch
	jerr := jsonpb.Unmarshal(p4infoFile, &p4Info)
	if (jerr != nil) {
		log.Fatalf("Error parsing p4info json file")
	}
	// Send the device ID and the P4Info to the RT agent
	fwdPlCfg :=  &p4.SetForwardingPipelineConfigRequest{DeviceId: deviceId, RoleId: roleId,
														Config: &p4.ForwardingPipelineConfig{P4Info: &p4Info}}
	_, err1 := client.SetForwardingPipelineConfig(context.Background(),
												  fwdPlCfg)
	if (err1 != nil) {
		log.Fatalf("Got error from setting pipeline config")
	}

	// Read the json file with the operations we need to perform
	playTableOperations(*opInfoPtr)
/*
	fmt.Println("Controller: Parsing table operations")
	operations := getOperations(*opInfoPtr)
	for _, operation := range operations.Table {
		fmt.Println(operation.Name, operation.Prefix,
					operation.Plen, operation.Nexthop, operation.Port)
		switch operation.Op {
		case "insert":
			insertPrefix(operation.Name,
						 operation.Prefix,
						 int32(operation.Plen),
						 operation.Nexthop,
						 operation.Port)
		case "delete":
			deletePrefix(operation.Name,
						 operation.Prefix,
						 int32(operation.Plen))
		default:
			fmt.Println("Unsupported operation")
		}

	}
*/
/*
	insertPrefix("ipv4_lpm", "0a000000", 24, "0b000001", "12")
	insertPrefix("ipv4_lpm", "0c000000", 24, "0b000001", "12")
    deletePrefix("ipv4_lpm", "0a000000", 24)
*/
	waitg.Wait()
}
