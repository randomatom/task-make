# 第一行不要指定 shell, 默认使用当前shell. 方便 ${SHELL} 判断

QJS_BIN=qjs_bin/$(uname)-$(uname -m)/qjs
DL=/tmp/task-make-download
rm -rf ${DL}
mkdir -p ${DL}

if [ $# -eq 1 ]; then
	if [ "$1" = 'github' ]; then
		REMOTE=https://github.com/randomatom/task-make/raw/main
	elif [ "$1" = 'gitee' ] ;then
		REMOTE=https://gitee.com/randomatom/task-make/raw/main
	else
		echo "param [$1] ERROR"
		exit 1
	fi

	if [ -n "${REMOTE}" ]; then
		wget -P ${DL} ${REMOTE}/task.js
		echo ${REMOTE}/${QJS_BIN}
		wget -P ${DL} ${REMOTE}/${QJS_BIN}
	fi
else
	cp task.js ${DL}
	cp ${QJS_BIN} ${DL}
fi

exit 1

# docker 里没有sudo
SUDO=""
if which sudo 2> /dev/null > /dev/null ; then
	SUDO=sudo
fi

if [ ! -f ${DL}/qjs ]; then
	echo Please compile the "qjs" file yourself.
	exit 1
fi


[ -d /usr/local/bin ] || ${SUDO} mkdir -p /usr/local/bin
${SUDO} cp ${DL}/task.js /usr/local/bin
${SUDO} cp ${DL}/qjs /usr/local/bin

alias_cmd="alias m='qjs /usr/local/bin/task.js'"
case ${SHELL} in
	*bash)
		echo "${alias_cmd}" >> ~/.bashrc
		;;
	*zsh)
		echo "${alias_cmd}" >> ~/.zshrc
		;;
	*sh)
		echo "${alias_cmd}" >> ~/.profile
	;;
	*) echo error;;
esac

