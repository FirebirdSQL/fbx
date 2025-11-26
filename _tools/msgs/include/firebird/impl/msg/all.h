/*
 *  The contents of this file are subject to the Initial
 *  Developer's Public License Version 1.0 (the "License");
 *  you may not use this file except in compliance with the
 *  License. You may obtain a copy of the License at
 *  http://www.ibphoenix.com/main.nfs?a=ibphoenix&page=ibp_idpl.
 *
 *  Software distributed under the License is distributed AS IS,
 *  WITHOUT WARRANTY OF ANY KIND, either express or implied.
 *  See the License for the specific language governing rights
 *  and limitations under the License.
 *
 *  The Original Code was created by Adriano dos Santos Fernandes
 *  for the Firebird Open Source RDBMS project.
 *
 *  Copyright (c) 2021 Adriano dos Santos Fernandes <adrianosf@gmail.com>
 *  and all contributors signed below.
 *
 *  All Rights Reserved.
 *  Contributor(s): ______________________________________.
 */

// Include headers by their facility order (see firebird/impl/msg_helper.h)
#include "jrd.h"
#include "jrd_rdb.h"
#include "gfix.h"
#include "gfix_rdb.h"
#include "dsql.h"
#include "dsql_rdb.h"
#include "dyn.h"
#include "dyn_rdb.h"
#include "gbak.h"
#include "gbak_rdb.h"
#include "sqlerr.h"
#include "sqlerr_rdb.h"
#include "sqlwarn.h"
#include "jrd_bugchk.h"
#include "jrd_bugchk_rdb.h"
#include "isql.h"
#include "isql_rdb.h"
#include "gsec.h"
#include "gsec_rdb.h"
#include "gstat.h"
#include "gstat_rdb.h"
#include "fbsvcmgr.h"
#include "utl.h"
#include "nbackup.h"
#include "nbackup_rdb.h"
#include "fbtracemgr.h"
#include "repldiff.h"
