cmake_minimum_required(VERSION 3.12)

project(demo LANGUAGES C CXX)

####################################################
#              编译选项开关
####################################################
option(ENABLE_MEM_CHECK "Enable memory check" OFF)

option(WITH_OPENCV "With Opencv Lib" ON)
option(WITH_NCNN "With ncnn Lib" OFF)
option(WITH_OPENMP "With OpenMP" OFF)


####################################################
#              设置编译器参数
####################################################
if (NOT CMAKE_BUILD_TYPE)
	set (CMAKE_BUILD_TYPE Release CACHE STRING "Build Type" FORCE)
endif()
message(STATUS "====> CMAKE_BUILD_TYPE is ${CMAKE_BUILD_TYPE}")

if (CMAKE_BUILD_TYPE STREQUAL "Release")
	add_compile_options(-O2)
	add_compile_options(-g)
elseif (CMAKE_BUILD_TYPE STREQUAL "Debug")
	add_compile_options(-O0)
	add_compile_options(-g)
else()
	message(FATAL_ERROR "CMAKE_BUILD_TYPE error")
endif()

# std=c++只针对c++编译器. 所以单独设置
set(CMAKE_CXX_FLAGS "${CMAKE_CXX_FLAGS} -std=c++11")

# android ndk中char类型是unsigned. x86是signed. 设置为一致
add_definitions("-fsigned-char")

# 内存检测
if (ENABLE_MEM_CHECK)
	message(STATUS "====> option ENABLE_MEM_CHECK ON")
	add_definitions(-fsanitize=address -fno-omit-frame-pointer)
	add_link_options(-fsanitize=address)
endif (ENABLE_MEM_CHECK)


####################################################
#              设置预编译宏
####################################################
add_compile_definitions(RKNN=0)


####################################################
#              设置项目路径
####################################################
if (NOT PRJ_DIR)
	set(PRJ_DIR ${CMAKE_SOURCE_DIR})
endif()
message(STATUS "\$\{PRJ_DIR\} => ${PRJ_DIR}")


####################################################
#              设置 SRC_FILES
####################################################
# 不太推荐用 GLOB_RECURSE, 推荐多写几个目录
file(GLOB SRC_FILES CONFIGURE_DEPENDS
	${PRJ_DIR}/src/*.c
	${PRJ_DIR}/src/*.cpp
)


####################################################
#              配置第三方库
####################################################
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


####################################################
#              include头文件搜索路径
####################################################
include_directories(
	${PRJ_DIR}/src/
)


####################################################
#              链接库的搜索路径
####################################################
# link_directories(
	# ${PRJ_DIR}/lib
# )


####################################################
#              添加链接库名称
####################################################
# list(APPEND LINKER_LIBS "mylib")
# message(STATUS "lib " ${LINKER_LIBS})


####################################################
#              生成可执行文件
####################################################
add_executable(${PROJECT_NAME} ${SRC_FILES})
list(APPEND LINKER_LIBS ${MY_LIB_NAME} m)
target_link_libraries(${PROJECT_NAME}
	PRIVATE
	${LINKER_LIBS}
)

