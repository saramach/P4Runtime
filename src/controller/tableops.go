// tableops.go

// This file is closely tied to simple_router.p4. This code is
// only for test purposes
package main

import (
	"bufio"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"golang.org/x/net/context"
	"io/ioutil"
	"log"
	"os"
	"p4"
	"p4InfoUtils"
)

// Structure to import the data read from the json file
type P4Operations struct {
	Table []struct {
		Name    string `json:"name"`
		Op      string `json:"op"`
		Prefix  string `json:"prefix,omitempty"`
		Plen    int    `json:"plen,omitempty"`
		Nexthop string `json:"nexthop,omitempty"`
		Port    string `json:"port,omitempty"`
		NhIP    string `json:"nhIP,omitempty"`
		MacAddr string `json:"macAddr,omitempty"`
	} `json:"table"`
}

// Convert an uint32 number in the hex format to a stream of
// bytes. Typically used for things like IP address.
// Example:
// Input: "0x0a0a0a0a"
// Output: [10 10 10 10]
// Input: "01020304"
// Output: [1 2 3 4]
// Input: "ffffffff"
// Output: [255 255 255 255]
func getByteArrFromHexStr(inString string) []byte {
	byteArr, _ := hex.DecodeString(inString)
	return byteArr
}

func getOperations(opFile string) P4Operations {
	data, err := ioutil.ReadFile(opFile)
	if err != nil {
		log.Fatalf("Error opening operations file")
		os.Exit(1)
	}
	var operations P4Operations
	json.Unmarshal(data, &operations)
	fmt.Printf("Unmarshalled: %v\n", operations)
	return operations
}

func deletePrefix(tableName string, prefix string, prefixLen int32) {
	fmt.Println("Controller: Delete prefix")
	tableId := p4InfoUtils.GetTableIdFromName(&p4Info, tableName)
	matchIdDestAddr := p4InfoUtils.GetMatchIdInTable(&p4Info, tableId, "ipv4.dstAddr")

	var tableEntry p4.TableEntry
	tableEntry.TableId = tableId
	matchList := make([]*p4.FieldMatch, 1)
	LpmMatch := p4.FieldMatch_LPM{Value: getByteArrFromHexStr(prefix), PrefixLen: prefixLen}
	matchList[0] = &p4.FieldMatch{FieldId: matchIdDestAddr,
		FieldMatchType: &p4.FieldMatch_Lpm{&LpmMatch}}
	tableEntry.Match = matchList

	// Ship this table entry delete to the RT agent in the switch
	tabEntity := &p4.Entity_TableEntry{TableEntry: &tableEntry}
	entity := &p4.Entity{Entity: tabEntity}
	updates := make([]*p4.Update, 1)
	singleUpdate := &p4.Update{Type: p4.Update_DELETE, Entity: entity}
	fmt.Printf("%v\n", singleUpdate)
	updates[0] = singleUpdate
	_, errw := client.Write(context.Background(),
		&p4.WriteRequest{DeviceId: deviceId,
			RoleId:  roleId,
			Updates: updates})
	if errw != nil {
		log.Fatalf("Error sending table entry to rt agent in switch")
	}
}

func insertPrefix(tableName string, prefix string, prefixLen int32, nexthop string, port string) {
	// Create a table entry ipv4_lpm table.
	// Filling the complex protobuf generated struct is not easy because of the
	// nesting.
	// Is there a better way of doing this?
	fmt.Println("Controller: Insert prefix")
	tableId := p4InfoUtils.GetTableIdFromName(&p4Info, tableName)
	matchIdDestAddr := p4InfoUtils.GetMatchIdInTable(&p4Info, tableId, "ipv4.dstAddr")
	actionSetNhId := p4InfoUtils.GetActionId(&p4Info, "set_nhop")
	actionParam1Id := p4InfoUtils.GetParamIdInAction(&p4Info, actionSetNhId, "nhop_ipv4")
	actionParam2Id := p4InfoUtils.GetParamIdInAction(&p4Info, actionSetNhId, "port")

	var tableEntry p4.TableEntry
	tableEntry.TableId = tableId
	matchList := make([]*p4.FieldMatch, 1)
	LpmMatch := p4.FieldMatch_LPM{Value: getByteArrFromHexStr(prefix), PrefixLen: prefixLen}
	matchList[0] = &p4.FieldMatch{FieldId: matchIdDestAddr,
		FieldMatchType: &p4.FieldMatch_Lpm{&LpmMatch}}
	actionParam1 := p4.Action_Param{ParamId: actionParam1Id, Value: getByteArrFromHexStr(nexthop)}
	actionParam2 := p4.Action_Param{ParamId: actionParam2Id, Value: getByteArrFromHexStr(port)}
	// 2 action params
	paramList := make([]*p4.Action_Param, 2)
	paramList[0] = &actionParam1
	paramList[1] = &actionParam2
	action := p4.Action{ActionId: actionSetNhId,
		Params: paramList}
	tableAction := p4.TableAction_Action{Action: &action}
	tableEntry.Match = matchList
	tableEntry.Action = &p4.TableAction{Type: &tableAction}

	//fmt.Printf("%+v", tableEntry)

	// Ship this table entry to the RT agent in the switch
	tabEntity := &p4.Entity_TableEntry{TableEntry: &tableEntry}
	entity := &p4.Entity{Entity: tabEntity}
	updates := make([]*p4.Update, 1)
	singleUpdate := &p4.Update{Type: p4.Update_INSERT, Entity: entity}
	fmt.Printf("%v\n", singleUpdate)
	updates[0] = singleUpdate
	_, errw := client.Write(context.Background(),
		&p4.WriteRequest{DeviceId: deviceId,
			RoleId:  roleId,
			Updates: updates})
	if errw != nil {
		log.Fatalf("Error sending table entry to rt agent in switch")
	}
}

