package  main

import (
	"log"
	"fmt"
	"os"
	p4_config "p4/config"
	"github.com/golang/protobuf/jsonpb"
)

func main() {
	file, err := os.Open("./jsonfile.json")
	if  err != nil {
		log.Fatalln(err)
	}
	var p4Info p4_config.P4Info
	jerr := jsonpb.Unmarshal(file, &p4Info)
	if jerr != nil {
		fmt.Println("Error unmarshalling json data")
	} else {
		fmt.Println("Worked %+v", p4Info)
	}
}
