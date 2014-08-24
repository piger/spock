#!/bin/bash

set -e

# http://getbootstrap.com/
BOOTSTRAP_VERSION="3.2.0-dist"
BOOTSTRAP_URL="https://github.com/twbs/bootstrap/releases/download/v3.2.0/bootstrap-${BOOTSTRAP_VERSION}.zip"

# http://bootflat.github.io/
BOOTFLAT_URL="https://github.com/Bootflat/Bootflat.github.io/archive/master.zip"

# http://ace.c9.io
ACE_VERSION="1.1.6"
ACE_URL="https://github.com/ajaxorg/ace-builds/archive/v${ACE_VERSION}.zip"


DESTDIR="$1"

if [[ -z $DESTDIR ]]; then
    echo "Usage: $0 <static dir>"
    exit 1
fi

if [[ -d $DESTDIR ]]; then
    echo "DESTDIR already exists"
    exit 1
fi

TMP_DIR=$(mktemp -d -t vandine-static)
trap "rm -rf $TMP_DIR" EXIT

mkdir $DESTDIR
mkdir $DESTDIR/css
mkdir $DESTDIR/js
mkdir $DESTDIR/fonts


curl -L -o $TMP_DIR/bootstrap.zip $BOOTSTRAP_URL
curl -L -o $TMP_DIR/bootflat.zip $BOOTFLAT_URL
curl -L -o $TMP_DIR/ace.zip $ACE_URL

for f in bootstrap.zip bootflat.zip ace.zip; do
    unzip -q -d $DESTDIR $TMP_DIR/$f
done

mv $TMP_DIR/ace-builds-${ACE_VERSION}/src-min $DESTDIR/ace
mv $TMP_DIR/bootstrap-${BOOTSTRAP_VERSION}/css/* $DESTDIR/css/
mv $TMP_DIR/bootstrap-${BOOTSTRAP_VERSION}/js/* $DESTDIR/js/
mv $TMP_DIR/bootstrap-${BOOTSTRAP_VERSION}/fonts/* $DESTDIR/fonts/
mv $TMP_DIR/bootflat.github.io-master/css/bootflat.css $DESTDIR/css/
