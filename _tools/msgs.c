#include <stdint.h>

#include "firebird/impl/msg_helper.h"

extern void addMessage(int code, const char* message);

typedef unsigned short USHORT;
typedef USHORT ISC_USHORT;
typedef intptr_t ISC_STATUS;
typedef long SLONG;

#define stringify_literal(x) #x

#define FB_IMPL_MSG_NO_SYMBOL(facility, number, text)

#define FB_IMPL_MSG_SYMBOL(facility, number, symbol, text)

#define FB_IMPL_MSG(facility, number, symbol, sqlCode, sqlClass, sqlSubClass, text) \
    addMessage(make_isc_code(FB_IMPL_MSG_FACILITY_##facility, number), stringify_literal(text));

#define ISC_CODE 0x14000000

int make_isc_code(int facility, int code) {
    return ISC_CODE | ((facility & 0x1F) << 16) | (code & 0x3FFF);
}

void process_messages() {
    #include "firebird/impl/msg/all.h"
}