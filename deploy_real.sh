#!/usr/bin/env bash

SERVER="clearview.rocks"
OUTPATH="/var/tmp"

echo "*** Building ..."

if ! go build -o "${OUTPATH}/clearview-server.out" ./server/*.go
then
   	echo "*** Error building clearview-server.out"
	exit 1
fi

if ! go build -o "${OUTPATH}/clearview-agent.out" ./agent/*.go
then
   	echo "*** Error building clearview-agent.out"
	exit 1
fi

echo "*** Copying to ${SERVER} ..."

if ! rsync -acvz \
    "${OUTPATH}/clearview-server.out" \
    "${OUTPATH}/clearview-agent.out" \
    "kman@${SERVER}:/var/www/bin/"
then
	echo "*** Error syncing /var/www/bin/"
fi

if ! rsync -acvz \
    ./root/ \
    "kman@${SERVER}:/var/www/html/"
then
	echo "*** Error syncing /var/www/html/"
fi

if ! rsync -acvz \
    ./cv/ \
    "kman@${SERVER}:/var/www/html/cv"
then
	echo "*** Error syncing /var/www/html/cv"
fi

if ! rsync -acvz \
    clearview-server.service \
    clearview-agent.service \
    "root@${SERVER}:/etc/systemd/system"
then
	echo "*** Error syncing /etc/systemd/system"
fi

if [ -d ./package/out ]
then
	if ! rsync -acvz \
	    ./package/out/ \
	    "kman@${SERVER}:/var/www/download"
	then
		echo "*** Error syncing /var/www/download"
	fi
fi


echo "*** Restarting ..."

systemctl --host "root@${SERVER}" daemon-reload
systemctl --host "root@${SERVER}" restart clearview-server clearview-agent
