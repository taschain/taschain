//
// Created by GuangYu Jing on 2018/7/17.
//

#ifndef TVM_TVM_H
#define TVM_TVM_H


void tvm_start(void);
void tvm_test(void);
typedef int (*callback_fcn)(int);
void some_c_func(callback_fcn);
void tvm_setup_func(callback_fcn callback);
void tvm_execute(char *str);

#endif //TVM_TVM_H
