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
2. 以 * 开头的行，为默认任务，当执行 m, 后面没有参数，直接执行该任务
3. 以 "###" 开头的行，分隔行, 会显示'--------'
4. 以 "^\t#" 开头的行，为注释行，不会被执行
5. 以 "^\t##" 开头的行，为注释行，同时会被 -l 参数显示

task.mk 范例如下:
-------------------------------
__init__:
	# 该模块在所有后续任务执行前调用，用于设定公共内容
	echo "init"
	export NDK_PATH="xxx"
cmake:
	# 内部注释，该行不会被 -l 显示。（前面只有一个#)
	rm -rf build
	mkdir build
	cd build
	cmake ..
mk_dir:
	# workdir默认由顶层task.mk设定，使子脚本无法识别自身路径，难以引用当前目录文件
	# 新增 MK_DIR 变量，标识mk脚本所在目录
	python ${MK_DIR}/run.py

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
	### 三个#表示增加分隔线
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

├── __init__.sh
├── run_file_list.txt
└─- repo
	├── build.mk
	└─── init.mk

1. __init__.sh: 可以将公共的函数放在这里，本用户运行的 *.mk 都能复用
2. run_file_list.txt: 本机运行过的所有 *.mk 文件的列表，方便回顾
3. repo: 全部模块的存放目录

## 使用

### 本地task.mk
1.显示当前任务
$ m -l
Select a Task:
	  1. make
	  2. install
	  3. claen                       # 任务后面第一行开头有两个##, 该行会被显示
==>   4. make_and_push / mp          # 上面的"/"后面的mp是简称，方便输入
      --------------------
	  5. test                        # 三个#表示增加分隔线
	  6. all

2. 执行任务, 以下三者效果一样
$ m make_and_push  # 全名
$ m mp             # 简称
$ m 4              # 序号

3. 执行默认任务
$ m        # 不带任何参数
$ m 0      # 序号0 代表默认目标

4. 当前目录新建task.mk
$ m -c

5. 编辑当前目录task.mk
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

function time_stamp() {
	let d = new Date()
	return std.sprintf('[%02d:%02d:%02d] ', d.getHours(), d.getMinutes(), d.getSeconds())
}

let log = (s) => print(s)
let logd = (s) => print(s)
let logi = (s) => print('\x1b[0;33m' + s + '\x1b[0m')
let loge = (s) => print('\x1b[0;31m' + s + '\x1b[0m')
let logi2 = (s) => print('\x1b[0;33m' + time_stamp() + s + '\x1b[0m')
let loge2 = (s) => print('\x1b[0;31m' + time_stamp() + s + '\x1b[0m')
let log_obj = (s) => print(JSON.stringify(s, null, 4))

function input_str(s) {
	std.out.printf('\x1b[0;33m' + s + '\x1b[0m')
	std.out.flush()
	return std.in.getline().trim()
}

