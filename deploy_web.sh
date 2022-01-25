#!/usr/bin/env bash

SERVER="clearview.rocks"

echo "*** Copying to ${SERVER} ..."

if ! rsync -acvz \
    ./root/ \
    "kman@${SERVER}:/var/www/html/"
then
	echo "*** Error syncing /var/www/html/"
	exit 1
fi

if ! rsync -acvz \
    ./cv/ \
    "kman@${SERVER}:/var/www/html/cv"
then
	echo "*** Error syncing /var/www/html/cv"
	exit 1
fi