func insertMacMapping(tableName string, nhIP string, macAddr string) {
	fmt.Println("Controller: Insert mac binding", tableName, nhIP, macAddr)
	tableId := p4InfoUtils.GetTableIdFromName(&p4Info, tableName)
	matchIdnhIP := p4InfoUtils.GetMatchIdInTable(&p4Info, tableId, "routing_metadata.nexthop_ipv4")
	actionSetDmac := p4InfoUtils.GetActionId(&p4Info, "set_dmac")
	actionParamId := p4InfoUtils.GetParamIdInAction(&p4Info, actionSetDmac, "dmac")

	var tableEntry p4.TableEntry
	tableEntry.TableId = tableId
	matchList := make([]*p4.FieldMatch, 1)
	ExactMatch := p4.FieldMatch_Exact{Value: getByteArrFromHexStr(nhIP)}
	matchList[0] = &p4.FieldMatch{FieldId: matchIdnhIP,
		FieldMatchType: &p4.FieldMatch_Exact_{&ExactMatch}}
	actionParam1 := p4.Action_Param{ParamId: actionParamId, Value: getByteArrFromHexStr(macAddr)}

	paramList := make([]*p4.Action_Param, 1)
	paramList[0] = &actionParam1

	action := p4.Action{ActionId: actionSetDmac,
		Params: paramList}
	tableAction := p4.TableAction_Action{Action: &action}
	tableEntry.Match = matchList
	tableEntry.Action = &p4.TableAction{Type: &tableAction}

	// Ship this table entry to the RT agent in the switch
	tabEntity := &p4.Entity_TableEntry{TableEntry: &tableEntry}
	entity := &p4.Entity{Entity: tabEntity}
	updates := make([]*p4.Update, 1)
	singleUpdate := &p4.Update{Type: p4.Update_INSERT, Entity: entity}

	fmt.Printf("%v\n", singleUpdate)
	updates[0] = singleUpdate
	_, errw := client.Write(context.Background(),
		&p4.WriteRequest{DeviceId: deviceId,
			RoleId:  roleId,
			Updates: updates})
	if errw != nil {
		log.Fatalf("Error sending table entry to rt agent in switch")
	}
}

func deleteMacMapping(tableName string, nhIP string) {
	fmt.Println("Controller: Delete mac binding")
	tableId := p4InfoUtils.GetTableIdFromName(&p4Info, tableName)
	matchIdnhIP := p4InfoUtils.GetMatchIdInTable(&p4Info, tableId, "routing_metadata.nexthop_ipv4")

	var tableEntry p4.TableEntry
	tableEntry.TableId = tableId
	matchList := make([]*p4.FieldMatch, 1)
	ExactMatch := p4.FieldMatch_Exact{Value: getByteArrFromHexStr(nhIP)}
	matchList[0] = &p4.FieldMatch{FieldId: matchIdnhIP,
		FieldMatchType: &p4.FieldMatch_Exact_{&ExactMatch}}

	// Ship this table entry to the RT agent in the switch
	tabEntity := &p4.Entity_TableEntry{TableEntry: &tableEntry}
	entity := &p4.Entity{Entity: tabEntity}
	updates := make([]*p4.Update, 1)
	singleUpdate := &p4.Update{Type: p4.Update_DELETE, Entity: entity}
	fmt.Printf("%v\n", singleUpdate)
	updates[0] = singleUpdate
	_, errw := client.Write(context.Background(),
		&p4.WriteRequest{DeviceId: deviceId,
			RoleId:  roleId,
			Updates: updates})
	if errw != nil {
		log.Fatalf("Error sending table entry to rt agent in switch")
	}
}

// Play out the operations in the json file.
func playTableOperations(opInfoPtr string) {
	fmt.Println("Controller: Parsing table operations")
	operations := getOperations(opInfoPtr)
	reader := bufio.NewReader(os.Stdin)

	for _, operation := range operations.Table {
		// fmt.Println(operation.Name, operation.Prefix,
		//			operation.Plen, operation.Nexthop, operation.Port)
		fmt.Printf("Controller: Do operation: %v\n", operation)
		switch operation.Op {
		case "insert":
			switch operation.Name {
			case "ipv4_lpm":
				insertPrefix(operation.Name,
					operation.Prefix,
					int32(operation.Plen),
					operation.Nexthop,
					operation.Port)
			case "forward":
				insertMacMapping(operation.Name,
					operation.NhIP,
					operation.MacAddr)
			default:
				fmt.Println("Unsupported table")
			}
		case "delete":
			switch operation.Name {
			case "ipv4_lpm":
				deletePrefix(operation.Name,
					operation.Prefix,
					int32(operation.Plen))
			case "forward":
				deleteMacMapping(operation.Name,
					operation.NhIP)
			default:
				fmt.Println("Unsupported table")
			}
		default:
			fmt.Println("Unsupported operation")
		}
		fmt.Println("\nPress RETURN to continue...")
		reader.ReadString('\n')
	}
}
