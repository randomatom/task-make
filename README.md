# Task-Make

## 概述
日常开发或运维往往有很多繁琐的步骤和流程，需要执行一系列复杂的指令。<br>
对于需要多次重复执行的场合，每次都依赖人工重新输入执行，不仅效率低下, 又容易出错。  
将日常的命令组合，提取公共部分抽象成 task，进行复用，既减少工作量，也能减少出错概率。  
如何更好管理这些命令组合（task），有几种方案。

### 旧方案

#### 1. scripts 目录添加脚本

1. 这对于复杂的脚本比较合适，但对于一些就几行的小任务，用脚本有点划不来；
2. 对于小任务多的场景，导致脚本会非常多，不方便管理。
3. 另外，检索和执行也麻烦。

#### 2. 类似 make 方案

借用 make。在bash 定义alias，将需要的工作流放在本地 task.mk 文件里。这样可以将很多小任务放在一起，管理起来也比较方便。
```
alias m=`make -f task.mk`
```
然后执行
```
m make
m run
```

后面发现有很多不方便的地方：
1. makefile里的 写稍微复杂点的命令非常反人性。
   它不是一个完整的bash，不同命令行都是独立的，上下文是割裂的。条件判断非常麻烦。
   比如下面无法正常执行，`cd build` 之后 `pwd` 是不会变化的。
    ```
    cd build
    make 
    make install
    ```
    需要改为写在同一行
    ```
    cd build && make && make install
    ```
2. 不方便总览显示当前有多个目标（任务）；
3. 没有简写，每次需要打冗长的 目标名，也容易出错。
4. 不方便复用。

### 本方案

主要思路，借鉴make的方案 进行优化。采用类似 makefile 格式的脚本，很方便管理各种小任务。
同时便于利用该生态，用vim、vscode编辑自动语法高亮。  
相对makefile主要改进点：  
1. 每个task（或者叫tareget）里的命令，是完整的bash小脚本。而不是makefile 里面反直觉的方法。
1. 支持task 的显示、快速调用；
1. 允许 task 调用其他 task；
1. 除了本地的task.mk 文件，同时支持 全局的 task 库，类似c语言的系统的库，可以方便积累、调用。


##  安装

### 手动安装
1. 依赖 QuickJS  https://bellard.org/quickjs ，下载地址：https://bellard.org/quickjs/binary_releases 。  
    本来想用python脚本，但考虑到有时候会在一些嵌入式小系统里使用，python过于庞大的。  
    开始用lua写过一个版本，但考虑lua生态弱，后改为js，用QuickJS 执行。
