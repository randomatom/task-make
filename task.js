/*
Auth : @RandomAtom
Date: 2023/06/13

## 概述
优化工作流，将日常工作的重复脚本固化。
1. 本地工作目录下有个 task.mk 文件，语法类似 makefile 格式.
2. 将日常需要的命令和工作流程，抽象成一个个的小Task。
3. 脚本是更好的文档
4. 自己或者他人 继续接手工作的时候，方便复现，不用很多东西从头再来.
5. 累积复用工作流, 提取到全局脚本


## task.mk 语法
1. 以 : 结尾的行，为任务行，后面的命令会被执行
3. 以 * 开头的行，为默认任务，当执行 m, 后面没有参数，直接执行该任务
1. 以 # 开头的行，为注释行，不会被执行
2. 以 ## 开头的行，为注释行，同时会被 -l 参数显示

task.mk 范例如下:
-------------------------------
__init__:
	# 后面所有任务的执行之前会首先被调用
	echo "init"
	export NDK_PATH="xxx"
cmake:
	# 内部注释，该行不会被 -l 显示。（前面只有一个#)
	rm -rf build
	mkdir build
	cd build
	cmake ..
 *make:
	# [*]代表默认任务. 当执行 m, 后面没有参数，直接执行该任务
	cd build
	make -j8
install:
	cd build
	make install
claen:
	## 任务后面第一行开头有两个##, 该行会被 -l 显示
	cd build
	make clean
make_and_push / mp:
	## 上面的"/" 后面的 mp是简称，方便输入.
	cd build
	make -j8
	adb push test /data/app/test
test:
	for f in `ls *.mk` ; do
		cat $f
	done
all:
	# 可以用 m 调用其他任务
	m cmake
	m make
	m install
-------------------------------

## install
1. 依赖 qjs
2. 将task.js 放到 /usr/local/bin/
3.  在~/.bashrc 里加入
	export _TASK_PROFILE_DIR=~/.local/task
	alias m="qjs /usr/local/bin/task.js"


## task_main_dir 目录结构
可通过 _TASK_PROFILE_DIR 环境变量设置指定，没有设置则默认路径 ~/.local/task.

├── init_rc.sh
├── run_file_list.txt
└─- repo
	├── build.mk
	└─── init.mk

1. init_rc.sh: 可以将公共的函数放在这里，本用户运行的 *.mk 都能复用
2. run_file_list.txt: 本机运行过的所有 *.mk 文件的列表，方便回顾
3. repo: 全部模块的存放目录
 
## 使用

### 本地task.mk
1.显示当前任务
$ m -l
Select a Task:
	  0. cmake
	  1. make
	  2. install
	  3. claen                       # 任务后面第一行开头有两个##, 该行会被显示
	  4. make_and_push / mp          # 上面的"/"后面的mp是简称，方便输入
	  5. all

2. 执行当前任务, 以下三者效果一样
$ m make_and_push  (全名)
$ m mp             (简称)
$ m 4              (序号)

3. 当前目录新建task.mk
$ m -c

4. 编辑当前目录task.mk
$ m -e

### 仓库模块
1. 显示仓库模块, 模块文件默认在 ~/.local/task/repo/*.mk
$ m @
	 @ linux
	 @ mac
	 @ ssh
$ ls ~/.local/task/repo
   linux.mk    mac.mk   ssh.mk

2. 显示某个模块里面的任务
$ m -l @linux
$ m @linux


3. 执行某个模块的任务
$ m @linux:sensor

4. 新建 new_mod.mk
$ m -c @new_mod

5. 编辑模块
$ m -e @new_mod

*/

import * as os from 'os'
import * as std from 'std'


let log = (s) => console.log(s)
let logd = (s) => console.log(s)
let logi = (s) => console.log('\x1b[0;33m' + s + '\x1b[0m')
let loge = (s) => console.log('\x1b[0;31m' + s + '\x1b[0m')
let log_obj = (s) => console.log(JSON.stringify(s, null, 4))

function input_str(s) {
	std.out.printf('\x1b[0;33m' + s + '\x1b[0m')
	std.out.flush()
	return std.in.getline().trim()
}

