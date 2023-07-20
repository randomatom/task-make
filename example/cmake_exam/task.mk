__init__:
	# 设置 build 平台: 比如 host / ndk / oh 等等，可自行增加task
	[ -z ${PLATFROM+x} ] && PLATFROM=host
	BUILD_DIR=build/${PLATFROM}
	[ -d ${BUILD_DIR} ] || mkdir -p ${BUILD_DIR}
	# 设置 build 类型: Debug / Release / MinSizeRel / RelWithDebInfo
	BUILD_TYPE=Release
	ROOT_DIR=$(pwd)

build / b:
	echo "===> ${BUILD_DIR}"
	m build_${PLATFROM}

build_host / bh:
	## 编译平台: host
	cd ${BUILD_DIR}
	cmake \
		-D CMAKE_BUILD_TYPE=${BUILD_TYPE} \
		${ROOT_DIR}

build_ndk / bn:
	## 编译平台: NDK
	cd ${BUILD_DIR}
	NDK_ROOT=/Users/wzw/Library/Android/sdk
	cmake \
		-D CMAKE_BUILD_TYPE=${BUILD_TYPE} \
		-D CMAKE_TOOLCHAIN_FILE=${NDK_ROOT}/ndk-bundle/build/cmake/android.toolchain.cmake \
		-D ANDROID_NDK=${NDK_ROOT}/ndk-bundle \
		-D ANDROID_TOOLCHAIN=clang  \
		-D ANDROID_ABI=armeabi-v7a \
		-D ANDROID_PLATFORM=android-22 \
		-D OpenCV_DIR=/Users/wzw/work/prj/cv/3rdparty/OpenCV-android-sdk/sdk/native/jni \
		${ROOT_DIR}

build_oh / bo:
	## 编译平台: OH
	cd ${BUILD_DIR}
	OH_CMAKE=/Users/wzw/Library/Huawei/Sdk/openharmony/9/native/build/cmake/ohos.toolchain.cmake
	cmake \
		-D CMAKE_BUILD_TYPE=${BUILD_TYPE} \
		-D CMAKE_TOOLCHAIN_FILE=${OH_CMAKE} \
		-D OHOS_ARCH="armeabi-v7a" \
		${ROOT_DIR}

make / m:
	cd ${BUILD_DIR}
	make -j8

make_dbg / md:
	cd ${BUILD_DIR}
	make VERBOSE=1

clean / c:
	cd ${BUILD_DIR}
	make clean

clean_cache / cc:
	# 强制删除缓存，避免option使用缓存,改后不生效
	[ -d ${BUILD_DIR} ] && rm -f ${BUILD_DIR}/CMakeCache.txt

clean_all / ca:
	rm -rf ${BUILD_DIR}
	mkdir -p ${BUILD_DIR}

run / r:
	cd ${BUILD_DIR}
	for r in app*; do
		./${r}
	done

run_gprof / rg:
	cd ${BUILD_DIR}
	./demo
	gprof ./demo gmon.out

all / a:
	m build
	m make_dbg

pack:
	tar zcvf demo_src-$(date +%Y%m%d%H%M).tgz src apps extern scripts include docs CMakeLists.txt task.mk

