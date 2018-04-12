#include <stdio.h>
#include <stdlib.h>
#include <dlfcn.h>
#include "_cgo_export.h"

/* Some stubs to get over undefined symbol errors */
int _sal_assert(int i) { return 0;}
int sal_config_set(int i) { return 0;}
int cidl_inst_sync_mfl_free_ext(int i) { return 0;}  
int cidl_inst_sync_mfl_encode_ext(int i) { return 0;}
int cidl_inst_sync_mfl_duplicate(int i) { return 0;}
int cidl_inst_sync_mfl_decode_ext(int i) { return 0;}
int fia_cfg_profile_get_otn_only_200g(int i) { return 0;}
int pkg_verify_ignore_file_list(int i) { return 0;}
int instcomm_log(int i) { return 0;} 
int instdir_get_all_nodes(int i) { return 0;}
int pkg_api_consumer_gen_mismatched_report(int i) { return 0;}
int pkg_api_consumer_gen_exempted_report(int i) { return 0;}
int instdir_inv_get_lrs_from_nodeids(int i) { return 0;}
int pkg_package_get_node_types_from_pkg_version(int i) { return 0;}
int pkg_api_consumer_gen_unresolved_report(int i) { return 0;}
int nodeid_instance_frm_encoded_inst(int i) { return 0;}

typedef int (*createRoute)(unsigned int, unsigned int, unsigned int, unsigned int, unsigned char*);
typedef int (*deleteRoute)(unsigned int, unsigned int);

void *getDLLHandle() {
	dlerror();
	void *xrP4impl = dlopen("libxr_p4impl.so", RTLD_NOW);
	if (xrP4impl == NULL) {
		printf("Error opening IOS XR P4RT implementation: %s\n", dlerror());
		return NULL;
	} else {
		printf("Successfully opened IOS XR P4RT implementation\n");
	}

	return xrP4impl;
}

int createOFATableEntry(char *tableName, 
						unsigned int prefix,
						unsigned int prefixLen,
						unsigned int nextHop,
						unsigned int outPort,
						void *dMac) {

	unsigned char *destMac = (unsigned char *)dMac;
	printf("OFA: Received Table entry add for:\n"
			"\tTable  : %s\n"
			"\tPrefix : %x\n"
			"\tPfxLen : %x\n"
			"\tNexthop: %x\n"
			"\tDport  : %x\n"
			"\tdmacP  : %p\n"
			"\tdestmac: %x:%x:%x:%x:%x:%x\n",
			tableName,
			prefix,
			prefixLen,
			nextHop,
			outPort,
			destMac,
			destMac[0], destMac[1], destMac[2], destMac[3], destMac[4], destMac[5]);

	printf("Trying to open XRs' P4RT implementation...\n");
	void *xrP4impl = getDLLHandle();
	if (xrP4impl == NULL) return -1;

    // Get hold of the route entry add implementor in the libraty.
	dlerror();
	createRoute createRoute_func;
	createRoute_func = (createRoute)dlsym(xrP4impl, "ofa_intf_route_entry_add");
	if (createRoute_func == NULL) {
		printf("Error finding the OFA function implementing route add\n");
		return -1;
	}
	int ret = (*createRoute_func)(prefix, prefixLen, nextHop, outPort, destMac);
	printf("Route entry create returned: %d\n", ret);

	return 0;
}

int deleteOFATableEntry(char *tableName,
						unsigned int prefix,
						unsigned int prefixLen) {

	printf("OFA: Received Table entry delete for:\n"
			"\tTable  : %s\n"
			"\tPrefix : %x\n"
			"\tPfxLen : %x\n",
			tableName,
			prefix,
			prefixLen);

	void *xrP4impl = getDLLHandle();
	if (xrP4impl == NULL) return -1;
    // Get hold of the route entry add implementor in the libraty.
	dlerror();
	deleteRoute deleteRoute_func;
	deleteRoute_func = (deleteRoute)dlsym(xrP4impl, "ofa_intf_route_entry_delete");
	if (deleteRoute_func == NULL) {
		printf("Error finding the OFA function implementing route delete\n");
		return -1;
	}
	int ret = (*deleteRoute_func)(prefix, prefixLen);

	return 0;
}