class MkInfo {
	constructor(file) {
		this.file = file
		// block_list: [block, block, ...]
		//        block: { tasks: [], cmd_block: '', comment: '', lineid: 0 }
		this.block_list = []
		// default_tasks: [name, name, ...]
		this.default_tasks = []
		this.init_block_task = ''
		this.err = 0
		this.err_msg = ''
		this.parse_file()
	}

	search_task(module_name, keys) {
		if (this.err != 0) {
			return []
		}
		let task_list = []
		for (let i = 0; i < this.block_list.length; i++) {
			let block = this.block_list[i]
			let task_name = `${module_name}:${block.tasks}`
			let line = task_name
			if (block.comment) {
				line += `#${block.comment}`
			}
			let find = true
			for (let j = 0; j < keys.length; j++) {
				if (!line.includes(keys[j])) {
					find = false
				}
			}
			if (find) {
				task_list.push(task_name)
			}
		}
		return task_list
	}

	find_task_block(tasks) {
		let block = null
		for (let i = 0; i < tasks.length; i++) {
			let t = tasks[i]
			if (t.match(/^\d+$/)) {
				let n = Number(t)
				if (n == 0) {
					if (this.default_tasks.length > 0) {
						block = this.find_task_block(this.default_tasks)
						break
					}
				}
				else if (1 <= n && n < this.block_list.length + 1) {
					block = this.block_list[n - 1]
					break
				}
			}
			else {
				let is_find = false
				for (let j = 0; j < this.block_list.length; j++) {
					let cur_block = this.block_list[j]
					for (let k = 0; k < cur_block.tasks.length; k++) {
						if (t == cur_block.tasks[k]) {
							is_find = true
							block = cur_block
							break
						}
					}
				}
			}
		}
		return block
	}

	parse_file() {
		let fd = std.open(this.file, 'r')
		if (!fd) {
			this.err = 1
			this.err_msg = `${this.file} DON'T exist!`
			return 1
		}

		let null_block = { tasks: [], cmd_block: '', comment: '', lineid: 0 }
		// 注意：要用拷贝语法，才能创建一个新的对象。直接赋值只沿用同一对象
		let block = { ...null_block }
		let default_flag = false
		let line_id = 0

		let line
		let init_task_name = '__init__'
		let after_task_first_line = false
		while ((line = fd.getline()) != undefined) {
			line_id++
			if (line.trim() == '') continue
			// logd('+ ' + line)
			let tasks = []
			let task_line_id = 0
			if (line.includes(':')) {
				let m0 = line.match(/^(__init__)\s*:/)
				let m1 = line.match(/^(\*?)([A-Za-z]\w*)\s*(?:\/\s*([A-Za-z]\w*)\s*)?:/)
				if (m0) {
					task_line_id = line_id
					tasks = [init_task_name]
				} else if (m1) {
					task_line_id = line_id
					default_flag = m1[1] == '*'
					tasks = [m1[2]]
					if (m1[3]) tasks.push(m1[3])
				}
			}

			if (tasks.length > 0) {
				if (block.tasks[0] == init_task_name && block.cmd_block) {
					if (this.init_block_task != '') {
						loge(`Error: duplication of [${init_task_name}]`)
						this.err = 2
					} else {
						this.init_block_task = block.cmd_block
						block = { ...null_block }
					}
				} else if (block.tasks.length > 0) {
					if (block.cmd_block) {
						this.block_list.push(block)
						block = { ...null_block }
					} else {
						this.err = 2
						this.err_msg = `Error: task [${block.tasks[0]}] have NO command. at line ${line_id}.`
						return
					}
				}

				if (this.find_task_block(tasks)) {
					this.err = 2
					this.err_msg = `Error: duplication of task [${tasks}] at line ${line_id}:\n`
					this.err_msg += `==>    ${line}`
					return
				}

				if (default_flag) {
					if (this.default_tasks.length > 0) {
						this.err = 2
						this.err_msg = `Error: duplication of default task [${tasks[0]}] at line ${line_id}:\n`
						this.err_msg += `==>    ${line}`
						return
					} else {
						this.default_tasks = tasks
						default_flag = false
					}
				}
				block.tasks = tasks
				block.lineid = task_line_id
				after_task_first_line = true
			} else {
				// cmd块 区域
				if (block.tasks.length == 0) {
					this.err = 2
					this.err_msg = `Error [1] at line ${line_id}:\n`
					this.err_msg += `==>    ${line}`
					return
				}
				if (!line.match(/^\t/)) {
					this.err = 2
					this.err_msg = `Error [2] at line ${line_id}:\n`
					this.err_msg += `==>    ${line}`
					return
				}
				if (after_task_first_line) {
					// 当注释以 ## 开头，将提取并显示
					if (line.trim().startsWith('##')) {
						block.comment = line.trim().substring(2)
					}
					after_task_first_line = false
				}
				block.cmd_block += line.slice(1) + '\n'
			}
		}

		if (block.tasks.length > 0 && this.find_task_block(block.tasks)) {
			this.err = 2
			this.err_msg = `Error: duplication of task [${block[0]}]: at line ${line_id}.`
			return
		}

		if (block.tasks.length > 0) {
			if (block.cmd_block) {
				this.block_list.push(block)
				if (default_flag) {
					this.default_tasks = block.tasks
					default_flag = false
				}
			} else {
				this.err = 2
				this.err_msg = `Error: task [${block.tasks[0]}] have NO command. at line ${line_id}.`
				return
			}
		}
	}

