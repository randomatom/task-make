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
	cd ${BUILD_DIR}
	if [ ${PLATFROM} == "host" ]; then
		cmake \
			-D CMAKE_BUILD_TYPE=${BUILD_TYPE} \
			${ROOT_DIR}
	elif [ ${PLATFROM} == "ndk" ]; then
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
	elif [ ${PLATFROM} == "oh" ]; then
		OH_CMAKE=/Users/wzw/Library/Huawei/Sdk/openharmony/9/native/build/cmake/ohos.toolchain.cmake
		cmake \
			-D CMAKE_BUILD_TYPE=${BUILD_TYPE} \
			-D CMAKE_TOOLCHAIN_FILE=${OH_CMAKE} \
			-D OHOS_ARCH="armeabi-v7a" \
			${ROOT_DIR}
	else
		echo 'error' && exit 1
	fi

make / m:
	## make [ d ]
	cd ${BUILD_DIR}
	if [ $# == 0 ]; then
		make -j8
	elif [ $1 == 'd' ]; then
		# dbg 模式, 显示更多细节
		make VERBOSE=1
	else
		echo 'error' &&  exit 1
	fi

clean / c:
	## clean [ c | a ]
	if [ $# == 0 ]; then
		cd ${BUILD_DIR} && make clean
	elif [ $1 == 'c' ] ; then
		# 强制删除缓存，避免option使用缓存,改后不生效
		echo 'clean cache'
		[ -d ${BUILD_DIR} ] && rm -f ${BUILD_DIR}/CMakeCache.txt
	elif [ $1 == 'a' ] ; then
		# 删除全部目录
		echo 'clean all'
		rm -rf ${BUILD_DIR}
	else
		echo 'error' && exit 1
	fi

run / r:
	## run [ g ]
	cd ${BUILD_DIR}
	for r in app*; do
		./${r}
		if [ $# == 1 ] && [ $1 == 'g' ] ; then
			gprof ./${r} gmon.out
		fi
	done

all / a:
	m build
	m make

pack:
	tar zcvf demo_src-$(date +%Y%m%d%H%M).tgz src apps extern scripts include docs CMakeLists.txt task.mk

