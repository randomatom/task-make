cmake_minimum_required(VERSION 3.12)

project(demo LANGUAGES C CXX)

# -----------------------------------------------------------------------
#                        编译选项开关
# -----------------------------------------------------------------------
option(ENABLE_MEM_CHECK "Enable memory check" OFF)
option(ENABLE_GPROF "Enable gprof" OFF)

option(WITH_OPENCV "With Opencv Lib" OFF)
option(WITH_NCNN "With ncnn Lib" OFF)
option(WITH_OPENMP "With OpenMP" OFF)

# -----------------------------------------------------------------------
#                        设置编译器参数
# -----------------------------------------------------------------------
message(STATUS "====> CMAKE_BUILD_TYPE is ${CMAKE_BUILD_TYPE}")

set(CMAKE_CXX_STANDARD 11)

# android ndk中char类型是unsigned. x86是signed. 设置为一致
add_definitions("-fsigned-char")

# 内存检测
if (ENABLE_MEM_CHECK)
	# https://gcc.gnu.org/onlinedocs/gcc/Instrumentation-Options.html
	# -fsanitize 选项。互斥, 建议每个都排查一遍
	set(MEM_CHK_OPTION "address")
	set(MEM_CHK_OPTION_LIST "address" "leak" "thread" "undefined")
	list(FIND MEM_CHK_OPTION_LIST ${MEM_CHK_OPTION} MEM_INDEX)
	if(${MEM_INDEX} GREATER -1)
		add_definitions(-Og) # 关闭原有优化选项
		add_definitions(-fsanitize=${MEM_CHK_OPTION} -fno-omit-frame-pointer)
		add_link_options(-fsanitize=${MEM_CHK_OPTION})
		message(STATUS "====> option ENABLE_MEM_CHECK ON. [-fsanitize=${MEM_CHK_OPTION}]")
	else()
		message(FATAL_ERROR "MEM_CHK_OPTION error")
	endif()
endif (ENABLE_MEM_CHECK)

# gprof
if (ENABLE_GPROF)
	message(STATUS "====> option ENABLE_GPROF ON")
	add_definitions(-pg)
	add_link_options(-pg)
endif (ENABLE_GPROF)

# -----------------------------------------------------------------------
#                        设置预编译宏
# -----------------------------------------------------------------------
add_compile_definitions(RKNN=0)

# -----------------------------------------------------------------------
#                        设置项目路径
# -----------------------------------------------------------------------
if (NOT PRJ_DIR)
	set(PRJ_DIR ${CMAKE_SOURCE_DIR})
endif()
message(STATUS "\$\{PRJ_DIR\} => ${PRJ_DIR}")

# -----------------------------------------------------------------------
#                        设置 SRC_FILES
# -----------------------------------------------------------------------
# 不太推荐用 GLOB_RECURSE, 推荐多写几个目录
file(GLOB SRC_FILES CONFIGURE_DEPENDS
	${PRJ_DIR}/src/*.c
	${PRJ_DIR}/src/*.cpp
)

# -----------------------------------------------------------------------
#                        设置 APP_FILES
# -----------------------------------------------------------------------
# 不太推荐用 GLOB_RECURSE, 推荐多写几个目录
file(GLOB APP1_FILES CONFIGURE_DEPENDS
	   ${PRJ_DIR}/apps/app1.cpp
)

file(GLOB APP2_FILES CONFIGURE_DEPENDS
	   ${PRJ_DIR}/apps/app2.cpp
)

# -----------------------------------------------------------------------
#                        配置第三方库
# -----------------------------------------------------------------------
if (WITH_OPENCV)
	message(STATUS "====> option WITH_OPENCV ON")
	find_package(OpenCV QUIET)
	include_directories(${OpenCV_INCLUDE_DIRS})
	list(APPEND LINKER_LIBS ${OpenCV_LIBS})
endif()

if (WITH_NCNN)
	message(STATUS "====> option WITH_NCNN ON")
	if (ncnn_DIR)
		include_directories(${ncnn_DIR}/include)
		link_directories(${ncnn_DIR}/lib)
		link_directories(${ncnn_DIR}/lib64)
	else()
		message(STATUS "\${ncnn_DIR} set default")
	endif ()
	list(APPEND LINKER_LIBS ncnn)
endif()

if (WITH_OPENMP)
	message(STATUS "option WITH_OPENMP ON")
	find_package(OpenMP QUIET)
	add_compile_options(${OpenMP_C_FLAGS})
	add_link_options(${OpenMP_EXE_LINKER_FLAGS} -fopenmp)
endif ()

# -----------------------------------------------------------------------
#                        include头文件搜索路径
# -----------------------------------------------------------------------
include_directories(
	${PRJ_DIR}/include/
	${PRJ_DIR}/src/
	${PRJ_DIR}/apps/
)

# -----------------------------------------------------------------------
#                        链接库的搜索路径
# -----------------------------------------------------------------------
# link_directories(
	# ${PRJ_DIR}/lib
# )

# -----------------------------------------------------------------------
#                        添加链接库名称
# -----------------------------------------------------------------------
# list(APPEND LINKER_LIBS "mylib")
# message(STATUS "lib " ${LINKER_LIBS})

# -----------------------------------------------------------------------
#                        生成动态库
# -----------------------------------------------------------------------
set(MY_LIB_NAME mylib)
add_library(${MY_LIB_NAME} SHARED ${SRC_FILES})
target_link_libraries(${MY_LIB_NAME}
	PRIVATE
	${LINKER_LIBS}
)

# -----------------------------------------------------------------------
#                        生成可执行文件
# -----------------------------------------------------------------------
set(APP1_NAME app1)
add_executable(${APP1_NAME} ${APP1_FILES})
list(APPEND LINKER_LIBS ${MY_LIB_NAME} m)
target_link_libraries(${APP1_NAME}
	PRIVATE
	${LINKER_LIBS}
)

set(APP2_NAME app2)
add_executable(${APP2_NAME} ${APP2_FILES})
list(APPEND LINKER_LIBS ${MY_LIB_NAME} m)
target_link_libraries(${APP2_NAME}
	PRIVATE
	${LINKER_LIBS}
)

