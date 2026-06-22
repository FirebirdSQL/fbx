#ifndef _yacc_defines_keywords
#define _yacc_defines_keywords

#define TOK_RANGE_NUMBER 257
#define TOK_INTEGER32 258
#define TOK_INTEGER64 259
#define TOK_DOUBLE 260
#define TOK_JSON_NULL 261
#define TOK_TRUE 262
#define TOK_FALSE 263
#define TOK_STRING 264
#define TOK_QUOTED_STRING 265
#define TOK_PASSING_NAME 266
#define TOK_LAX 267
#define TOK_STRICT 268
#define TOK_LAST 269
#define TOK_TO 270
#define TOK_KEYVALUE 271
#define TOK_DATETIME 272
#define TOK_DECIMAL 273
#define TOK_METHOD 274
#define TOK_STARTS 275
#define TOK_WITH 276
#define TOK_LIKE_REGEX 277
#define TOK_FLAGS 278
#define TOK_IS 279
#define TOK_UNKNOWN 280
#define TOK_EXISTS 281
#define TOK_AND 282
#define TOK_OR 283
#define TOK_EQ 284
#define TOK_NEQ 285
#define TOK_LEQ 286
#define TOK_GEQ 287
#define TOK_UMINUS 288
#define TOK_UPLUS 289
#define YYERRCODE 256

#endif /* _yacc_defines_keywords */

#ifndef _yacc_defines_yystype
#define _yacc_defines_yystype

typedef union
{
	bool boolVal;
	SLONG int32Val;
	SINT64 int64Val;
	double doubleVal;
	Firebird::string* stringPtr;
	struct DecimalPair
	{
		FBJSON::JsonExprNode* scale;
		FBJSON::JsonExprNode* precision;
	} pairVal;

	FBJSON::RangeSize rangeVal;
	FBJSON::PathMethod methodType;
	FBJSON::JsonExprNode* exprNode;
	FBJSON::PathVariable* variable;
	FBJSON::FilterNode* filterNode;
	FBJSON::JsonExprOperation exprOperation;
	FBJSON::PathVariable::Type variableType;
	FBJSON::PathVariable::Position passingPos;
	FBJSON::JsonPathContext* jpathContext;
} YYSTYPE;

#endif /* _yacc_defines_yystype */

