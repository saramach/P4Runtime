/*
 * p4rt_intf.c
 * OFA - P4Runtime interface
 */
#include <stdlib.h>
#include <unistd.h>
#include <sys/types.h>
#include <arpa/inet.h>
#include <platforms/common/bcm-dpa/bcmdpa.h>
#include <platforms/common/bcm-dpa/dpa_cerrno.h>
#include <platforms/common/bcm-dpa/dpa_iproute.h>
#include <platforms/common/bcm-dpa/dpa_ipnhgroup.h>
#include <platforms/common/bcm-dpa/dpa_ipnh.h>
#include <platforms/common/rid_mgr/grid_api.h>
#include "p4rt_intf.h"


uint64_t nhGrpid = 2147487358ULL;
int      xr_init_done = 0;

int
ofa_intf_ipnh_get_create (uint32_t  ipnhAddr,
                          bcmdpa_mac_t nhMac,
                          uint32_t  port,
                          bcmdpa_table_obj_handle_t   *ipnhObjHandle)
{
    dpa_ipnh_t      ipnhObj;
    dpa_ipnh_key_t  ipnhKey;
    uint8_t     found = 0;
    int         rc;

    printf("%s: ipnhAddr: 0x%x\n", __func__, ipnhAddr);
    // Check if there is already a nexthop 
    // over the interface. If there is one,
    // just use that. Else, create a new
    // ip nexthop with the specified mac
    // address.
    //
    // Right thing to to do:
    // dpa_ipnh_t ipnhObj;
    // dpa_ipnh_key_t  ipnhKey;
    // bzero(&ipnhKey, sizeof(ipnhKey));
    // ipnhKey.intf = destPort;
    // ipnhKey.nh_addr = (uint32_t)strtoul(ipNextHop, NULL, 16);
    // dpa_ipnh_get(obj);
    ipnhObj = dpa_ipnh_ctor();

    bzero(&ipnhKey, sizeof(ipnhKey));
    dpa_ipnh_set_key(ipnhObj, ipnhKey);

    rc = dpa_ipnh_get_first(ipnhObj);
    while (CERR_IS_OK(rc)) {
        printf("Checking NH with nhAddr 0x%x\n", htonl(dpa_ipnh_get_nh_addr(ipnhObj)));
        printf("ipnhKey:\n");
        printf("\tnhaddr      : 0x%x\n", htonl(dpa_ipnh_get_nh_addr(ipnhObj)));
        printf("\tinterface   : 0x%x\n", dpa_ipnh_get_intf(ipnhObj));
        printf("\tproto       : %d\n", dpa_ipnh_get_proto(ipnhObj));
        printf("\tmpls proto  : %d\n", dpa_ipnh_get_mpls_proto(ipnhObj));

        if  (htonl(dpa_ipnh_get_nh_addr(ipnhObj)) == ipnhAddr) {
            found = 1;
            break;
        }
        rc = dpa_ipnh_get_next(ipnhObj);
    }

    // If a matching nexthop is found, set the handle to the
    // nexthop and return.
    //
    if (found == 1) {
        *ipnhObjHandle = dpa_ipnh_get_hdl(ipnhObj);
        dpa_ipnh_dtor(ipnhObj);
        return EOK;
    }

    // The nexthop was not found. We'll need to create one.
    printf("Could not find the nexthop. Proceeding to create one.\n");
    // Start clean
    dpa_ipnh_dtor(ipnhObj);
    ipnhObj = dpa_ipnh_ctor();
    *ipnhObjHandle = NULL;
    
    // Derive the port parameters
    // Port related variable declarations.
    
    uint8_t  unit       = 0;
    uint16_t pp_port_id = 0, local_port_id = 0, system_port_id = 0, rif = 0;
    uint32_t lif        = 0;
    void     *l3_intf_handle = NULL, *system_port_handle = NULL;

    rc = ifh_2_port_id(port,
                       &unit,
                       &pp_port_id,
                       &local_port_id,
                       &system_port_id);
    P4RT_TRY(CERR_IS_OK(rc), ERR_IFH_2_PORT_ID); 

    rc = ifh_2_rif(port, &rif, &l3_intf_handle);
    P4RT_TRY(CERR_IS_OK(rc), ERR_IFH_2_RIF); 

    rc = ifh_2_lif(port, &lif, &system_port_handle);
    P4RT_TRY(CERR_IS_OK(rc), ERR_IFH_2_LIF); 

    // Set the object attributes.
    bzero(&ipnhKey, sizeof(ipnhKey));
    ipnhKey.intf    = port;
    ipnhKey.proto   = PROTO_IPV4;
    ipnhKey.nh_addr = htonl(ipnhAddr);

    dpa_ipnh_set_key(ipnhObj, ipnhKey);
    dpa_ipnh_set_intf(ipnhObj, ipnhKey.intf);
    dpa_ipnh_set_nh_addr(ipnhObj, ipnhKey.nh_addr);
    dpa_ipnh_set_proto(ipnhObj, ipnhKey.proto);
    dpa_ipnh_set_mac_addr(ipnhObj, nhMac);
    dpa_ipnh_set_is_local(ipnhObj, 1);
    // How to figure out the npu_mask?
    // Set to all the NPUs??
    dpa_ipnh_set_npu_mask(ipnhObj, 0x1);
    dpa_ipnh_set_gport(ipnhObj, system_port_id);
    dpa_ipnh_set_l3intf_refhdl(ipnhObj, l3_intf_handle);
    dpa_ipnh_set_l2port_refhdl(ipnhObj, system_port_handle);
    dpa_ipnh_set_gport(ipnhObj, system_port_id);
    dpa_ipnh_set_complete(ipnhObj, 1);
    dpa_ipnh_set_is_lbdl(ipnhObj, 0);
    dpa_ipnh_set_encap_id_rid_level(ipnhObj, GRID_CLIENT_RES_LEVEL_0);
    dpa_ipnh_set_alloc_sz(ipnhObj, 1);
    rc = dpa_ipnh_create(ipnhObj); 
    P4RT_TRY(((rc & 0xffff) == EOK), ERR_IPNH_CREATE_FAIL);
    *ipnhObjHandle = dpa_ipnh_get_hdl(ipnhObj);
    dpa_ipnh_dtor(ipnhObj);
    printf("ipnh created..\n");

    return EOK;

    P4RT_CATCH(ERR_IFH_2_PORT_ID):
        printf("ifh_2_port_id failed..\n");
        dpa_ipnh_dtor(ipnhObj);
        return -1;

    P4RT_CATCH(ERR_IFH_2_RIF):
        printf("ifh_2_rif failed..\n");
        dpa_ipnh_dtor(ipnhObj);
        return -1;

    P4RT_CATCH(ERR_IFH_2_LIF):
        printf("ifh_2_lif failed..\n");
        dpa_ipnh_dtor(ipnhObj);
        return -1;

    P4RT_CATCH(ERR_IPNH_CREATE_FAIL):
        printf("ipnh creation failed..\n");
        dpa_ipnh_dtor(ipnhObj);
        return -1;
}

