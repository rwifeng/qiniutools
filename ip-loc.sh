#!/usr/local/bin/bash

usage () {
	echo "Usage: $(basename $0) <path_to_file>"
	exit 1
}


declare -A ipm

traverse () {
	while read line; do
		#echo $line"
		process "$line"
	done < "$1"
}


process () {
	ip=$(grep -Eo '([0-9]{1,3}[\.]){3}[0-9]{1,3}' <<< "$1")
	line="$1"
	if [[ ${ip} == "" ]]; then
		echo "$line"
	else
		iparr=(`echo $ip`)
		for i in "${iparr[@]}"; do
			ip_loc="$i["$(iploc "$i")"]"
			line=$(sed s/"$i"/"$ip_loc"/g <<< "$line")
		done
		echo "$line"
	fi
}

#args: ip
iploc () {
	loc=${ipm["$1"]}
	if [[ $loc == "" ]]; then
		loc=$(loc_ "$1")
		ipm["$1"]=$loc
	fi
	echo $loc
}

loc_ () {
	url="http://ip.taobao.com/service/getIpInfo.php?ip=$1"
	ret=$(curl -s $url | jq .data) 
	info=$(jq '.country, .city, .county, .isp' <<< $ret)

	echo $(sed s/\"//g <<< $info | sed s/'中国'//g)
}

if [[ $# -ne 1 ]]; then
	usage
fi

traverse "$1"

for i in "${!ipm[@]}"; do
	echo "$i";
done
