// Glue code to implement simple_router.p4
// For now, this code strictly binds to the P4 program. Just for the PoC
// For general use, this binding has to be removed.

package main

import (
	"encoding/binary"
	"errors"
	"fmt"
	"p4"
	"p4InfoUtils"
	"strings"
	"unsafe"
)

/*
#cgo LDFLAGS: -ldl
#include <stdlib.h>
extern int createOFATableEntry(char *tableName,
						unsigned int prefix,
						unsigned int prefixLen,
						unsigned int nextHop,
						unsigned int outPort,
						void	*destMac);
extern int deleteOFATableEntry(char *tableName,
						unsigned int prefix,
						unsigned int prefixLen);
*/
import "C"

type prefixData struct {
	prefix uint32
	plen   uint32
}
type nexthopData struct {
	nhPrefix uint32
	port     uint32
}

var routeMap = map[prefixData]nexthopData{}
var arpMap = map[uint32][]byte{}

// Convert a byte array of size 4 to an unsigned 32-bit integer
// Example:
// [ff ff ff ff] -> 0xffffffff
// [ab ab ab ab] -> 0xabababab
// [1 2 3 4]     -> 0x01020304
func getUint32FromByteArr(byteArr []byte) uint32 {
	if len(byteArr) == 4 {
		return binary.BigEndian.Uint32(byteArr)
	}
	return 0
}

func routeInsert(tableEntry *p4.TableEntry) error {
	var pfxData prefixData
	var nhData nexthopData

	tableName := p4InfoUtils.GetTableNameFromId(&p4Info, tableEntry.TableId)
	for _, match := range tableEntry.GetMatch() {
		switch match.GetFieldMatchType().(type) {
		case *p4.FieldMatch_Lpm:
			lpmMatch := match.GetLpm()
			pfxData.prefix = getUint32FromByteArr(lpmMatch.GetValue())
			pfxData.plen = uint32(lpmMatch.GetPrefixLen())
		default:
			fmt.Println("Unsupported match type")
			return errors.New("Unsupported match type")
		}
	}

	tabAction := tableEntry.GetAction()
	switch tabAction.GetType().(type) {
	case *p4.TableAction_Action:
		action := tabAction.GetAction()
		actionId := action.GetActionId()
		var paramName string
		for _, param := range action.GetParams() {
			paramName = p4InfoUtils.GetParamNameInAction(
				&p4Info,
				actionId,
				param.GetParamId())
			if strings.Compare(paramName, "nhop_ipv4") == 0 {
				nhData.nhPrefix = getUint32FromByteArr(param.GetValue())
			} else if strings.Compare(paramName, "port") == 0 {
				nhData.port = getUint32FromByteArr(param.GetValue())
			}
		}
	default:
		fmt.Println("Unsupported action type")
		return errors.New("Unsupported action type")
	}

	routeMap[pfxData] = nhData
	fmt.Printf("%v\n", routeMap)

	// For the destination IP, get the DMAC from the MAC binding table.
	destMac, ok := arpMap[nhData.nhPrefix]
	fmt.Println("DMAC for the nexthop IP is: ", destMac)
	if ok {
		fmt.Println("DMAC for the nexthop IP is: ", destMac)
		// Relay it to OFA
		tabName := C.CString(tableName)
		dMacRelay := C.CBytes(destMac)
		// NOTE: These gets allocated from the heap. Defer freeing this
		// towards the end.
		defer C.free(unsafe.Pointer(tabName))
		defer C.free(unsafe.Pointer(dMacRelay))
		C.createOFATableEntry(tabName, C.uint(pfxData.prefix),
			C.uint(pfxData.plen), C.uint(nhData.nhPrefix),
			C.uint(nhData.port), dMacRelay)
	} else {
		fmt.Println("DMAC for the nexthop IP ", destMac, " NOT found")
		return errors.New("Destination mac not resolved for the nexthop IP")
	}

	return nil
}

