# P4Runtime with OFA prototype

This is a demo repository to integrate P4Runtime with Cisco OFA(Open Forwarding Abstraction)
## Architecture
## Code layout
src/rtagent      : Code running in the switch/network element, which acts as the gRPC server. This code also includes the glue, which                        binds P4Runtime to the OFA implementation.

src/controller   : A test controller, which acts as the gRPC client. This drives the RTAgent that runs in the network element.

src/p4InfoUtils  : Some utilities for cross-mapping

ios-xr           : Code that implements the OFA chains.

## Running this code
1. Clone the repo:
```
git clone https://github.com/saramach/P4Runtime.git
```
2. Fetch some dependencies (TBD: Script this)
```
go get github.com/golang/protobuf/proto
go get github.com/golang/protobuf/ptypes/any
go get golang.org/x/net/context
go get google.golang.org/genproto/googleapis/rpc/status
go get google.golang.org/grpc
go get google.golang.org/grpc/reflection
```
3. Build the controller and the rtagent (TBD: 'make' this)
```
go install rtagent
go install controller
```
This will place the 'rtagnet' and 'controller' binaries in the bin/ directory

4. Start the rtagent and then the controller. 
```
[xr-vm_node0_0_CPU0:/]$./controller -p4info=./simple_router-p4info.json -operations=./operations.json
```