2. 将task.js 和 qjs 放到 /usr/local/bin/
3. 在配置文件(~/.bashrc, ~/.zshrc等等）里加入
```
alias m="qjs /usr/local/bin/task.js"
```

### 一键安装
已经 提供在部分硬件平台上在mac和linux编译好的qjs二进制文件, 特殊平台的qjs需要自己编译。
需要bash支持, 只支持类 *nit系统，**不支持win**。


1. 从 github 安装. 由于网络问题，不一定能安装成功
```
curl https://github.com/randomatom/task-make/raw/main/install.sh | bash -s github
```
2. 从 gitee镜像 安装. 由于网络问题，不一定能安装成功
```
curl https://gitee.com/randomatom/task-make/raw/main/install.sh | bash -s gitee
```
3. `git` 下载到本地之后 `./install.sh`。

## mk文件

1. 类似于 makefile文件。以 `:` 结尾的行，为目标行，后面的命令会被执行。
2. 以 `*` 开头的目标行，为默认目标。当执行 m，不带参数，直接执行该目标。
3. 以 # 开头的行，为注释行，不会被执行
4. 在目标行之后以 ## 开头的第一行，作为对该目标的注释，会被显示
5. `__init__` 目标在正常目标执行之前，会被首先执行。

`task.mk`范例如下
```makefile
__init__:
	# 所有命令的执行之前会首先被调用
	echo "init"
	export NDK_PATH="xxx"
cmake:
	# 内部注释，该行不会被 -l 显示。（前面只有一个#)
	rm -rf build
	mkdir build
	cd build
	cmake ..
*make:
	## [*]代表默认命令. 当执行 m, 后面没有参数，直接执行该目标
	cd build
	make -j8
install:
	cd build
	make install
claen:
	## 目标后面第一行开头有两个##, 该行会被 -l 显示
	cd build
	make clean
make_and_push / mp:
	## 上面的"/" 后面的 mp是简称，方便输入.
	cd build
	make -j8
	adb push test /data/app/test
test:
	for f in $(find . -name "*.mk") ; do
		cat $f
	done
all:
	# 可以用 m 调用其他命令
	m cmake
	m make
	m install
```


## 目录结构

可通过 _TASK_PROFILE_DIR 环境变量设置指定，没有设置则默认路径 `~/.local/task`.
```
├── README.md
├── init_rc.sh
├── run_file_list.txt
└── repo
    ├── build.mk
    ├── linux.mk
    └──test.mk
```

1. init_rc.sh: 可以将公共的函数放在这里，本用户运行的 *.mk 都能复用
2. run_file_list.txt: 本机运行过的所有 *.mk 文件的列表，方便回顾
3. repo: 全局模块的存放目录

## 使用方式

### 本地task.mk

执行当前目录 task.mk 任务。用来管理一个项目中常用的task。

1.显示当前目标
```
$ m -l
Select a Task:
       1. cmake
  ==>  2. make                  # [*]代表默认命令. 当执行 m, 后面没有参数，直接执行该目标
       3. install
       4. claen                 # 目标后面第一行开头有两个##, 该行会被显示
       5. make_and_push / mp    # 上面的"/"后面的mp是简称，方便输入
       6. all
```

2. 执行目标。有三者方式，效果一样。
```
$ m make_and_push  # 全名
$ m mp             # 简称
$ m 5              # 序号
```

3. 执行默认目标
```
$ m   # 不带任何参数
$ m 0 # 序号0 代表默认目标
```

3. 当前目录新建task.mk
``` 
$ m -c 
```

4. 编辑当前目录task.mk
```
$ m -e
```

### 全局仓库

这部分是全局的，在任意目录都能直接执行。可被 当前目录task.mk 调用。  
主要目的用于积累复用公共流程。类似c语言的标准函数库。

1. 显示仓库模块
```
$ m @
    @ linux
    @ mac
    @ ssh
```
模块文件默认在 ~/.local/task/repo.
```
$ ls ~/.local/task/repo
  linux.mk    mac.mk   ssh.mk
```

2. 显示某个模块里面的命令

```
$ m -l @linux
$ m @linux
Select a Task:
      1. wifi_on
      2. wifi_off
      3. sensor
```

3. 执行某个模块目标

```
$ m @linux:sensor
$ m @linux:3
```

4. 新建 new_mod.mk

```
$ m -c @new_mod
```

5. 编辑模块

```
$ m -e @new_mod
```


### 调用其他模块的任务

当前目录 example/simple\_exam 有个例子

```
├── sub_mod
│   └── task.mk
└── task.mk
```


```makefile
run_sub_mod:
	## 调用子目录的模块
	echo "Root Working Directory: $(pwd)"
	m sub_mod/task.mk:run
	m -C sub_mod/task.mk:run
	echo  "=======find hello.txt========="
	find . -name "hello.txt"
```

用以下语法调用其他路径的模块的任务
有个参数 `-C`, 是否切换工作路径(change)。 含义同 make -C 的参数.
```
m sub_mod/task.mk:run     # 工作路径不变
m -C sub_mod/task.mk:run  # 切换工作路径到新模块所在路径
```

是否有带-C 的参数，执行结果如下:

```
Run Task: [ run_sub_mod ]
Root Working Directory: task-make/example/simple_exam
Run Task: [ sub_mod/task.mk:run ]
Working Directory: task-make/example/simple_exam. Create hello.txt
Run Task: [ sub_mod/task.mk:run ]
     % Enter Dir: [ sub_mod ]
Working Directory: task-make/example/simple_exam/sub_mod. Create hello.txt
=======find hello.txt=========
./sub_mod/hello.txt
./hello.txt
```








