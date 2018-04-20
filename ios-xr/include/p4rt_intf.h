#ifndef __P4RT_INTF_H__
#define __P4RT_INTF_H__

extern int
ofa_intf_table_entry_add(uint32_t ipPrefix,
                         uint32_t prefixLen,
                         uint32_t ipNextHop,
                         uint32_t destPort,
                         unsigned char *destMac);

#define P4RT_TRY(expr, label) \
    if ((expr) == 0) goto label

#define P4RT_CATCH(label) \
    label

#endif // __P4RT_INTF_H__
