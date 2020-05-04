#include <stdio.h>
#include <string.h>
#include "xdag_libp2p.h" // 此处为上一步生成的.h文件

int main(int argc, char **argv){
    char c1[] = "world";
    GoString s1 = {c1, strlen(c1)};// 构建go类型
    char *result = hello(s1);
    printf("hello result:%s\n", result);

    if(argc > 1) {
        char *c2 = argv[1];
        GoString addr = {c2, strlen(c2)};
        char *r2 = xdag_libp2p_send(addr);
        printf("xdag_libp2p_send result:%s\n", r2);
    }

    return 0;
}
