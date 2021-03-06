#!/usr/bin/env bash

SERVER="clearview.rocks"
OUTPATH="/var/tmp"

echo "*** Building ..."

if ! go build -o "${OUTPATH}/clearview-server.out" ./server/*.go
then
   	echo "*** Error"
	exit 1
fi

if ! go build -o "${OUTPATH}/clearview-agent.out" ./agent/*.go
then
   	echo "*** Error"
	exit 1
fi

echo "*** Copying to ${SERVER} ..."

if ! rsync -acvz \
    "${OUTPATH}/clearview-server.out" \
    "${OUTPATH}/clearview-agent.out" \
    "kman@${SERVER}:/var/www/bin/"
then
	echo "*** Error"
	exit 1
fi

if ! rsync -acvz \
    ./root/ \
    "kman@${SERVER}:/var/www/html/"
then
	echo "*** Error"
	exit 1
fi

if ! rsync -acvz \
    ./cv/ \
    "kman@${SERVER}:/var/www/html/cv"
then
	echo "*** Error"
	exit 1
fi

if ! rsync -acvz \
    clearview-server.service \
    clearview-agent.service \
    "root@${SERVER}:/etc/systemd/system"
then
	echo "*** Error"
	exit 1
fi

echo "*** Restarting ..."

systemctl --host "root@${SERVER}" daemon-reload
systemctl --host "root@${SERVER}" restart clearview-server clearview-agent
