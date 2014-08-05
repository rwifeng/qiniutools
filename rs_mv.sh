#!/usr/bin/env bash

bucket="mingshi-mp4"
qboxrsctl listprefix mingshi-mp4 "download/mp4bsdb/"  1000 | \
while read fn
do
	if [[ $fn == download/mp4bsdb* ]];
	then
	       entry1="$bucket:$fn"
	       entry2=$bucket:`echo "$fn"| cut -d '/' -f 3` 
	       echo "$entry1=>$entry2"
		qboxrsctl mv "$entry1" "$entry2"
	fi
done