class MkInfo {
	constructor(file) {
		this.file = file
		// block_list: [block, block, ...]
		//     block: { tasks: [name, name，...], task_id: 0, cmd_block: '', comment: '', start_line_num: 0 }
		this.block_list = []
		// init_block:
		//     block: { tasks: [name, name，...], task_id: 0, cmd_block: '', comment: '', start_line_num: 0 }
		this.init_block = null
		// default_tasks: [name, name, ...]
		this.default_tasks = []
		this.err = 0
		this.err_msg = ''
		this.parse_file()
		// log_obj(this)
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
				task_list.push([task_name, block.comment])
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
				else {
					for (let j = 0; j < this.block_list.length; j++) {
						let cur_block = this.block_list[j]
						if (cur_block.task_id == n) {
							block = cur_block
							break
						}
					}
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

	parse_lines_info() {
		let fd = std.open(this.file, 'r')
		if (!fd) {
			this.err = 1
			this.err_msg = `${this.file} DON'T exist!`
			return null
		}
		let lines = []
		let line
		let line_num = 0
		while ((line = fd.getline()) != undefined) {
			line = line.trimEnd()
			let type = ''
			let tasks = []
			let param = []
			let default_flag = false
			line_num++
			if (line.trim() == '') {
				continue
			} else if (line.startsWith('###')){
				type = 'div'
			} else if (line.startsWith('#!')){
				type = 'pre_cmd'
				let m0 = line.match(/^#!\s+import\s+([\/|\w|\.|-]*)(\s+as\s+(\w*))*\s*$/)
				if (m0) {
					param = ['import', m0[1], m0[3]]
				} else {
					this.err = 2
					this.err_msg = `Error at line ${line_num}. pre_cmd Error.\n`
					this.err_msg += `==>    ${line}`
					return null
				}
			} else if (line.startsWith('#')) {
				continue
			} else if (line.endsWith(':')) {
				let m0 = line.match(/^(__init__)\s*:/)
				let m1 = line.match(/^(\*?)(_?[A-Za-z][\w|-]*)\s*(?:\/\s*([A-Za-z][\w|-]*)\s*)?:/)
				//                      *          TASK              /             s           :
				if (m0) {
					type = 'task'
					tasks = ['__init__']
				} else if (m1) {
					type = 'task'
					default_flag = (m1[1] == '*')
					tasks = m1.slice(2).filter(x => x != null)
				} else {
					this.err = 2
					this.err_msg = `Error at line ${line_num}. The task name is invalid. For example "Aa1-A1_1" or "_abc" is OK. "1a1" or "__abc" is Error.\n`
					this.err_msg += `==>    ${line}`
					return null
				}
			} else if (line.startsWith('\t')) {
				type = 'code'
				line = line.substring(1)
			} else {
				this.err = 2
				this.err_msg = `Error at line ${line_num}.\n`
				this.err_msg += `==>    ${line}`
				return null
			}
			let line_info = {num: line_num, str:line, type:type, param:param, tasks:tasks, default_flag:default_flag}
			lines.push(line_info)
		}
		return lines
	}

	parse_file() {
		function lines_into_group(lines) {
			let block_group = []
			let type_list = lines.map(x => x.type[0]).join('')
			let block_start = 0
			while (1) {
				// d:div t:task c:code
				let m = type_list.match(/^d*tc*/)
				if (m) {
					let block_len = m[0].length
					let line_block = lines.slice(block_start, block_start + block_len)
					block_start += block_len
					block_group.push(line_block)
					type_list = type_list.slice(block_len)
					if (type_list.length == 0) break
				} else {
					return null
				}
			}
			return block_group
		}

		let lines = this.parse_lines_info()
		if (lines == null) {
			return 1
		}
		let pre_cmd = lines.filter(x => x.type == 'pre_cmd')
		lines = lines.filter(x => x.type != 'pre_cmd')
		let raw_task_block_list = lines_into_group(lines)
		let thiz = this
		let task_id = 1
		let last_line_num = 0
		raw_task_block_list.forEach(raw_task_block => {
			let block = { tasks: [], task_id: 0, cmd_block: '', comment: '', start_line_num: 0 , with_div: false}
			raw_task_block.forEach(line => {
				if (line.type == 'task') {
					block.tasks = line.tasks
					if (line.default_flag) {
						if (thiz.default_tasks.length > 0) {
							thiz.err = 2
							thiz.err_msg = `Error at line ${line.num}: Duplication of default task.\n`
							this.err_msg += `==>    ${line.str}`
							return 1
						}
						thiz.default_tasks = line.tasks
					}
					if (line.tasks[0] != '__init__' && !line.tasks[0].startsWith('_')) {
						block.task_id = task_id
						task_id ++
					}
					if (thiz.find_task_block(line.tasks)) {
						thiz.err = 2
						thiz.err_msg = `Error at line ${line.num}. Duplication of task [${line.tasks}].\n`
						this.err_msg += `==>    ${line.str}`
						return 1
					}
					block.start_line_num = line.num
				} else if (line.type == 'code') {
					block.cmd_block += '\n'.repeat(line.num - last_line_num) + line.str
					if (line.str.trim().startsWith('##')) {
						block.comment += line.str.trim().substring(2)
					}
				} else if (line.type == 'div') {
					block.with_div = true
				}
				last_line_num = line.num
			});
			block.cmd_block = block.cmd_block.trim()
			if (block.tasks[0] == '__init__') {
				if (thiz.init_block != null) {
					thiz.err = 2
					thiz.err_msg = `Error: Duplication of task [__int__].\n`
					return 2
				}
				thiz.init_block = block
			} else {
				thiz.block_list.push(block)
			}
		});
		// log_obj(this.block_list)
	}

	print() {
		log('==> MkInfo')
		log(`file = ${this.file} `)
		log(`default = ${this.default_tasks} `)
		log(`err = ${this.err} `)
		this.block_list.forEach((x, idx) => {
			log(`  task[${idx}]: ${x.task}, start_line_num: ${x.start_line_num} `)
			// log(`      cmd: ${ x.cmd_block } `)
		})
		log('--------------------')
	}

	print_err() {
		if (this.err > 0) {
			loge(this.err_msg)
		}
	}

	print_task(list_option) {
		function cal_task_str_len(arr) {
			let len = 0
			for (let i = 0; i < arr.length; i++) {
				len += arr[i].length + 3
			}
			if (arr.length >= 1) len -= 3
			return len
		}
		if (list_option == 's') {
			this.block_list.forEach((x, idx) => {
				let line = `${x.tasks[0]}`
				log(line)
			}, this)
		} else {
			logi('Select a Task:')
			let task_max_length = 0
			this.block_list.forEach((x, idx) => {
				let len = cal_task_str_len(x.tasks)
				if (len > task_max_length) {
					task_max_length = len
				}
			}, this)
			if (task_max_length > 30) {
				task_max_length = 30
			}
			this.block_list.forEach((x, idx) => {
				// log_obj(x)
				if (list_option == '' && x.with_div) {
					let sep_line = ''
					if (this.block_list.length < 10) sep_line += '      ---';
					else sep_line += '      ----'
					for (let i = 0; i < task_max_length; i++) {
						sep_line += '-'
					}
					log(sep_line)
					let start_new = 0
					for (let i = 0; i < x.comment.length; i++) {
						if (x.comment[i] == '#') start_new += 1
					}
					x.comment = x.comment.slice(start_new)
				}

				if (x.task_id == 0) return

				let line = ''
				if (this.default_tasks.length > 0 && x.tasks == this.default_tasks) line += '==>  '
				else line += '     '

				let ord = x.task_id
				if (ord <= 9) line += ' ' + ord
				else line += ord

				let length1 = line.length

				line += `. ${x.tasks[0]}`
				for (let i = 1; i < x.tasks.length; i++) {
					line += ` / ${x.tasks[i]}`
				}

				if (x.comment && list_option == '') {
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


class ArgInfo {
	constructor(scriptArgs, task_main_dir) {
		this.action = 'default'
		this.file = ''
		this.compile_bash_file = ''
		this.task = ''
		this.scriptArgs = scriptArgs
		this.list_option = ''
		this.search_option = ''
		this.search_key_works = []
		this.shell_args = []
		this.incomplete_text = ''
		this.task_main_dir = task_main_dir
		this.task_repo_dir = task_main_dir + '/repo'
		this.task_root_workdir = std.getenv('_TASK_ROOT_WORKDIR')
		this.err = 0
		if (!this.task_root_workdir) {
			this.task_root_workdir = os.getcwd()[0]
		}
		this.change_workdir = false
		this.mkfile_dir = ''
		this.default_task_file = std.getenv('_TASK_CUR_DEFAULT_FILE')
		if (!this.default_task_file) this.default_task_file = 'task.mk'
		this.is_subtask = false
		if (std.getenv('_TASK_IS_SUBTASK')) {
			this.is_subtask = true
		}
		// logi(`is_subtask: ${this.is_subtask}`)
		this.tmp_dir = std.getenv('_TASK_TMP_DIR')
		if (!this.tmp_dir) {
			if (os.lstat('/dev/shm')[1] == 0) {
				this.tmp_dir = `/dev/shm/mk_task_dir@${std.getenv('USER')}`
			} else {
				this.tmp_dir = `/tmp/mk_task_dir@${std.getenv('USER')}`
			}
			if (os.lstat(this.tmp_dir)[1] != 0) {
				if (os.mkdir(this.tmp_dir) == 0) {
					if (os.exec(['chmod', '700', this.tmp_dir]) != 0) {
						loge(`chmod ${this.tmp_dir} error!`)
						this.err = 1
					}
				} else {
					loge(`create ${this.tmp_dir} error!`)
					this.err = 1
				}
			}
		}
		this.run_task_flag = 0
		if (this.err == 0) {
			this.err = this.parse_args()
		}
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
				} else {
					loge(`open ${run_file_list_fname} error!`)
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
					if (fd) {
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
					} else {
						loge(`open ${run_file_list_fname} error!`)
					}
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
			let m_file = arr[0].match(/^@\w+(\/|\w|\.|-)*$|^(\/|\w|\.|-)+$|^$/)
			let m_task = arr[1].match(/^[A-Za-z_][\w|-]*$|^\d+$|^$/)
			if (m_file && m_task) {
				if (m_file[0]) this.file = m_file[0]
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
				let m_task = arr[0].match(/^[A-Za-z_][\w-]*$|^\d+$|^$/)
				if (m_task) {
					this.file = this.default_task_file
					this.task = m_task[0]
					ret = 0
				}
			}
		} else {
			ret = 1
		}
		if (ret != 0) {
			loge(`Task [${arg}] Illegal format!`)
			if (arg.endsWith('.mk')) {
				let f_st = os.lstat(arg)
				if (f_st[1] == 0 && (f_st[0].mode & os.S_IFREG)) {
					loge(`    Do you mean "${arg}:"?`)
				}
			}
		}
		return ret
	}

	parse_flag(arg) {
		// logd(`arg: [${arg}]`)
		if (arg[0] == '-') {
			switch (arg[1]) {
				case 'l':
					this.action = 'list'
					if (arg.length > 2 && 'sc'.includes(arg[2])) {
						// simple mode
						this.list_option = arg[2]
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
				case 'w':
					// 类似 make -C 参数
					this.change_workdir = true
					break
				case 'C':
					this.action = 'compile'
					break
				case 't':
					this.action = 'tab_complete'
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
				return `${this.task_repo_dir}/${short_name.slice(1)}`
			} else {
				return `${this.task_repo_dir}/${short_name.slice(1)}.mk`
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
		this.file = this.default_task_file
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
			} else if (this.action == 'create') {
				this.file = args[i]
			} else if (this.action == 'tab_complete') {
				this.incomplete_text = args[i]
			} else if (this.action == 'edit') {
				this.parse_file_and_task(args[i])
			} else if (this.action == 'search' || this.action == 'run') {
				this.search_key_works = args.slice(i)
			} else if (this.action == 'compile') {
				this.file = args[i]
				if (i + 1 < args.length) {
					this.compile_bash_file = args[i + 1]
				} else {
					this.compile_bash_file = this.file.replace(/\.mk$/, '.sh')
				}
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

	get_absolute_dir(path) {
		let p1 = this.get_base_dir(path)
		if (p1 == "") {
			return os.getcwd()[0]
		} else {
			let p2 = os.realpath(p1)
			if (p2[1] == 0) return p2[0];
			else return ""
		}
	}

	get_pretty_relative_path(path, base_dir) {
		let short_path = path
		if (path.indexOf(base_dir) == 0) {
			short_path = path.slice(base_dir.length + 1)
		}
		if (short_path.startsWith(this.task_repo_dir)) {
			short_path = short_path.replace(this.task_repo_dir + '/', '@')
			if (short_path.endsWith('.mk')) {
				short_path = short_path.slice(0, -3)
			}
		}
		return short_path
	}

	read_dir_mks(dir_name) {
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

	print_repo(sub_dir, list_option) {
		if (!sub_dir.match(/^@$|^@.+\/$/)) {
			loge(`error: ${sub_dir}`)
			return
		}
		let real_dir = `${this.task_main_dir}/repo/${sub_dir.slice(1)}`
		let repo_list = this.read_dir_mks(real_dir)
		if (!repo_list) {
			loge(`Module Path [${sub_dir}] error!`)
			return
		}

		if (list_option == 's') {
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
		// task_list: [[task_name, comment], ...]
		function show_task_list(task_list) {
			logi(`    0. EXIT`)
			for (let i = 0; i < task_list.length; i++) {
				if (task_list[i][1]) {
					std.printf("%5d. %-32s #%s\n", i + 1, task_list[i][0], task_list[i][1])
				} else {
					std.printf("%5d. %s\n", i + 1, task_list[i][0])
				}
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
						let line = x[0] + x[1]
						if (line.includes(arr[i])) {
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
			let task_cmd = [task_list[0][0]]
			let comment = task_list[0][1]
			if (task_cmd[0].includes(',')) {
				task_cmd[0] = task_cmd[0].split(',')[0]
			}
			if (comment) {
				std.printf(" ===> %-32s #%s\n", task_cmd[0], comment)
			} else {
				std.printf(" ===> %s\n", task_cmd[0])
			}
			let line = input_str(' Input parameters: ')
			if (line) task_cmd = task_cmd.concat(line.split(' '))
			return [task_cmd, ret]
		}
	}

	do_compile() {
		let info = new MkInfo(this.file)
		if (info.err != 0) {
			loge(`parse file [${this.file}] error!`)
			return 1
		}
		let fd = std.open(this.compile_bash_file, 'w')
		if (!fd) {
			loge(`open compile bash file [${this.compile_bash_file}] error!`)
			return 2
		}
		fd.puts('#!/bin/bash\n')
		fd.puts('# This file is automatically generated by take-make.\n')
		fd.puts('# https://github.com/randomatom/task-make\n\n')
		fd.puts('#################################################################\n')
		fd.puts('#                       user defined tasks\n')
		fd.puts('#################################################################\n\n')
		let task_list = []
		function put_task(fd, task_name, cmd_block) {
			fd.puts(`${task_name}() {\n`)
			let cmd_arr = cmd_block.split('\n')
			let empty = true
			for (let i = 0; i < cmd_arr.length; i++) {
				let x = cmd_arr[i]
				if (x.trim()) {
					x = x.trimEnd()
					if (x.startsWith('m ')) {
						if (x.trim()[0] == '-') {
							loge(`error: "m ${x}" can not use - option!`)
							return 3
						}
						x = `run_task "do_${x.slice(2)}"`
					}
					if (!x.startsWith('#')) {
						empty = false
					}
					fd.puts(`	${x}\n`)
				}
			}
			if (empty) {
				// bash 函数必须有内容，否则会报错
				fd.puts(`    __fill_blank=1\n`)
			}
			fd.puts(`}\n\n`)
			return 0
		}
		if (info.init_block != null) {
			put_task(fd, "__init__", info.init_block.cmd_block)
		}
		for (let i = 0; i < info.block_list.length; i++) {
			let b = info.block_list[i]
			task_list.push(b.tasks[0])
			if (put_task(fd, 'do_' + b.tasks[0], b.cmd_block) != 0) {
				return 3
			}
		}

		let init_func = ''
		if (info.init_block != null) {
			init_func += '    __init__\n'
		}

		fd.puts(`task_list=(\n`)
		task_list.forEach((x, _) => {
			if (x.startsWith('_')) return
			fd.puts(`	${x}\n`)
		})
		fd.puts(')\n\n')

		fd.puts('#################################################################\n')
		fd.puts('#                         main script\n')
		fd.puts('#################################################################\n\n')
		let func_call_task_str = 'run_task() {\n' +
			'	cwd=$(pwd)\n' +
			'	arg=""\n' +
			'	[ $# -eq 1 ] || arg="${*: 2} "\n' +
			'	echo -e "\\x1b[0;33mRun Task: [ ${1:3} ${arg}] \\x1b[0m"\n' +
			init_func +
			'	"$@"\n' +
			'	cd "${cwd}"\n' +
			'}\n\n'
		fd.puts(func_call_task_str)

		let main_shell_str = 'task=""\n' +
			'if [ $# -eq 0 ]; then\n' +
			'	echo "Select a Task:"\n' +
			'	PS3="Enter a number? "\n' +
			'	select t in "${task_list[@]}"; do\n' +
			'		task=$t\n' +
			'		break\n' +
			'	done\n' +
			'else\n' +
			'	for t in "${task_list[@]}"; do\n' +
			'		[ "$t" == "$1" ] && task=$1\n' +
			'	done\n' +
			'fi\n' +
			'if [ -z "$task" ]; then\n' +
			'	echo "Invalid selection or argument!"\n' +
			'	exit 1\n' +
			'fi\n' +
			'shift\n\n' +
			'run_task "do_$task" "$@"\n'
		fd.puts(main_shell_str)
		fd.close()
		os.exec(['chmod', '+x', this.compile_bash_file])
		logi2(`compile bash file [${this.compile_bash_file}] success!`)
		return 0
	}

	do_help() {
		log('Usage:')
		log('    m    [[file:]task] [arg]...')
		log('    m -w [file:task]   [arg]...')
		log('    m -c [file]')
		log('    m -l [file]')
		log('    m -e [file]')
		log('    m -s [pattern]...')
		log('    m -r [pattern]...')
		log('    m -C [mk_file] [sh_file]')
		log('    m -h')
		log('Arguments:')
		log('    [[file:]task]    Specifies the task name of the file. If not specified, the default')
		log('                     is the task.mk file in the current directory.')
		log('    [pattern]        the search pattern.')
		log('Options:')
		log('    -w               Change to WORKDIR where the task file is located before executing the task.')
		log('    -c               Create the mk_file.')
		log('    -e               Edit the mk_file.')
		log('    -l               List the tasks.')
		log('    -s               Search the tasks in global module.')
		log('    -r               Run the tasks in global module.')
		log('    -C               Compile the *.mk file into *.sh.')
		log('    -h               Help.')
		log(`\nExample template for task.mk script:\n
__init__:
	# 该模块在所有后续任务执行前调用，用于设定公共内容
	BUILD=build

build / b:
	# 内部注释，该行不会被 -l 显示。（前面只有一个#)
	# 上面的"/" 后面的 mp是简称，方便输入.
	rm -rf \${BUILD}
	mkdir \${BUILD}
	cd \${BUILD}
	cmake ..

*make:
	# [*]代表默认任务. 当执行 m, 后面没有参数，直接执行该任务
	cd \${BUILD}
	make -j8

claen:
	## 任务后面第一行开头有两个##, 该行会被 -l 显示
	cd \${BUILD}
	make clean

MK_DIR_exam:
	# workdir默认由顶层task.mk设定，使子脚本无法识别自身路径，难以引用当前目录文件
	# 新增 MK_DIR 变量，标识mk脚本所在目录
	python \${MK_DIR}/run.py

#############################
######  会显示分隔线 ########

all:
	# 可以用 m 调用其他任务
	m build
	m make
`)
		return 0
	}

	do_create() {
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
				fd.puts('__init__:\n\t# echo \${MK_DIR}\n\n*run1 / r:\n\tpwd\n')
				fd.close()
				logi2(`create ${file}`)
				let link_file = this.expand_file(this.file)
				os.symlink(file, link_file)
				os.exec(['vi', file, '+'])
			}
		}
		return 0
	}

	do_edit() {
		let file = this.expand_file(this.file)
		if (os.lstat(file)[1] == 0) {
			if (this.task) {
				let info = new MkInfo(file)
				if (info.err == 0) {
					let block = info.find_task_block([this.task])
					if (!block) {
						let fd = std.open(file, 'a')
						if (fd) {
							fd.puts(`\n${this.task}:\n`)
							fd.close()
						}
						os.exec(['vi', '+', file])
					} else {
						os.exec(['vi', `+${block.start_line_num}`, file])
					}
				} else {
					loge(info.err_msg)
					input_str('Press any key to continue...')
					os.exec(['vi', '+', file])
				}
			} else {
				os.exec(['vi', file])
			}
		} else {
			loge(`${this.file} DON'T exist!`)
			return 1
		}
		return 0
	}

	do_search() {
		let task_list = this.search_task_in_module(this.search_key_works)
		task_list.forEach((x, _) => {
			if (x[1]) {
				std.printf("%-32s #%s\n", x[0], x[1])
			} else {
				std.printf("%s\n", x[0])
			}
		})
		return 0
	}

	do_run() {
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
						a.do_main()
					}
				}
			}
		} else {
			loge(`error: no task match!`)
			return 1
		}
		return 0
	}

	do_list() {
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
		} else {
			return 1
		}
		return 0
	}

	// 这个模块是作为bash里补全功能,采用渐进式补全
	do_tab_complete() {
		let complete_task_list = []
		let complete_mk_list = []
		let complete_path_list = []
		let file = this.expand_file(this.file)
		if (file.endsWith('.mk')) {
			let info = new MkInfo(file)
			if (info.err == 0) {
				info.block_list.forEach((x, idx) => {
					if (!x.tasks[0].startsWith('_')) {
						complete_task_list.push(`${x.tasks[0]}`)
					}
				})
			}
		}
		let real_dir = `${this.task_main_dir}/repo`
		let module_names = this.read_dir_mks(real_dir)
		if (module_names != null) {
			module_names[0].forEach((x, _) => {
				complete_mk_list.push(`@${x}:`)
			})
		}
		let cur_dir_mks = this.read_dir_mks(os.getcwd()[0])
		if (cur_dir_mks != null) {
			cur_dir_mks[0].forEach((x, _) => {
				if (x != 'task') {
					complete_mk_list.push(`${x}.mk:`)
				}
			})
			cur_dir_mks[1].forEach((x, _) => {
				complete_path_list.push(`${x}/`)
			})
		}

		// 当输入的字符为空时，列出当前 task.mk 里所有模块，以及当前目录的下*.mk文件，以及目录
		if (this.incomplete_text == '') {
			let all_list = complete_task_list.concat(complete_mk_list).concat(complete_path_list)
			// all_list.forEach((x, _) => log(x.replace(':', '\\:')))
			all_list.forEach((x, _) => log(x))
		} else {    // 当输入的字符不为空时，渐进式补全。
			// 补全 默认 task模块
			complete_task_list.forEach((x, _) => {
				if (x == this.incomplete_text) {
				} else if (x.startsWith(this.incomplete_text)) {
					log(x)
				} else if (this.incomplete_text.startsWith(x)) {
				}
			})
			// 补全 *.mk 文件
			complete_mk_list.forEach((x, _) => {
				if (x == this.incomplete_text) {
					let info = new MkInfo(this.expand_file(x.slice(0, -1)))
					if (info.err == 0) {
						x = x.replace(':', '\\:')
						info.block_list.forEach((y, _) => {
							log(`${x}${y.tasks[0]}`)
						})
					}
				} else if (x.startsWith(this.incomplete_text)) {
					log(x)
				} else if (this.incomplete_text.startsWith(x)) {
					let info = new MkInfo(this.expand_file(x.slice(0, -1)))
					if (info.err == 0) {
						x = x.replace(':', '\\:')
						let part_task = this.incomplete_text.split(':')[1]
						info.block_list.forEach((y, _) => {
							if (y.tasks[0].startsWith(part_task)) {
								log(`${x}${y.tasks[0]}`)
							}
						})
					}
				}
			})
			// 补全路径
			complete_path_list.forEach((p, _) => {
				if (p == this.incomplete_text) {
					let cur_dir_mks = this.read_dir_mks(os.getcwd()[0] + '/' + p)
					if (cur_dir_mks != null) {
						cur_dir_mks[0].forEach((x, _) => {
							log(`${p}${x}.mk\\:`)
						})
						cur_dir_mks[1].forEach((x, _) => {
							log(`${p}${x}/`)
						})
					}
				} else if (p.startsWith(this.incomplete_text)) {
					log(p)
				} else if (this.incomplete_text.startsWith(p)) {
					let lastSlashIndex = this.incomplete_text.lastIndexOf('/')
					if (lastSlashIndex == -1) return 0
					let base_path = this.incomplete_text.slice(0, lastSlashIndex + 1)
					let cur_dir_mks = this.read_dir_mks(os.getcwd()[0] + '/' + base_path)
					log_obj(cur_dir_mks)
					if (cur_dir_mks != null) {
						cur_dir_mks[0].forEach((x, _) => {
							log(`${base_path}${x}.mk\\:`)
						})
						cur_dir_mks[1].forEach((x, _) => {
							log(`${base_path}${x}/`)
						})
					}
				}
			})
		}
		return 0
	}

	do_default() {
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
			this.mkfile_dir = this.get_absolute_dir(this.file)
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
					let rel_path = this.get_pretty_relative_path(this.file, this.task_root_workdir)
					if (rel_path == 'task.mk') {
						run_info = `Run Task: [ ${block.tasks[0]} ]`
					} else {
						run_info = `Run Task: [ ${rel_path}:${block.tasks[0]} ]`
					}
					if (this.shell_args.length > 0) {
						run_info = run_info.slice(0, -1) + `${this.shell_args} ]`
					}
					logi2(run_info)
					if (cur_wd[1] == 0 && new_wd[1] == 0 && cur_wd[0] != new_wd[0]) {
						// let p1 = this.get_relative_path(cur_wd[0], this.task_root_workdir)
						let p2 = this.get_pretty_relative_path(new_wd[0], this.task_root_workdir)
						logi(`     % Enter Dir: [ ${p2} ]`)
					}
					return this.run_task(info.file, block, info.init_block, this.shell_args, new_workdir, this.mkfile_dir)
				} else {
					loge(`Task [${this.task}] DON'T exist!`)
					return 1
				}
			} else {
				info.print_task(this.list_option)
			}
		}
		return 0
	}

	do_main() {
		let ret = 0
		if (this.action == 'help') {
			ret = this.do_help()
		} else if (this.action == 'create') {
			ret = this.do_create()
		} else if (this.action == 'edit') {
			ret = this.do_edit()
		} else if (this.action == 'search') {
			ret = this.do_search()
		} else if (this.action == 'run') {
			ret = this.do_run()
		} else if (this.action == 'list') {
			ret = this.do_list()
		} else if (this.action == 'compile') {
			if (this.file == '') this.file = 'task.mk'
			if (this.compile_bash_file == '') this.compile_bash_file = 'task.sh'
			ret = this.do_compile()
		} else if (this.action == 'tab_complete') {
			ret = this.do_tab_complete()
		} else if (this.action == 'default') {
			ret = this.do_default()
		} else {
			loge('Option error')
			ret = 2
		}
		return ret
	}

	run_task(cur_file, block, init_block, shell_args, new_workdir, mkfile_dir) {
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

		// 截获Ctrl+C 按键，可以中断程序
		let trap_int_func = "trap 'onCtrlC ${LINENO}' INT\n" +
			'onCtrlC() {\n' +
			'\t_SIGINT_FLAG=1\n' +
			`\t((line_num=\${1}+@2))\n` +
			'\t[ ${1} -lt @1 ] && exit 1\n' +
			`\techo -e \"\\033[31m  SIGINT on [ ${cur_file} +\${line_num} ].\\033[0m\"\n` +
			'\texit 1\n' +
			'}\n'
		// 截获错误，显示行号
		let trap_err_func = "trap 'OnError ${LINENO}' ERR\n" +
			'OnError() {\n' +
			'\terrcode=$?\n' +
			'\t_ERROR_FLAG=1\n' +
			`\tif [ \${1} -ge @2 ] ; then \n` +
			`\t    ((line_num=\${1} - @5))\n` +
			`\telif [ \${1} -ge @1 ] ; then \n` +
			`\t    ((line_num=\${1} - @4))\n` +
			`\tfi\n` +
			// '\techo OnError @ $@, err = $errcode\n' +
			'\t[ ${1} -lt @1 ] && exit 1\n' +
			'\tif [ $errcode -eq 127 ]; then\n' +
			`\t    echo -e \"\\033[31m  Error on [ ${cur_file} +\${line_num} ]. \\033[0m\"\n` +
			'\t    exit 1\n' +
			'\telse\n' +
			`\t    echo -e \"\\033[31m  Error on [ ${cur_file} +\${line_num} ]. code \${errcode}. \\033[0m\"\n` +
			`\t    echo -e \"\\033[33m  To allow the program to continue running after an error, try \\"cmd1 | cmd2 || true\\"\\033[0m\"\n` +
			'\t    exit "${errcode}"\n' +
			'\tfi\n' +
			'}\n'
		let trap_exit_func = "trap 'OnExit ${LINENO}' EXIT\n" +
			'OnExit() {\n' +
			'\terrcode=$?\n' +
			// '\techo OnExit @ $@, errcode = $errcode\n' +
			`\t((line_num=\${1}+@2))\n` +
			'\t[ ${errcode} -eq 0 ] && exit 0\n' +
			'\t[ -z ${_ERROR_FLAG+x} ] || exit ${errcode}\n' +
			'\t[ -z ${_SIGINT_FLAG+x} ] || exit ${errcode}\n' +
			'\t[ ${1} -lt @1 ] && exit ${errcode}\n' +
			'\t[ ${1} -ge @3 ] && exit 0\n' +
			`\techo -e \"\\033[31m  Exit \${errcode} on [ ${cur_file} +\${line_num} ]. \\033[0m\"\n` +
			'\texit "${errcode}"\n' +
			'}\n'
		let trap_cmd = trap_int_func + trap_err_func + trap_exit_func

		// set -e 当出错的时候，程序退出
		// set -u 当使用未初始化变量，程序退出
		// set -o pipefail 当在管道中出现错误，程序退出
		let set_cmd = 'set -eu\n' + 'set -o pipefail\n'

		// 定义 m() 函数，覆盖原来的定义
		// 便于在 Trap Exit 里根据行号 区分 是正常命令调用，还是 m调用
		let m_func_cmd = ''
		if (this.scriptArgs[0] == 'm') {
			let sys_path = std.getenv('PATH').split(':')
			for (let i = 0; i < sys_path.length; i++) {
				let m_path = sys_path[i] + '/m'
				if (os.lstat(m_path)[1] == 0) {
					m_func_cmd = `m() {\n\t"${m_path}" "$@"\n}\n`
					break
				}
			}
		} else if (this.scriptArgs[0].endsWith('.js')) {
			// 当用 qjs /usr/local/bin/qjs.js 的方式执行
			// 子shell中无法看到父shell中的函数，所以在子shell里需要重新定义m()函数
			m_func_cmd = `m() {\n\tqjs "${scriptArgs[0]}" "$@"\n}\n`
		} else {
			m_func_cmd = `m() {\n\t"${scriptArgs[0]}" "$@"\n}\n`
		}

		let init_cmd = ''
		// workdir默认由顶层task.mk设定，使子脚本无法识别自身路径，难以引用当前目录文件
		// 新增 MK_DIR 变量，标识mk脚本所在目录
		init_cmd += `MK_DIR="${mkfile_dir}"\n`
		if (this.task_main_dir) {
			let init_file = this.task_main_dir + '/__init__.sh'
			if (os.lstat(init_file)[1] == 0) {
				init_cmd += `. ${init_file}\n`
			}
		}
		if (new_workdir) {
			// 注意加双引号，防止路径中有空格
			init_cmd += `cd "${new_workdir}"\n`
		}
		let comment_line = '\n##########################\n\n'
		let init_block_cmd_block = ''
		let init_block_start_line_num = -1
		if (init_block != null) {
			init_block_cmd_block = init_block.cmd_block.trimEnd()
			init_block_start_line_num = init_block.start_line_num
		}

		block.cmd_block = block.cmd_block.trimEnd()
		// 动态计算行号
		let shell_cmd = trap_cmd + set_cmd + m_func_cmd + init_cmd + comment_line
		let sh_task_init_block_start = shell_cmd.split(/\r?\n/).length
		let init_block_length = init_block_cmd_block.split(/\r?\n/).length
		let user_block_length = block.cmd_block.split(/\r?\n/).length
		let sh_task_user_block_start = sh_task_init_block_start + init_block_length
		let sh_all_line_end = sh_task_init_block_start + init_block_length + user_block_length
		let sh_init_block_offset = sh_task_init_block_start - init_block_start_line_num  - 1
		let sh_user_block_offset = sh_task_user_block_start - block.start_line_num - 1
		// loge(`sh_task_init_block_start = ${sh_task_init_block_start}`)
		// loge(`init_block_length = ${init_block_length}`)
		// loge(`sh_task_user_block_start = ${sh_task_user_block_start}`)
		// loge(`user_block_length = ${user_block_length}`)
		// loge(`sh_all_line_end = ${sh_all_line_end}`)
		// loge(`sh_init_block_offset = ${sh_init_block_offset}`)
		// loge(`sh_user_block_offset = ${sh_user_block_offset}`)
		shell_cmd = shell_cmd.replaceAll('@1', sh_task_init_block_start.toString())
		shell_cmd = shell_cmd.replaceAll('@2', sh_task_user_block_start.toString())
		shell_cmd = shell_cmd.replaceAll('@3', sh_all_line_end.toString())
		shell_cmd = shell_cmd.replaceAll('@4', sh_init_block_offset.toString())
		shell_cmd = shell_cmd.replaceAll('@5', sh_user_block_offset.toString())
		shell_cmd += init_block_cmd_block + '\n' + block.cmd_block

		let tag = crc16(shell_cmd)
		let tmp_dir = this.tmp_dir
		let shell_name = `${tmp_dir}/${block.tasks[0]}_${tag}.sh`
		// logd(`${shell_name}`)


		if (os.lstat(shell_name)[1] != 0) {
			let fd = std.open(shell_name, 'wb+')
			if (!fd) {
				loge(`create ${shell_name} error!`)
				return 1
			}
			fd.puts(shell_cmd)
			fd.close()
		} else {
			let fd = std.open(shell_name, 'rb')
			if (fd) {
				let data = fd.readAsString()
				if (data == shell_cmd) {
					fd.close()
				} else {
					// logi('crc16 error! mybe error.')
					fd.close()
					let fd2 = std.open(shell_name, 'wb')
					if (!fd2) {
						loge(`create ${shell_name} error!`)
						return 1
					}
					fd2.puts(shell_cmd)
					fd2.close()
				}
			} else {
				loge(`open ${shell_name} error!`)
				return 1
			}
		}
		let real_path = os.realpath(this.expand_file(this.file))[0]
		std.setenv('_TASK_CUR_DEFAULT_FILE', real_path)
		std.setenv('_TASK_ROOT_WORKDIR', this.task_root_workdir)
		std.setenv('_TASK_TMP_DIR', this.tmp_dir)
		std.setenv('_TASK_IS_SUBTASK', 1)
		// logd(bash_cmd)
		let bash_cmd = ['/bin/bash', shell_name].concat(shell_args)
		this.run_task_flag = 1
		let t0 = new Date().getTime()
		let ret = os.exec(bash_cmd)
		if (!this.is_subtask) {
			let t1 = new Date().getTime()
			logi2('Bye!' + ` (${(t1 - t0) / 1000} s)`)
		}
		return ret
	}
}


function main() {
	// example:
	// m
	// m  task
	// m  @build:task
	// m  @build:task   param1      param2
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
		ret = argInfo.do_main()
		// 过一段时间清理临时目录
		if (ret == 0 && !argInfo.is_subtask && Math.random() < 0.01) {
			let dirs_st = os.readdir(argInfo.tmp_dir)
			if (dirs_st[1] == 0) {
				dirs_st[0].forEach((f, _) => {
					if (f.endsWith('.sh')) os.remove(argInfo.tmp_dir + '/' + f)
				})
			}
		}
	}
	return ret
}

std.exit(main())