	cal_task_str_len(arr) {
		let len = 0
		for (let i = 0; i < arr.length; i++) {
			len += arr[i].length + 3
		}
		if (arr.length >= 1) len -= 3
		return len
	}

	print() {
		log('==> MkInfo')
		log(`file = ${this.file} `)
		log(`default = ${this.default_tasks} `)
		// log(`init_block_task = ${ this.init_block_task } `)
		log(`err = ${this.err} `)
		this.block_list.forEach((x, idx) => {
			log(`  task[${idx}]: ${x.task}, lineid: ${x.lineid} `)
			// log(`      cmd: ${ x.cmd_block } `)
		})
		log('--------------------')
	}

	print_err() {
		if (this.err > 0) {
			loge(this.err_msg)
		}
	}

	print_task(is_simple) {
		if (is_simple == 's') {
			this.block_list.forEach((x, idx) => {
				let line = `${x.tasks[0]}`
				log(line)
			}, this)
		} else {
			logi('Select a Task:')
			let task_max_length = 0
			this.block_list.forEach((x, idx) => {
				let len = this.cal_task_str_len(x.tasks)
				if (len > task_max_length) {
					task_max_length = len
				}
			}, this)
			if (task_max_length > 30) {
				task_max_length = 30
			}
			this.block_list.forEach((x, idx) => {
				let line = ''
				if (this.default_tasks.length > 0 && x.tasks == this.default_tasks) line = '==>  '
				else line = '     '

				let ord = idx + 1
				if (ord <= 9) line += ' ' + ord
				else line += ord

				let length1 = line.length

				line += `. ${x.tasks[0]}`
				for (let i = 1; i < x.tasks.length; i++) {
					line += ` / ${x.tasks[i]}`
				}

				if (x.comment && is_simple != 's') {
					let cur_length = line.length
					let length2 = length1 + task_max_length + 12
					if (cur_length < length2) {
						for (let i = 0; i < length2 - cur_length; i++) {
							line += ' '
						}
					}
					line += `# ${x.comment.trim()}`
				}
				log(line)
			}, this)
		}
	}
}

function crc16(s) {
	let n = 0
	let poly = 0xA001
	let ret = 0xA001
	for (let i = 0; i < s.length; i++) {
		let n = s.charCodeAt(i)
		ret = (ret ^ n) & 0xffff
		for (let j = 0; j < 8; j++) {
			if ((ret & 0x0001) > 0) {
				ret = (ret >> 1)
				ret = ((ret ^ poly) & 0xFFFF)
			} else {
				ret = (ret >> 1)
			}
		}
	}
	let hi = ((ret >> 8) & 0xFF)
	let lo = (ret & 0xFF)
	ret = ((lo << 8) | hi)
	return ret
}