int 
ofa_intf_ipnhgroup_get_create (bcmdpa_table_obj_handle_t ipnhObjHandle,
                               bcmdpa_table_obj_handle_t *ipnhGrpObjHandle,
                               dpa_intf_t intf)
{
    dpa_ipnhgroup_t     ipnhgObj;
    dpa_ipnhgroup_key_t ipnhgKey;
    int         rc;
    uint8_t     unit = 0;
    uint16_t    pp_port_id;
    uint16_t    local_port_id;
    uint16_t    system_port_id;

    // The route has to point to a nexthop group. Even if there
    // is no ECMP(loadbalancing), the route has to point to a
    // nexthop group with one ipnh!
    // 1 member for now
    *ipnhGrpObjHandle = NULL;
    ipnhgObj= dpa_ipnhgroup_ctor(1);
    bzero(&ipnhgKey, sizeof(ipnhgKey));
    // Check the nexthop ID
    nhGrpid++;
    memcpy(&ipnhgKey.nhgroup_id, &nhGrpid, sizeof(nhGrpid));
    dpa_ipnhgroup_set_key(ipnhgObj, ipnhgKey);
    dpa_ipnhgroup_set_num_primary_paths(ipnhgObj, 1);

    rc = ifh_2_port_id(intf,
                       &unit,
                       &pp_port_id,
                       &local_port_id,
                       &system_port_id);
    P4RT_TRY((CERR_IS_OK(rc)), ERR_IFH_2_PORT_ID);

    printf("%s: system_port_id: 0x%x\n", __func__, system_port_id);
    dpa_ipnhgroup_set_port(ipnhgObj, 0, system_port_id);
    dpa_ipnhgroup_set_ipnh_refhdl(ipnhgObj, 0, ipnhObjHandle);
    rc = dpa_ipnhgroup_create(ipnhgObj);
    printf("Creating ipnhg returned 0x%x\n", rc);
    P4RT_TRY(((rc & 0xffff) == EOK), ERR_IPNHGRP_CREATE_FAILED);
    *ipnhGrpObjHandle = dpa_ipnhgroup_get_hdl(ipnhgObj);
    printf("Created IP nexthop group\n");

    return EOK;

    P4RT_CATCH(ERR_IFH_2_PORT_ID):
        printf("Error creating IP nexthop group\n");
        dpa_ipnhgroup_dtor(ipnhgObj);
        return -1;

    P4RT_CATCH(ERR_IPNHGRP_CREATE_FAILED):
        printf("ipnhgroup create failed..");
        dpa_ipnhgroup_dtor(ipnhgObj);
        return -1;

    return EOK;
}

