#include <stdio.h>
#include <stdlib.h>

extern int 
ofa_intf_table_entry_add(uint32_t ipPrefix, 
                         uint32_t prefixLen,
                         uint32_t ipNextHop,
                         uint32_t destPort, 
                         unsigned char *destMac);
int main() 
{
    uint32_t ipPrefix, prefixLen, ipNextHop, destPort;
    unsigned char destMac[6];
    printf("p4rt test program\n");
    
    ipPrefix= 0xc8000000;
    prefixLen = 24;
    ipNextHop = 0x0f010102;
    destPort = 0x1d0;
    destMac[0] = 0x00;
    destMac[1] = 0x8a;
    destMac[2] = 0x96;
    destMac[3] = 0x94;
    destMac[4] = 0xb0;
    destMac[5] = 0x9c;

    ofa_intf_table_entry_add(ipPrefix, prefixLen, ipNextHop, destPort, &destMac[0]);
    return 0;

}