class ArgInfo {
	constructor(scriptArgs, task_main_dir) {
		this.action = 'default'
		this.file = ''
		this.task = ''
		this.scriptArgs = scriptArgs
		this.list_option = ''
		this.search_option = ''
		this.search_key_works = []
		this.shell_args = []
		this.task_main_dir = task_main_dir
		this.task_root_workdir = std.getenv('_task_root_workdir')
		if (!this.task_root_workdir) {
			this.task_root_workdir = os.getcwd()[0]
		}
		this.change_workdir = false
		this.cur_task_file = 'task.mk'
		let cur_file = std.getenv('_TASK_CUR_FILE')
		if (cur_file) {
			this.cur_task_file = cur_file
		}
		this.err = this.parse_args()
	}

	update_run_list() {
		let run_file_list_fname = this.task_main_dir + '/run_file_list.txt'
		if (this.task_main_dir && this.file == 'task.mk') {
			let mk_full_path = os.getcwd()[0] + '/' + this.file
			let fd = std.open(run_file_list_fname, 'r')
			if (!fd) {
				fd = std.open(run_file_list_fname, 'w')
				if (fd) {
					fd.puts(mk_full_path + '\n')
					fd.close()
				}
			} else {
				let file_list = fd.readAsString().split('\n')
				let is_in_list = false
				file_list.forEach((x, idx) => {
					let line = x.trim()
					if (line) {
						if (line == mk_full_path) is_in_list = true
					}
				})
				fd.close()
				if (!is_in_list) {
					fd = std.open(run_file_list_fname, 'w')
					file_list.push(mk_full_path)
					file_list.sort()
					file_list.forEach((x, idx) => {
						let fname = x.trim()
						if (fname) {
							if (os.lstat(fname)[1] == 0) {
								fd.puts(fname + '\n')
							}
						}
					})
					fd.close()
				}
			}
		}
	}


	parse_file_and_task(arg) {
		this.file = ''
		this.task = ''
		let ret = 1
		let arr = arg.split(':')
		if (arr.length == 2 && (arr[0] != '' || arr[1] != '')) {
			let m_file = arr[0].match(/^@\w+(\/|\w|\.|-)+$|^(\/|\w|\.|-)+$|^$/)
			let m_task = arr[1].match(/^[A-Za-z]\w*$|^\d+$|^$/)
			if (m_file && m_task) {
				this.file = m_file[0]
				if (this.file == '') this.file = this.cur_task_file
				this.task = m_task[0]
				ret = 0
			}
		} else if (arr.length == 1) {
			let m_file = arr[0].match(/^@(\/|\w|\.|-)*$/)
			if (m_file) {
				this.file = m_file[0]
				this.task = ''
				ret = 0
			} else {
				let m_task = arr[0].match(/^[A-Za-z]\w*$|^\d+$|^$/)
				if (m_task) {
					this.file = this.cur_task_file
					this.task = m_task[0]
					ret = 0
				}
			}
		} else {
			ret = 1
		}
		if (ret != 0) {
			loge(`Task [${arg}] Illegal format!`)
		}
		return ret
	}

	parse_flag(arg) {
		// logd(`arg: [${arg}]`)
		if (arg[0] == '-') {
			switch (arg[1]) {
				case 'l':
					this.action = 'list'
					if (arg.length > 2 && arg[2] == 's') {
						// simple mode
						this.list_option = 's'
					}
					break
				case 'e':
					this.action = 'edit'
					break
				case 'c':
					this.action = 'create'
					break
				case 's':
					this.action = 'search'
					break
				case 'r':
					this.action = 'run'
				case 'C':
					// 和 make -C 一样
					this.change_workdir = true
					break
				case 'h':
					this.action = 'help'
					break
				default:
					this.action = ''
					return 1
			}
		}
		return 0
	}

	expand_file(short_name) {
		if (short_name.startsWith('@')) {
			if (short_name.endsWith('/')) {
				return `${this.task_main_dir}/repo/${short_name.slice(1)}`
			} else {
				return `${this.task_main_dir}/repo/${short_name.slice(1)}.mk`
			}
		} else {
			return short_name
		}
	}

