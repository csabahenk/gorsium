#include <string.h>
#include <unistd.h>
#include <stdlib.h>
#include "compat.h"
#include "byteorder.h"
#include "mdigest.h"
#include "md5_2.h"

void get_md5_2(uchar *out, const uchar *input1, int n1, const uchar *input2, int n2)
{
#ifdef MDEBUG
	char *dbgs = NULL;
	int dbg = 0;

	dbgs = getenv("MDEBUG");
	if (dbgs && strcmp(dbgs, "1") == 0)
		dbg = 1;
	if (dbg) {
		write(2, out, 16);
		write(2, input1, n1);
		write(2, input2, n2);
	}
#endif

	md_context ctx;
	md5_begin(&ctx);
	md5_update(&ctx, input1, n1);
	md5_update(&ctx, input2, n2);
	md5_result(&ctx, out);

#ifdef MDEBUG
	if (dbg)
		write(2, out, 16);
#endif
}