func routeDelete(tableEntry *p4.TableEntry) error {
	var pfxData prefixData

	tableName := p4InfoUtils.GetTableNameFromId(&p4Info, tableEntry.TableId)
	for _, match := range tableEntry.GetMatch() {
		switch match.GetFieldMatchType().(type) {
		case *p4.FieldMatch_Lpm:
			lpmMatch := match.GetLpm()
			pfxData.prefix = getUint32FromByteArr(lpmMatch.GetValue())
			pfxData.plen = uint32(lpmMatch.GetPrefixLen())
		default:
			fmt.Println("Unsupported match type")
			return errors.New("Unsupported match type")
		}
	}

	// Relay the delete to the OFA layer.
	tabName := C.CString(tableName)
	defer C.free(unsafe.Pointer(tabName))
	C.deleteOFATableEntry(tabName,
		C.uint(pfxData.prefix),
		C.uint(pfxData.plen))

	delete(routeMap, pfxData)
	fmt.Printf("%v\n", routeMap)

	return nil
}

func handleRouteEntry(tableEntry *p4.TableEntry, op p4.Update_Type) error {
	switch op {
	case p4.Update_INSERT:
		return routeInsert(tableEntry)
	case p4.Update_MODIFY:
		fmt.Println("Unsupported operation")
		return errors.New("Unsupported table operation")
	case p4.Update_DELETE:
		return routeDelete(tableEntry)
	default:
		fmt.Println("Unsupported operation")
		return errors.New("Unsupported table operation")
	}

	return nil
}

func macBindInsert(tableEntry *p4.TableEntry) error {
	var nhIP []byte
	var macAddress []byte
	for _, match := range tableEntry.GetMatch() {
		switch match.GetFieldMatchType().(type) {
		case *p4.FieldMatch_Exact_:
			ExactMatch := match.GetExact()
			nhIP = ExactMatch.GetValue()
		default:
			fmt.Println("Unsupported match type")
			return errors.New("Unsupported match type")
		}
	}

	tabAction := tableEntry.GetAction()
	switch tabAction.GetType().(type) {
	case *p4.TableAction_Action:
		action := tabAction.GetAction()
		actionId := action.GetActionId()
		var paramName string
		// We expect to see only a destination MAC address
		for _, param := range action.GetParams() {
			paramName = p4InfoUtils.GetParamNameInAction(
				&p4Info,
				actionId,
				param.GetParamId())
			if strings.Compare(paramName, "dmac") == 0 {
				macAddress = param.GetValue()
			} else {
				fmt.Println("Unsupported paramater in mac binding")
				return errors.New("Unsupported paramater in mac binding")
			}
		}
	default:
		fmt.Println("Unsupported action type")
		return errors.New("Unsupported action type")
	}

	arpMap[getUint32FromByteArr(nhIP)] = macAddress
	fmt.Printf("%v\n", arpMap)

	return nil
}

func macBindDelete(tableEntry *p4.TableEntry) error {
	var nhIP []byte
	for _, match := range tableEntry.GetMatch() {
		switch match.GetFieldMatchType().(type) {
		case *p4.FieldMatch_Exact_:
			ExactMatch := match.GetExact()
			nhIP = ExactMatch.GetValue()
		default:
			fmt.Println("Unsupported match type")
			return errors.New("Unsupported match type")
		}
	}

	delete(arpMap, getUint32FromByteArr(nhIP))
	fmt.Printf("%v\n", arpMap)

	return nil
}

func handleMacBinding(tableEntry *p4.TableEntry, op p4.Update_Type) error {
	switch op {
	case p4.Update_INSERT:
		return macBindInsert(tableEntry)
	case p4.Update_MODIFY:
	case p4.Update_DELETE:
		return macBindDelete(tableEntry)
	default:
		fmt.Println("Unsupported operation")
		return errors.New("Unsupported table operation")
	}

	return nil

}

func handleTableOperation(tableEntry *p4.TableEntry, op p4.Update_Type) error {
	tableName := p4InfoUtils.GetTableNameFromId(&p4Info, tableEntry.TableId)
	fmt.Println("Table operation for :", tableName)
	switch tableName {
	case "ipv4_lpm":
		return handleRouteEntry(tableEntry, op)
	case "forward":
		return handleMacBinding(tableEntry, op)
	default:
		fmt.Println("Unsupported table")
		return errors.New("Unsupported table")
	}
}