	parse_args() {
		// 例子:
		// m
		// m   run
		// m   @
		// m   @build
		// m   @build:make
		// m   -l      run
		// m   -l      @
		// m   -l      @build
		// m   -l      @build:make
		// 0    1        2
		this.file = this.cur_task_file
		let i = 1
		let args = this.scriptArgs
		for (i = 1; i < args.length; i++) {
			let arg = args[i]
			if (arg[0] == '-') {
				let ret = this.parse_flag(arg)
				if (ret != 0) {
					loge(`option [${arg}] error`)
					return ret
				}
			} else {
				break
			}
		}

		if (i < args.length) {
			if (this.action == 'list' || this.action == 'default') {
				if (this.parse_file_and_task(args[i]) == 0) {
					this.shell_args = args.slice(i + 1)
				} else {
					return 1
				}
			} else if (this.action == 'edit' || this.action == 'create') {
				this.file = args[i]
			} else if (this.action == 'search' || this.action == 'run') {
				this.search_key_works = args.slice(i)
			} else {
				return 1
			}
		}
		return 0
	}

	get_base_dir(path) {
		let arr = path.split('/')
		if (arr.length > 1) {
			return arr.slice(0, arr.length - 1).join('/')
		} else {
			return ''
		}
	}

	get_relative_path(path, base_dir) {
		// loge(`path: ${path}, base_dir: ${base_dir}`)
		if (base_dir == '') {
			return path
		} else {
			if (path.indexOf(base_dir) == 0) {
				return path.slice(base_dir.length + 1)
			} else {
				return path
			}
		}
	}

	print_repo(sub_dir, is_simple) {
		if (!sub_dir.match(/^@$|^@.+\/$/)) {
			loge(`error: ${sub_dir}`)
			return
		}
		let real_dir = `${this.task_main_dir}/repo/${sub_dir.slice(1)}`
		let repo_list = read_dir_mks(real_dir)
		if (!repo_list) {
			loge(`Module Path [${sub_dir}] error!`)
			return
		}

		if (is_simple == 's') {
			repo_list[0].forEach((x, _) => {
				log(`${sub_dir}${x}`)
			})
			repo_list[1].forEach((x, _) => {
				log(`${sub_dir}${x}/ `)
			})
		} else {
			repo_list[0].forEach((x, _) => {
				log(`      ${sub_dir}${x}`)
			})

			repo_list[1].forEach((x, _) => {
				log(`    > ${sub_dir}${x}/ `)
			})
		}
	}

	search_task_in_module(key_words) {
		function find_all_file_recursion(dir_name) {
			let file_list = []
			let dirs_st = os.readdir(dir_name)
			if (dirs_st[1] != 0) return [];
			dirs_st[0].sort()
			dirs_st[0].forEach((x, _) => {
				let f_st = os.lstat(dir_name + '/' + x)
				if (f_st[1] == 0) {
					if (f_st[0].mode & os.S_IFDIR && !x.startsWith('.')) {
						let new_file_list = find_all_file_recursion(dir_name + '/' + x)
						file_list = file_list.concat(new_file_list)
					} else if (x.endsWith('.mk') && !x.startsWith('.')) {
						file_list.push(x.replace('.mk', ''))
					}
				}
			})
			return file_list
		}
		let file_list = find_all_file_recursion(this.task_main_dir)
		file_list.sort()
		let task_list = []
		file_list.forEach((x, _) => {
			let file = `${this.task_main_dir}/repo/${x}.mk`
			let info = new MkInfo(file)
			let tasks = info.search_task(`@${x}`, key_words)
			if (tasks.length > 0) {
				task_list = task_list.concat(tasks)
			}
		})
		return task_list
	}

