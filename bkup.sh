#!/usr/bin/env bash

bucket="rwxf"
prefix=""
limit=10
mkr=''

#Usage: qboxrsctl listprefix <bucket> <prefix> [<limit>] [<marker>]
IFS=$'\n'
while true; do
	fns=$(qboxrsctl listprefix $bucket "" $limit $mkr)
	for fn in $fns
	do
		if [[ $fn == marker:* ]];then
			mkr=$(sed 's/marker:[ \t]*//' <<<$fn) 
		else
			curl -v "http://127.0.0.1:51234/get?key=$fn" >> backup.log 2>&1
			if [[ $? != 0 ]];then
				echo "ERR===>$fn" 
			else
				echo "OK===>$fn"
			fi

		fi
	done 
	
	if [[ $mkr == '' ]]; then
		exit 200
	fi
done
