//
// Created by GuangYu Jing on 2018/7/17.
//

#ifndef TVM_TVM_H
#define TVM_TVM_H

#ifdef __cplusplus
extern "C" {
#endif


void tvm_set_lib_path(const char* path);

typedef enum {
    RETURN_TYPE_INT = 1,
    RETURN_TYPE_STRING,
    RETURN_TYPE_NONE,
    RETURN_TYPE_EXCEPTION,
    RETURN_TYPE_BOOL,
}tvm_return_type_t;

//parse.h mp_parse_input_kind_t
typedef enum {
    PARSE_KIND_SINGLE = 0,
    PARSE_KIND_FILE,
    PARSE_KIND_EVAL,
}tvm_parse_kind_t;

typedef struct _ExecuteResult {
    int resultType; //tvm_return_type_t
    int errorCode;
    char *content;
    char *abi;
}ExecuteResult;
void printResult(ExecuteResult *result);
void initResult(ExecuteResult *result);
void deinitResult(ExecuteResult *result);

void tvm_execute(const char *script, const char *alias, tvm_parse_kind_t parseKind, ExecuteResult *result);


typedef int (*callback_fcn)(int);
typedef void (*testAry_fcn)(void*);

_Bool pycode2bytecode(char *str);
_Bool runbytecode(char *buf, int len);
void some_c_func(callback_fcn);
void tvm_setup_func(callback_fcn callback);
void tvm_set_testAry_func(testAry_fcn);

void tvm_set_lib_line(int line);
typedef void (*TransferFunc)(const char*, const char*);

void setTransferFunc(TransferFunc);

/***********************/

void tvm_start();
void tvm_gc();
void tvm_delete();

void tvm_create_context();
void tvm_remove_context();

/***********************/

void tvm_set_gas(int limit);
int tvm_get_gas();
void tvm_gas_report();

/*********************************************************************************************/
typedef void (*Function1) (const char*);
typedef char* (*Function2) (const char*);
typedef unsigned long long (*Function3) (const char*);
typedef _Bool (*Function4) (const char*);
typedef void (*Function5) (const char*, const char*);
typedef void (*Function6) (const char*, unsigned long long);
typedef int (*Function7) (const char*);
typedef void (*Function8) (unsigned long long);
typedef unsigned long long (*Function9) ();
typedef char* (*Function10) (const char*);
typedef char* (*Function11) (const char*, const char*, const char*);
typedef void (*Function12) (int);
typedef int (*Function13)();
typedef char* (*Function14) (unsigned long long);
typedef char* (*Function15) ();
typedef void (*Function16)(const char*, int len);
typedef void (*Function17) (const char*, const char*, const char*, ExecuteResult *result);


 TransferFunc transferFunc;
 callback_fcn func;
 testAry_fcn testAry;
 Function1 create_account;
 Function5 sub_balance;
 Function5 add_balance;
 Function2 get_balance;
 Function3 get_nonce;
 Function6 set_nonce;
 Function2 get_code_hash;
 Function2 get_code;
 Function5 set_code;
 Function7 get_code_size;
 Function8 add_refund;
 Function9 get_refund;
 Function10 get_data;
 Function5 set_data;
 Function4 func_suicide;
 Function4 has_suicide;
 Function4 func_exists;
 Function4 func_empty;
 Function12 func_revert_to_snapshot;
 Function13 func_snapshot;
 Function5 add_preimage;
// block
 Function14 func_blockhash;
 Function15 func_coinbase;
 Function9 func_difficulty;
 Function9 func_number;
 Function9 func_gaslimit;
 Function9 func_timestamp;
// tx
 Function15 func_origin;
//
 Function17 contract_call;
 Function16 set_bytecode;
//event
 Function11 event_call;
 Function1 remove_data;
 //Function10 get_data_iter_next;
 //Function3 get_data_iter;
#ifdef __cplusplus
}
#endif
#endif //TVM_TVM_H
