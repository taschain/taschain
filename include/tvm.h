//
// Created by GuangYu Jing on 2018/7/17.
//

#ifndef TVM_TVM_H
#define TVM_TVM_H


//#ifdef __cplusplus
//extern "C" {
//#endif
void tvm_start(void);
void tvm_test(void);
void tvm_execute(char *str);
typedef int (*callback_fcn)(int);
typedef void (*testAry_fcn)(void*);
void some_c_func(callback_fcn);
void tvm_setup_func(callback_fcn callback);
void tvm_set_testAry_func(testAry_fcn);
callback_fcn func;
testAry_fcn testAry;

//#ifdef __cplusplus
//}
//#endif
#endif //TVM_TVM_H
