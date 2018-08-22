//
// Created by GuangYu Jing on 2018/7/17.
//

#ifndef TVM_TVM_H
#define TVM_TVM_H


#ifdef __cplusplus
extern "C" {
#endif
void tvm_start(void);
void tvm_test(void);
_Bool tvm_execute(char *str);
typedef int (*callback_fcn)(int);
typedef void (*testAry_fcn)(void*);
void some_c_func(callback_fcn);
void tvm_setup_func(callback_fcn callback);
void tvm_set_testAry_func(testAry_fcn);
callback_fcn func;
testAry_fcn testAry;

typedef void (*TransferFunc)(const char*, const char*, int);
TransferFunc transferFunc;
void setTransferFunc(TransferFunc);
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
typedef char* (*Function10) (const char*, const char*);
typedef void (*Function11) (const char*, const char*, const char*);
typedef void (*Function12) (int);
typedef int (*Function13)();
typedef char* (*Function14) (unsigned long long);
typedef char* (*Function15) ();



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
Function10 get_state;
Function11 set_state;
Function4 suicide;
Function4 has_suicide;
Function4 exists;
Function4 empty;
Function12 revert_to_snapshot;
Function13 snapshot;
Function5 add_preimage;
// block
Function14 blockhash;
Function15 coinbase;
Function9 difficulty;
Function9 number;
Function9 timestamp;
// tx
Function15 origin;
Function9 gaslimit;




#ifdef __cplusplus
}
#endif
#endif //TVM_TVM_H
