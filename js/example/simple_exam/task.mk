


ping_test:
	ping www.baidu.com


print_9x9:
	for ((i=1; i<10; i++)); do
		for ((j=1; j<10; j++)); do
			echo "$i x $j = $((i*j))"
		done
	done


run_sub_mod:
	## 调用子目录的模块
	echo "Root Working Directory: $(pwd)"
	m sub_mod/task.mk:1
	m -w sub_mod/task.mk:1
	echo  "=======find hello.txt========="
	find . -name "hello.txt"
	
rm_hello_txt:
	rm -f `find . -name "hello.txt"`

