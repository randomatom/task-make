__init__:
	BUILD=build

# 这是一个注释，不会显示在 -l 中
build / b:
	rm -rf ${BUILD}
	mkdir ${BUILD}

*make:
	## 默认任务
	cd ${BUILD}
	make -j8

clean:
	make clean

deploy / d:
	## 部署任务
	echo "deploying..."

test:
	npm test

#############################
######  分隔线测试   #########

all:
	m build
	m make
