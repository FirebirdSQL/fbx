#define FB_UDR_STATUS_TYPE ::Firebird::ThrowStatusWrapper

#include "ibase.h"
#include "firebird/UdrCppEngine.h"
#include <unistd.h>

using namespace Firebird;

//g++ -o libfbxci.so -shared -fPIC ./udrs.cpp -I../_tools/msgs/include

//recreate function fbsleep (delay float not null) returns float external name 'fbxci!fbsleep' engine udr;
FB_UDR_BEGIN_FUNCTION(fbsleep)
	FB_UDR_MESSAGE(InMessage,
		(FB_FLOAT, delay)
	);
	FB_UDR_MESSAGE(OutMessage,
		(FB_FLOAT, delay)
	);
	FB_UDR_EXECUTE_FUNCTION
	{
		usleep(in->delay * 1000000);
		out->delayNull = FB_FALSE;
		out->delay = in->delay;
	};
FB_UDR_END_FUNCTION

//recreate procedure fbsleep (delay float not null) returns(o float) external name 'fbxci!fbsleep' engine udr;
FB_UDR_BEGIN_PROCEDURE(fbsleep)
	FB_UDR_MESSAGE(InMessage,
		(FB_FLOAT, delay)
	);
	FB_UDR_MESSAGE(OutMessage,
		(FB_FLOAT, delay)
	);
	FB_UDR_EXECUTE_PROCEDURE
	{
		out->delayNull = FB_FALSE;
		fetched = false;
	};
	FB_UDR_FETCH_PROCEDURE
	{
	    if (fetched)
		    return false;
		usleep(in->delay * 1000000);
		out->delay = in->delay;
		fetched = true;
		return true;
	};
	bool fetched;
FB_UDR_END_PROCEDURE

FB_UDR_IMPLEMENT_ENTRY_POINT
