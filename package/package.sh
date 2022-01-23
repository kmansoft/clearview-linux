#!/bin/bash

OPT_ALLARCH=0
OPT_HTML=1
OPT_UPLOAD=0

ARCH_LIST="amd64"
WHAT_LIST=""

while [[ "$#" -gt 0 ]]; do case $1 in
	-a|--allarch) OPT_ALLARCH=1;;
  -h|--html) OPT_HTML=1;;
	-u|--upload) OPT_UPLOAD=1;;
	*) WHAT_LIST="${WHAT_LIST} $1";;
esac; shift; done

if [[ -z "${WHAT_LIST}" ]]
then
    WHAT_LIST="agent server"
fi

if [[ $OPT_ALLARCH -gt 0 ]]
then
    ARCH_LIST="386 amd64 arm64 arm"
fi

echo "Building: ${WHAT_LIST}"

##### Directories

SRCDIR="${PWD}"
PACKDIR="${SRCDIR}/package"
TEMPDIR_DEB="${PACKDIR}/temp-deb"
TEMPDIR_RPM="${PACKDIR}/temp-rpm"
OUTDIR="${PACKDIR}/out"

[[ -d "${TEMPDIR_DEB}" ]] && rm -rf "${TEMPDIR_DEB}"
[[ -d "${TEMPDIR_RPM}" ]] && rm -rf "${TEMPDIR_RPM}"
[[ -d "${OUTDIR}" ]] && rm -rf "${OUTDIR}"

mkdir -p "${TEMPDIR_DEB}" && echo "*** Created : ${TEMPDIR_DEB}"
mkdir -p "${TEMPDIR_RPM}" && echo "*** Created : ${TEMPDIR_RPM}"
mkdir -p "${OUTDIR}" && echo "*** Created : ${OUTDIR}"

##### Build loop

