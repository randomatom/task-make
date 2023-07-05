__init__:
	# 提取参数列表
	OPTION_LIST=""
	for i in $@; do
		echo $i
		OPTION_LIST="${OPTION_LIST} -D ${i} "
	done
	# 设置build 模式和路径
	if [ -z ${BUILD+x} ]; then
		BUILD=build
	fi

build / b:
	[ -d ${BUILD} ] || mkdir ${BUILD}
	cd ${BUILD}
	cmake \
		${OPTION_LIST} \
		..

build_ndk / bn:
	## NDK交叉编译
	[ -d ${BUILD} ] || mkdir ${BUILD}
	cd ${BUILD}
	NDK_ROOT=/Users/wzw/Library/Android/sdk
	cmake \
		-D CMAKE_BUILD_TYPE=Release \
		-D CMAKE_TOOLCHAIN_FILE=${NDK_ROOT}/ndk-bundle/build/cmake/android.toolchain.cmake \
		-D ANDROID_NDK=${NDK_ROOT}/ndk-bundle \
		-D ANDROID_TOOLCHAIN=clang  \
		-D ANDROID_ABI=armeabi-v7a \
		-D ANDROID_PLATFORM=android-22 \
		-D OpenCV_DIR=/Users/wzw/work/prj/cv/3rdparty/OpenCV-android-sdk/sdk/native/jni \
		${OPTION_LIST} \
		..

build_oh / bo:
	## OH 交叉编译
	[ -d ${BUILD} ] || mkdir ${BUILD}
	cd ${BUILD}
	OH_CMAKE=/Users/wzw/Library/Huawei/Sdk/openharmony/9/native/build/cmake/ohos.toolchain.cmake
	cmake \
		-D CMAKE_TOOLCHAIN_FILE=${OH_CMAKE} \
		-D OHOS_ARCH="armeabi-v7a" \
		..
	make VERBOSE=1

make / m:
	cd ${BUILD}
	make -j8

make_dbg / md:
	cd ${BUILD}
	make VERBOSE=1

clean_cache / cc:
	if [ -d ${BUILD} ]; then
		cd ${BUILD}
		# 强制删除缓存，避免option使用缓存,改后不生效
		rm -f CMakeCache.txt
	fi

clean / c:
	cd ${BUILD}
	make clean

clean_all / ca:
	rm -rf ${BUILD}
	mkdir ${BUILD}

run / r:
	cd ${BUILD}
	./demo

all / a:
	m ${BUILD} ENABLE_MEM_CHECK=OFF
	m make

all_platform:
	export BUILD=build
	m all
	export BUILD=build_ndk
	m all
	export BUILD=build_oh
	m all

pack:
	tar zcvf demo_src-$(date +%Y%m%d%H%M).tgz src scripts docs CMakeLists.txt task.mk

