#!/usr/local/bin/bash

declare -A ipm

ipm=(['jp1.vpnhide.com']=0 ['jp2.vpnhide.com']=0 ['jp3.vpnhide.com']=0 
	['us1.vpnhide.com']=0 ['us2.vpnhide.com']=0  ['us3.vpnhide.com']=0 ['us4.vpnhide.com']=0 ['us5.vpnhide.com']=0 \
	 ['sg1.vpnhide.com']=0 ['sg2.vpnhide.com']=0 ['tw1.vpnhide.com']=0 )

ip_addr=''
mtr_avg=0

for ip in "${!ipm[@]}"; do
	ipm["$ip"]=$(/usr/local/sbin/mtr --no-dns --report "$ip" | tail -n +3 | awk '{avg+=$6} END{print avg/NR}')
done


for ip in "${!ipm[@]}"; do
	avg=${ipm["$ip"]}
	echo "$ip==>$avg"
	if [[ $mtr_avg == 0 ]]; then
		mtr_avg=$avg
		ip_addr="$ip"
	else
		if (( $(bc <<< "$mtr_avg > $avg") ==1 )); then
			mtr_avg=$avg
			ip_addr="$ip"
		fi
	fi
done

echo "The Best:"${ip_addr}"==>>"$mtr_avg


function connect {
/usr/bin/env osascript <<-EOF
tell application "System Events"
        tell current location of network preferences
                set VPN to service "$1" -- your VPN name here
                if exists VPN then connect VPN
                repeat while (current configuration of VPN is not connected)
                    delay 1
                end repeat
        end tell
end tell
EOF
}

function disconnect {
/usr/bin/env osascript <<-EOF
tell application "System Events"
        tell current location of network preferences
                set VPN to service "$1" -- your VPN name here
                if exists VPN then disconnect VPN
        end tell
end tell
return
EOF
}



connect "${ip_addr}"
