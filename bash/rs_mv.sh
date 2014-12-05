#!/bin/zsh

bucket="bokanol"
qrsctl listprefix $bucket ""  1000 | \
while read fn
do
	entry1="$bucket:$fn"
	entry2=$bucket:`echo "$fn" | cut -d'.' -f1` 
	echo "$entry1=>$entry2"
	qrsctl mv "$entry1" "$entry2"
done