	filt_task_list(task_list) {
		function show_task_list(task_list) {
			logi(`    0. EXIT`)
			for (let i = 0; i < task_list.length; i++) {
				std.printf("%5d. %s\n", i + 1, task_list[i])
			}
		}
		let ret = 0
		while (true) {
			if (task_list.length <= 1) break
			show_task_list(task_list)
			let line = input_str(' Input index or key words: ')
			if (!line) break
			if (line.match(/^\d+$/)) {
				let index = parseInt(line)
				if (index == 0) {
					task_list = []
					break
				}
				else if (1 <= index && index <= task_list.length) {
					task_list = [task_list[index - 1]]
				} else {
					loge(`error: index out of range!`)
				}
			} else {
				let arr = []
				line.split(' ').forEach((x, _) => {
					if (x) arr.push(x)
				})
				let new_task_list = []
				task_list.forEach((x, _) => {
					let find = 0
					for (let i = 0; i < arr.length; i++) {
						if (x.includes(arr[i])) {
							find += 1
						}
					}
					if (find == arr.length) new_task_list.push(x)
				})
				task_list = new_task_list
				if (task_list.length == 0) ret = 1
			}
		}
		if (task_list.length == 0) return [null, ret]
		else if (task_list.length > 1) return [null, ret]
		else if (task_list.length == 1) {
			let task_cmd = [task_list[0]]
			logd(` ===>  ${task_cmd}`)
			let line = input_str(' Input parameters: ')
			task_cmd = task_cmd.concat(line.split(' '))
			return [task_cmd, ret]
		}
	}

	run() {
		let ret = 0
		if (this.action == 'help') {
			logd('Usage:')
			logd('    m    [[file:]task] [arg]...')
			logd('    m -C [file:task]   [arg]...')
			logd('    m -c [file]')
			logd('    m -l [file]')
			logd('    m -e [file]')
			logd('    m -s [pattern]...')
			logd('    m -r [pattern]...')
			logd('    m -h')
			logd('Arguments:')
			logd('    [[file:]task]    Specifies the task name of the file. If not specified, the default')
			logd('                     is the task.mk file in the current directory.')
			logd('    [pattern]        the search pattern.')
			logd('Options:')
			logd('    -C               Change to the directory where the task file is located before executing the task.')
			logd('    -c               Create the mk_file.')
			logd('    -e               Edit the mk_file.')
			logd('    -l               List the tasks.')
			logd('    -s               Search the tasks in global module.')
			logd('    -r               Run the tasks in global module.')
			logd('    -h               Help.')
		}
		else if (this.action == 'create') {
			if (!this.file || this.file == '@') return 1;
			let file = this.expand_file(this.file)
			if (file.endsWith('/')) {
				if (os.mkdir(file) != 0) {
					loge(`Create Directory [${file}] Error!`)
					return 1
				}
			} else {
				if (os.lstat(file)[1] == 0) {
					loge(`${file} has exist!`)
					return 1
				} else {
					let fd = std.open(file, 'w')
					if (!fd) {
						loge(`create [${file}] Error!`)
						return 1
					}
					fd.puts('*make:\n\t[ -f Makefile ] || [ -f makefile ] && make')
					fd.close()
					logi(`create ${file}`)
					let link_file = this.expand_file(this.file)
					os.symlink(file, link_file)
					os.exec(['vi', file, '+'])
				}
			}
		} else if (this.action == 'edit') {
			let file = this.expand_file(this.file)
			if (os.lstat(file)[1] == 0) {
				os.exec(['vi', file, '+'])
			} else {
				loge(`[${this.file}] DON'T exist!`)
				return 1
			}
		} else if (this.action == 'search') {
			let task_list = this.search_task_in_module(this.search_key_works)
			task_list.forEach((x, _) => {
				logd(x)
			})
		} else if (this.action == 'run') {
			let task_list = this.search_task_in_module(this.search_key_works)
			if (task_list.length > 0) {
				let result = this.filt_task_list(task_list)
				let task_with_args = result[0]
				let err = result[1]
				if (err != 0) {
					loge(`error: no task match!`)
				} else {
					if (task_with_args) {
						let a = new ArgInfo([this.scriptArgs[0]].concat(task_with_args), this.task_main_dir)
						if (a.err == 0) {
							a.run()
						}
					}
				}
			} else {
				loge(`error: no task match!`)
			}
		} else if (this.action == 'list') {
			if (this.file && this.task) {
				let file = this.expand_file(this.file)
				if (file.endsWith('.mk')) {
					let info = new MkInfo(file)
					if (info.err == 0) {
						let block = info.find_task_block([this.task])
						if (block) {
							log(block.cmd_block)
						} else {
							loge(`Task [${this.task}] DON'T exist!`)
							return 1
						}
					} else {
						info.print_err()
						return 1
					}
				} else if (file.endsWith('/')) {
					this.print_repo(file, this.list_option)
				}
			} else if (this.file) {
				if (this.file.match(/^@$|^@.+\/$/)) {
					if (this.list_option == '') {
						logi('Select a Module:')
					}
					this.print_repo(this.file, this.list_option)
				} else {
					let file = this.expand_file(this.file)
					let info = new MkInfo(file)
					if (info.err == 0) {
						info.print_task(this.list_option)
					} else {
						if (this.list_option == '') {
							info.print_err()
						}
						return 1
					}
				}
			}
		} else if (this.action == 'default') {
			// 没有 -l/-c 等参数
			if (this.file == '@' || this.file.endsWith('/')) {
				if (this.list_option == '') {
					logi('Select a Module:')
				}
				this.print_repo(this.file, this.list_option)
			} else if (this.file) {
				let file = this.expand_file(this.file)
				let info = new MkInfo(file)
				if (info.err > 0) {
					info.print_err()
					return 1
				}
				this.update_run_list()

				let new_workdir = ''
				if (this.change_workdir) new_workdir = this.get_base_dir(this.file)
				let cur_wd = os.getcwd()
				let new_wd = os.realpath(new_workdir)

				let tasks = []
				if (this.task) {
					tasks = [this.task]
				} else if (info.default_tasks.length > 0) {
					tasks = info.default_tasks
				}
				if (tasks.length > 0) {
					let block = info.find_task_block(tasks)
					if (block) {
						let run_info = ''
						let rel_path = this.get_relative_path(this.file, this.task_root_workdir)
						if (rel_path == 'task.mk') {
							run_info = `Run Task: [ ${block.tasks[0]} ]`
						} else {
							run_info = `Run Task: [ ${rel_path}:${block.tasks[0]} ]`
						}
						if (this.shell_args.length > 0) {
							run_info = run_info.slice(0, -1) + `${this.shell_args} ]`
						}
						logi(run_info)
						if (cur_wd[1] == 0 && new_wd[1] == 0 && cur_wd[0] != new_wd[0]) {
							// let p1 = this.get_relative_path(cur_wd[0], this.task_root_workdir)
							let p2 = this.get_relative_path(new_wd[0], this.task_root_workdir)
							logi(`     % Enter Dir: [ ${p2} ]`)
						}
						ret |= this.run_task(info.file, block, info.init_block_task, this.shell_args, new_workdir)
					} else {
						loge(`Task [${this.task}] DON'T exist!`)
					}
				} else {
					info.print_task(this.list_option)
				}
			}
		} else {
			loge('Option error')
		}
		return ret
	}

