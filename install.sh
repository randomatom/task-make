#!/bin/bash

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
		wget -P "${DL}" "${REMOTE}/task.js"
		echo "${REMOTE}/${QJS_BIN}"
		wget -P ${DL} "${REMOTE}/${QJS_BIN}"
	fi
else
	cp task.js "${DL}"
	cp "${QJS_BIN}" "${DL}"
fi

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
echo -e '#!/bin/bash\nqjs /usr/local/bin/task.js "$@"' > ${DL}/m
${SUDO} cp ${DL}/task.js /usr/local/bin
${SUDO} cp ${DL}/qjs /usr/local/bin
${SUDO} cp ${DL}/m /usr/local/bin/m
${SUDO} chmod a+rx /usr/local/bin/m

echo "Install OK"
echo "Make sure that /usr/local/bin is in your \$PATH"