int 
ofa_intf_route_entry_delete (uint32_t ipPrefix,
                             uint32_t prefixLen)
{
    dpa_iproute_t       obj;
    dpa_iproute_key_t   key;

    dpa_ipnhgroup_t     ipnhgObj;
    dpa_ipnhgroup_key_t ipnhgKey;

    int rc;

    bzero(&key, sizeof(key));
    key.ip_addr = htonl(ipPrefix);
    key.ip_mask = htonl(~((1 << (32 - prefixLen)) - 1));
    printf("%s: Prefix addr: 0x%x, mask: 0x%x\n", __func__, key.ip_addr, key.ip_mask);

    // Check if the object is present in the DB.
    obj = dpa_iproute_ctor();
    dpa_iproute_set_key(obj, key);
    rc = dpa_iproute_get(obj);
    P4RT_TRY(CERR_IS_OK(rc), IPROUTE_NOT_FOUND);
    ipnhgKey = dpa_iproute_get_ipnhgroup_refkey(obj);

    // Route found. Delete the route and the associated
    // ipnhGroup.
    // Delete the route first.
    rc = dpa_iproute_delete(obj);
    P4RT_TRY(CERR_IS_OK(rc), IPROUTE_DELETE_FAILED);

    // Delete the ip nexthop group
    ipnhgObj= dpa_ipnhgroup_ctor(1);
    dpa_ipnhgroup_set_key(ipnhgObj, ipnhgKey);
    rc = dpa_ipnhgroup_delete(ipnhgObj);
    P4RT_TRY(CERR_IS_OK(rc), IPNHG_DELETE_FAILED);

    // Cleanup
    dpa_iproute_dtor(obj);
    dpa_ipnhgroup_dtor(ipnhgObj);

    return EOK;


    P4RT_CATCH(IPROUTE_NOT_FOUND):
        printf("%s: Route object not in DB..\n", __func__);
        return EOK;

    P4RT_CATCH(IPROUTE_DELETE_FAILED):
        dpa_iproute_dtor(obj);
        printf("%s: Route object delete failed..\n", __func__);
        return -1;

    P4RT_CATCH(IPNHG_DELETE_FAILED):
        dpa_iproute_dtor(obj);
        dpa_ipnhgroup_dtor(ipnhgObj);
        printf("%s: IP Nexthopgroup object delete failed..\n", __func__);
        return -1;
}