	run_task(cur_file, block, init_block_task, shell_args, new_workdir) {
		// 截获Ctrl+C 按键，可以中断程序
		let trap_int_func = "trap 'onCtrlC' INT\n" +
			"onCtrlC() {\n\texit 1\n}\n" +
			"trap 'OnError ${LINENO} ' ERR\n"
		// 截获错误，显示行号
		let trap_err_func = 'OnError() {\n\terrcode="${3:-1}"\n' +
			`\t((lineid=\${1}+LINE_OFFSET))\n` +
			`\techo -e \"\\033[31mError on [ ${cur_file} +\${lineid} ]. Exit [\${errcode}]. \\033[0m\"\n ` +
			'\texit "${errcode}"\n}\n'
		let trap_cmd = trap_int_func + trap_err_func

		// set -e 当出错的时候，程序退出
		// set -u 当使用未初始化变量，程序退出
		// set -o pipefail 当在管道中出现错误，程序退出
		let set_cmd = 'set -eu\n' + 'set -o pipefail\n'
		// 子shell中无法看到父shell中的函数，所以在子shell里需要重新定义m()函数
		let m_func_cmd = `m() {\n\tqjs "${scriptArgs[0]}" "$@"\n}\n`
		let realpath = os.realpath(this.file)
		let export_cmd = `export _task_root_workdir="${this.task_root_workdir}"\n` +
			`export _TASK_CUR_FILE="${realpath[0]}"\n`

		let init_cmd = ''
		if (this.task_main_dir) {
			let init_file = this.task_main_dir + '/init_rc.sh'
			if (os.lstat(init_file)[1] == 0) {
				init_cmd = `. ${init_file}\n`
			}
		}
		if (new_workdir) {
			// 注意加双引号，防止路径中有空格
			init_cmd += `cd "${new_workdir}"\n`
		}
		let commit_line = '\n##########################\n\n'
		let shell_cmd = trap_cmd + set_cmd + m_func_cmd + export_cmd + init_cmd + commit_line
			+ init_block_task
		// 动态计算行号
		let lines = shell_cmd.split(/\r?\n/)
		let offset = block.lineid - lines.length + 1
		shell_cmd = shell_cmd.replace('LINE_OFFSET', offset.toString()) + block.cmd_block

		let tag = crc16(shell_cmd)
		let tmp_dir = `/tmp/mk_task_dir@${std.getenv('USER')}`
		// log_obj(block)
		let shell_name = `${tmp_dir}/${block.tasks[0]}_${tag}.sh`
		// logd(`${shell_name}`)

		if (os.lstat(tmp_dir)[1] != 0) {
			os.mkdir(tmp_dir)
			os.exec(['chmod', '700', tmp_dir])
		}

		if (Math.random() < 0.001) {
			// 过一段时间清理临时目录
			os.exec(['rm', '-f', tmp_dir + '/*.sh'])
		}

		if (os.lstat(shell_name)[1] != 0) {
			let fd = std.open(shell_name, 'wb+')
			if (!fd) return 1
			fd.puts(shell_cmd)
			fd.close()
		} else {
			let fd = std.open(shell_name, 'rb')
			if (fd) {
				let data = fd.readAsString()
				if (data == shell_cmd) {
					fd.close()
				} else {
					logi('crc16 error! mybe error.')
					fd.close()
					let fd2 = std.open(shell_name, 'wb')
					if (!fd2) return 1
					fd2.puts(shell_cmd)
					fd2.close()
				}
			} else {
				return 1
			}
		}
		let bash_cmd = ['/bin/bash', shell_name].concat(shell_args)
		// logd(bash_cmd)
		return os.exec(bash_cmd)
	}
}


