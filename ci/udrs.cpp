#define FB_UDR_STATUS_TYPE ::Firebird::ThrowStatusWrapper

#include "ibase.h"
#include "firebird/UdrCppEngine.h"
#include <unistd.h>

using namespace Firebird;

//g++ -o libfbxci.so -shared -fPIC ./udrs.cpp -I../_tools/msgs/include
//recreate function fbsleep (delay int not null) returns int external name 'fbxci!fbsleep' engine udr;
FB_UDR_BEGIN_FUNCTION(fbsleep)
	FB_UDR_MESSAGE(InMessage,
		(FB_INTEGER, delay)
	);
	FB_UDR_MESSAGE(OutMessage,
		(FB_INTEGER, delay)
	);
	FB_UDR_EXECUTE_FUNCTION
	{
		sleep(in->delay);
		out->delayNull = FB_FALSE;
		out->delay = in->delay;
	};
FB_UDR_END_FUNCTION

FB_UDR_IMPLEMENT_ENTRY_POINT