for WHAT in $WHAT_LIST
do
    mkdir -p "${TEMPDIR_DEB}/${WHAT}/etc/"
    mkdir -p "${TEMPDIR_DEB}/${WHAT}/usr/sbin/"
    mkdir -p "${TEMPDIR_DEB}/${WHAT}/usr/lib/systemd/system/"

    VERSION=`cat ${PACKDIR}/${WHAT}/VAR_VERSION | perl -ne 'chomp and print'`
    BUILD=`cat ${PACKDIR}/${WHAT}/VAR_BUILD | perl -ne 'chomp and print'`

    for ARCH in $ARCH_LIST
    do
        # Print what we're doing
        echo "***"
        echo "*** Building $WHAT for $ARCH"
        echo "***"

        # Map GO to Debian arch
        if [[ "$ARCH" == "386" ]]; then
            DEB_ARCH=i386
            RPM_ARCH=i386
        elif [[ "$ARCH" == "amd64" ]]; then
            DEB_ARCH=amd64
            RPM_ARCH=x86_64
        elif [[ "$ARCH" == "arm64" ]]; then
            DEB_ARCH=arm64
            RPM_ARCH=aarch64
        elif [[ "$ARCH" == "arm" ]]; then
            DEB_ARCH=armhf
            RPM_ARCH=armv7hl
        else
            echo "*** Unknown arch $ARCH"
            exit 1
        fi

        OUT_EXE="clearview-${WHAT}"
        OUT_DEB="clearview-${WHAT}_${VERSION}-${BUILD}_$DEB_ARCH.deb"
        OUT_RPM="clearview-${WHAT}-${VERSION}-${BUILD}.$RPM_ARCH.rpm"
        OUT_TAR="clearview-${WHAT}_${VERSION}-${BUILD}_$DEB_ARCH.tar.gz"

        echo "*** Version : ${VERSION}-${BUILD}"
        echo "*** Arch    : ${ARCH} / ${DEB_ARCH}"
        echo "*** File    : ${OUT_DEB}"

        # Build the binary

        export GOARCH="${ARCH}"

        if ! go build -o "${TEMPDIR_DEB}/${WHAT}/usr/sbin/${OUT_EXE}" "./${WHAT}/${WHAT}_main.go"
        then
            echo "*** Build error"
            exit 1
        fi

        # Build .deb

        ls -lh "${TEMPDIR_DEB}/${WHAT}/usr/sbin/${OUT_EXE}"
        file "${TEMPDIR_DEB}/${WHAT}/usr/sbin/${OUT_EXE}"

        # Copy supporting files

        cp "${PACKDIR}/${WHAT}/clearview-${WHAT}.service" "${TEMPDIR_DEB}/${WHAT}/usr/lib/systemd/system/"

        # Generate debian-binary

        echo "2.0" > "${TEMPDIR_DEB}/${WHAT}/debian-binary"

        # Generate control

        echo "Version: $VERSION-$BUILD" > "${TEMPDIR_DEB}/${WHAT}/control"
        echo "Installed-Size:" `du -sb "${TEMPDIR_DEB}/${WHAT}" | awk '{print int($1/1024)}'` >> "${TEMPDIR_DEB}/${WHAT}/control"
        echo "Architecture: $DEB_ARCH" >> "${TEMPDIR_DEB}/${WHAT}/control"
        cat "${PACKDIR}/${WHAT}/deb/control" >> "${TEMPDIR_DEB}/${WHAT}/control"

        # Copy pre/post scripts

        cp "${PACKDIR}/${WHAT}/deb/preinst" "${TEMPDIR_DEB}/${WHAT}/preinst"
        cp "${PACKDIR}/${WHAT}/deb/postinst" "${TEMPDIR_DEB}/${WHAT}/postinst"
        cp "${PACKDIR}/${WHAT}/deb/postrm" "${TEMPDIR_DEB}/${WHAT}/postrm"

        # "what" specific parts

        DIR_LIST=""

        case "${WHAT}" in
            server)
                DIR_LIST="./usr/sbin ./usr/lib/systemd/system ./etc ./var/lib/clearview-server/site/cv/web"

                mkdir -p "${TEMPDIR_DEB}/${WHAT}/var/lib/clearview-server/site/cv/web/"

                cp "${PACKDIR}/${WHAT}/clearview-server.conf" "${TEMPDIR_DEB}/${WHAT}/etc/"
                cp "${SRCDIR}/cv/web/"* "${TEMPDIR_DEB}/${WHAT}/var/lib/clearview-server/site/cv/web/"
                cp "${PACKDIR}/${WHAT}/deb/conffiles" "${TEMPDIR_DEB}/${WHAT}/conffiles"
           ;;
            agent)
                DIR_LIST="./usr/sbin ./usr/lib/systemd/system ./etc"

                cp "${PACKDIR}/${WHAT}/clearview.conf" "${TEMPDIR_DEB}/${WHAT}/etc/"
                cp "${PACKDIR}/${WHAT}/deb/conffiles" "${TEMPDIR_DEB}/${WHAT}/conffiles"
            ;;
            *) echo "Unknown software 'what' kind"
            exit 1
            ;;
        esac

        (
            # Generate md5 sums

            cd "${TEMPDIR_DEB}/${WHAT}"

            find ${DIR_LIST} -type f | while read i ; do
                md5sum "$i" | sed 's/\.\///g' >> md5sums
            done

            # Archive control

            chmod 644 control md5sums
            chmod 755 preinst postrm postinst
            fakeroot -- tar -cz -f ./control.tar.gz ./control ./md5sums ./preinst ./postinst ./postrm

            # Archive data

            fakeroot -- tar -cz -f ./data.tar.gz ${DIR_LIST}

            # Make final archive

            fakeroot -- ar -cr "../../out/${OUT_DEB}" debian-binary control.tar.gz data.tar.gz

            # Sign it

            if which debsigs 2> /dev/null
            then
                debsigs --sign=origin --default-key=20AE9981FBC18F91 "../../out/${OUT_DEB}"
            fi

            # Create a simple tar file too

            tar -cvz -f "../../out/${OUT_TAR}" ${DIR_LIST}
        )

        # RPM

        if [[ "${WHAT}" == "agent" ]]
        then

            mkdir -p "${TEMPDIR_RPM}/${WHAT}/SPECS"
            cat > "${TEMPDIR_RPM}/${WHAT}/SPECS/clearview-${WHAT}.spec" <<EOF
Summary: Clearview agent
Name: clearview-agent
Version: ${VERSION}
Release: ${BUILD}
License: GPLv2
URL: https://clearview.rocks
Group: System
Packager: Kostya Vasilyev <kmansoft@gmail.com>
Requires: libpthread
Requires: libc

%description
Clearview is an easy to install, easy to use performance monitoring for
your Linux servers. Inspired by Linode Longview.