function read_dir_mks(dir_name) {
	let mk_list = []
	let dir_list = []
	let dirs_st = os.readdir(dir_name)
	if (dirs_st[1] == 0) {
		dirs_st[0].sort()
		dirs_st[0].forEach((x, _) => {
			let f_st = os.lstat(dir_name + '/' + x)
			if (f_st[1] == 0) {
				if (f_st[0].mode & os.S_IFDIR && !x.startsWith('.')) {
					dir_list.push(x)
				} else if (x.endsWith('.mk') && !x.startsWith('.')) {
					x = x.replace('.mk', '')
					mk_list.push(x)
				}
			}
		})
	} else {
		return null
	}
	return [mk_list, dir_list]
}

function main() {
	// example:
	// m
	// m  task
	// m  @build.task
	// m  @build.task             param1       param2
	// m      -l        @
	// m      -c        @build
	// m      -l        @build:task
	// 0       1           2          3            4

	let task_main_dir = std.getenv('_TASK_PROFILE_DIR')
	if (!task_main_dir) {
		task_main_dir = std.getenv('HOME') + '/.local/task'
	}
	let task_repo_dir = task_main_dir + '/repo'
	if (os.lstat(task_main_dir)[1] != 0) {
		os.exec(['mkdir', '-p', task_main_dir])
		os.exec(['mkdir', '-p', task_repo_dir])
		let st = os.lstat(task_main_dir)
		if (!(st[1] == 0 && st[0].mode & os.S_IFDIR)) {
			loge('_TASK_PROFILE_DIR DON\'T exist or create ~/.local/task failed!')
			return 1
		}
	}
	let argInfo = new ArgInfo(scriptArgs, task_main_dir)
	// log_obj(argInfo)
	let ret = argInfo.err
	if (argInfo.err == 0) {
		ret = argInfo.run()
	}
	return ret
}

std.exit(main())