int
ofa_intf_route_entry_add (uint32_t ipPrefix,
                          uint32_t prefixLen,
                          uint32_t ipNextHop,
                          uint32_t destPort,
                          unsigned char *destMac)
{
    dpa_iproute_t       obj;
    dpa_iproute_key_t   key;
    dpa_ipnh_t          ipnhObj;
    dpa_ipnhgroup_t     ipnhgObj;
    bcmdpa_table_obj_handle_t   ipnhObjHandle;
    bcmdpa_table_obj_handle_t   ipnhgObjHandle;
    bcmdpa_mac_t        nhMac;
    int         rc;
    printf("%s: Prefix: %x, PfxLen: %x, Nexthop: %x, Destport: %x, macP: %p "
            "destMac: %x:%x:%x:%x:%x:%x",
           __func__, ipPrefix, prefixLen, ipNextHop, destPort, destMac,
           destMac[0], destMac[1], destMac[2],
           destMac[3], destMac[4], destMac[5]);

    if (xr_init_done == 0) {
        rc = p4rt_ofa_intf_setup();
        P4RT_TRY(CERR_IS_OK(rc), ERR_OFA_SETUP_FAILED);
        xr_init_done = 1;
        printf("%s: IOS-XR environment successfully setup..\n", __func__);
    }

    memcpy(&nhMac.mac, destMac, 6);
    rc = ofa_intf_ipnh_get_create(ipNextHop, nhMac, destPort, &ipnhObjHandle);
    P4RT_TRY(CERR_IS_OK(rc), ERR_IPNH_CREATE_FAILED);

    rc = ofa_intf_ipnhgroup_get_create(ipnhObjHandle, &ipnhgObjHandle, destPort);
    P4RT_TRY(CERR_IS_OK(rc), ERR_IPNHG_CREATE_FAILED);

    // Create the IP Route
    bzero(&key, sizeof(key));
    key.ip_addr = htonl(ipPrefix);
    key.ip_mask = htonl(~((1 << (32 - prefixLen)) - 1));
    printf("%s: Prefix addr: 0x%x, mask: 0x%x\n", __func__, key.ip_addr, key.ip_mask);
    obj = dpa_iproute_ctor();
    dpa_iproute_set_key(obj, key);
    dpa_iproute_set_vrf(obj, key.vrf);
    dpa_iproute_set_ip_addr(obj, key.ip_addr);
    dpa_iproute_set_ip_mask(obj, key.ip_mask);
    dpa_iproute_set_ipnhgroup_refhdl(obj, ipnhgObjHandle);

    rc = dpa_iproute_create(obj);
    printf("Creating iproute returned 0x%x\n", rc);
    P4RT_TRY(CERR_IS_OK(rc), ERR_IPROUTE_CREATE_FAILED);
    printf("Successfully created IP route\n");
    dpa_iproute_dtor(obj);

    // TEST CODE
    obj = dpa_iproute_ctor();
    dpa_iproute_set_key(obj, key);
    rc = dpa_iproute_get(obj);
    P4RT_TRY(CERR_IS_OK(rc), ERR_IPROUTE_GET_FAILED);
    dpa_ipnhgroup_key_t ipnhgKey;
    ipnhgKey = dpa_iproute_get_ipnhgroup_refkey(obj);
    printf("Got NHG key: 0x%x 0x%x 0x%x 0x%x "
           "0x%x 0x%x 0x%x 0x%x \n",
           ipnhgKey.nhgroup_id.id[0], ipnhgKey.nhgroup_id.id[1],
           ipnhgKey.nhgroup_id.id[2], ipnhgKey.nhgroup_id.id[3],
           ipnhgKey.nhgroup_id.id[4], ipnhgKey.nhgroup_id.id[5],
           ipnhgKey.nhgroup_id.id[6], ipnhgKey.nhgroup_id.id[7]);
    dpa_iproute_dtor(obj);


    // END TEST CODE
    return EOK;

    P4RT_CATCH(ERR_IPROUTE_GET_FAILED):
        printf("%s: Error getting iproute object..\n");
        return -1;

    P4RT_CATCH(ERR_OFA_SETUP_FAILED):
        printf("%s: Error setting up the XR environment..\n");
        return -1;

    P4RT_CATCH(ERR_IPNH_CREATE_FAILED):
        printf("Error fetching/creating ipnh matching nexthop address.\n");
        return -1;

    P4RT_CATCH(ERR_IPNHG_CREATE_FAILED):
        // TODO: What to do with the IPNH created?
        printf("Error creating ipnhgroup.\n");
        return -1;

    P4RT_CATCH(ERR_IPROUTE_CREATE_FAILED):
        // TODO: Cleanup the chain created thus far?
        dpa_iproute_dtor(obj);
        printf("Error creating IP route\n");
        return -1;
}