%files
%attr(0744, root, root) /usr/sbin/*
%attr(0644, root, root) /usr/lib/systemd/system/*

%prep
echo "BUILDROOT = \$RPM_BUILD_ROOT"

mkdir -p \$RPM_BUILD_ROOT/usr/sbin/
mkdir -p \$RPM_BUILD_ROOT/usr/lib/systemd/system/

cp ${SRCDIR}/package/temp-deb/${WHAT}/usr/sbin/${OUT_EXE} \$RPM_BUILD_ROOT/usr/sbin/
cp ${SRCDIR}/package/${WHAT}/clearview-${WHAT}.service \$RPM_BUILD_ROOT/usr/lib/systemd/system/
EOF

            cat "${TEMPDIR_RPM}/${WHAT}/SPECS/clearview-${WHAT}.spec"

            rpmbuild -bb --target "${RPM_ARCH}" \
                "${TEMPDIR_RPM}/${WHAT}/SPECS/clearview-${WHAT}.spec"

            RPMROOT=`rpmbuild --eval="%_topdir" | perl -ne 'chomp and print'`

            cp "${RPMROOT}/RPMS/${RPM_ARCH}/${OUT_RPM}" "${OUTDIR}/${OUT_RPM}"

        elif [[ "${WHAT}" == "server" ]]
        then

            mkdir -p "${TEMPDIR_RPM}/${WHAT}/SPECS"
            cat > "${TEMPDIR_RPM}/${WHAT}/SPECS/clearview-${WHAT}.spec" <<EOF
Summary: Clearview server
Name: clearview-server
Version: ${VERSION}
Release: ${BUILD}
License: GPLv2
URL: https://clearview.rocks
Group: System
Packager: Kostya Vasilyev <kmansoft@gmail.com>
Requires: libpthread
Requires: libc

%description
Clearview is an easy to install, easy to use performance monitoring for
your Linux servers. Inspired by Linode Longview.

%files
%attr(0744, root, root) /usr/sbin/*
%attr(0644, root, root) /usr/lib/systemd/system/*
%attr(0644, root, root) /etc/*
%attr(0644, root, root) /var/lib/clearview-server/site/cv/web/*

%prep
echo "BUILDROOT = \$RPM_BUILD_ROOT"

mkdir -p \$RPM_BUILD_ROOT/etc
mkdir -p \$RPM_BUILD_ROOT/usr/sbin/
mkdir -p \$RPM_BUILD_ROOT/usr/lib/systemd/system/
mkdir -p \$RPM_BUILD_ROOT/var/lib/clearview-server/site/cv/web

cp ${SRCDIR}/package/${WHAT}/clearview-${WHAT}.conf \$RPM_BUILD_ROOT/etc/
cp ${SRCDIR}/package/temp-deb/${WHAT}/usr/sbin/${OUT_EXE} \$RPM_BUILD_ROOT/usr/sbin/
cp ${SRCDIR}/package/${WHAT}/clearview-${WHAT}.service \$RPM_BUILD_ROOT/usr/lib/systemd/system/
cp ${SRCDIR}/cv/web/* \$RPM_BUILD_ROOT/var/lib/clearview-server/site/cv/web/
EOF

            cat "${TEMPDIR_RPM}/${WHAT}/SPECS/clearview-${WHAT}.spec"

            rpmbuild -bb --target "${RPM_ARCH}" \
                "${TEMPDIR_RPM}/${WHAT}/SPECS/clearview-${WHAT}.spec"

            RPMROOT=`rpmbuild --eval="%_topdir" | perl -ne 'chomp and print'`

            cp "${RPMROOT}/RPMS/${RPM_ARCH}/${OUT_RPM}" "${OUTDIR}/${OUT_RPM}"

        fi

    done # ARCH
done # WHAT

# Display what we just created
ls -lh "${OUTDIR}"

# Generate download.html with the right version number / file names

if [[ $OPT_HTML -gt 0 ]]
then
  echo "***** Updating download.html"

  AGENT_VERSION=`cat ${PACKDIR}/agent/VAR_VERSION | perl -ne 'chomp and print'`
  AGENT_BUILD=`cat ${PACKDIR}/agent/VAR_BUILD | perl -ne 'chomp and print'`

  SERVER_VERSION=`cat ${PACKDIR}/server/VAR_VERSION | perl -ne 'chomp and print'`
  SERVER_BUILD=`cat ${PACKDIR}/server/VAR_BUILD | perl -ne 'chomp and print'`

  unset GOARCH

  go run \
    "$PACKDIR/download.go" \
    -template "$PACKDIR/download.html" \
    -agent-version "${AGENT_VERSION}-${AGENT_BUILD}" \
    -server-version "${SERVER_VERSION}-${SERVER_BUILD}" \
    -html "./root/download.html"
fi

# Upload to server

if [[ $OPT_UPLOAD -gt 0 ]]
then
  echo "***** Uploading to web site"

  SERVER="clearview.rocks"

  if ! rsync -acvz \
      ./root/ \
      "kman@${SERVER}:/var/www/html/"
  then
    echo "*** Error syncing /var/www/html/"
    exit 1
  fi

  if ! rsync -acvz \
      "$OUTDIR/" \
      "kman@${SERVER}:/var/www/download/"
  then
    echo "*** Error syncing /var/www/download/"
    exit 1
  fi
fi
