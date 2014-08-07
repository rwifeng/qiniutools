#!/usr/bin/env bash

bucketsrc="mingshi-mp4"
bucketdest="mingshi-video"
prefix=""
limit=10
mkr=''

#Usage: qboxrsctl listprefix <bucket> <prefix> [<limit>] [<marker>]
IFS=$'\n'
while true; do
	fns=$(qboxrsctl listprefix $bucketsrc "" $limit $mkr)
	for fn in $fns
	do
		if [[ $fn == marker:* ]];then
			mkr=$(sed 's/marker:[ \t]*//' <<<$fn) 
		else
			entrysrc="$bucketsrc:$fn"
			entrydest="$bucketdest:$fn"
			echo "$entrysrc  =>  $entrydest"
			#Usage: qboxrsctl cp <Bucket1:KeySrc> <Bucket2:KeyDest>
			qboxrsctl cp "$entrysrc" "$entrydest"
		fi
	done 
	
	if [[ $mkr == '' ]]; then
		exit 200
	fi
done
