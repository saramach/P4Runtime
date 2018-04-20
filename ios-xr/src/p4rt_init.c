// Code needed to work in the XR environment
//
#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>

#include <infra/event_manager.h>
#include <platforms/common/bcm-dpa/tables/bcmdpa_tables_api.h>
#include <platforms/common/bcm-dpa/dpa_global.h>

#define P4RT_BIND_ATTEMPTS  (4)
int
p4rt_ofa_intf_setup ()
{
    event_mgr_p     p4rt_evm;
    dpa_npu_mask_t  npu_mask;
    int             rc = 0;
    const char      *interested_tables[] = {DPA_GLOBAL_STR};

    p4rt_evm = event_manager_create(0, "p4rt_thin_client", 0, NULL);
    if (p4rt_evm == NULL) {
        // Error creating the event manager.
        printf("Error setting up event manager..\n");
        return -1;
    }

    // Need to bind to the OFA server. For two reasons:
    // 1. Let OFA server know about the existance of such a client.
    // 2. Bind to the GRID. This is needed for resource allocation
    //    like the ipnh encap-id.
    int retry_attempts = 0;
    int bound = 0;
    while (retry_attempts < P4RT_BIND_ATTEMPTS) {
        rc = bcmdpa_table_bind("p4rt_thin_client",
                               1,
                               interested_tables,
                               &npu_mask,
                               p4rt_evm,
                               NULL, NULL);
        printf("Bind returned 0x%x...\n", rc);
        if (rc == EAGAIN) {
            retry_attempts++;
        } else {
            bound = 1;
            break;
        }
    }
    if (!bound) {
        printf("Error binding to the OFA server after enough "
               "retries.. Can't proceed further");
        return -1;
    }


        
    return 0;
}
