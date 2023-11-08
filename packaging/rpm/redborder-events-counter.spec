Name:    redborder-events-counter
Version: %{__version}
Release: %{__release}%{?dist}

License: GNU AGPLv3
URL: https://github.com/redBorder/events-counter
Source0: %{name}-%{version}.tar.gz

BuildRequires: go rsync gcc git pkgconfig librd-devel librdkafka-devel
Requires: librd0 librdkafka

Summary: Counts bytes of kafka topics
Group:   Development/Libraries/Go

%description
%{summary}

%prep
%setup -qn %{name}-%{version}

%build
export PKG_CONFIG_PATH=/usr/lib/pkgconfig
export GOPATH=${PWD}/gopath
export PATH=${GOPATH}:${PATH}

mkdir -p $GOPATH/src/github.com/redBorder/events-counter
rsync -az --exclude=gopath/ ./ $GOPATH/src/github.com/redBorder/events-counter
cd $GOPATH/src/github.com/redBorder/events-counter
make

%install
export PARENT_BUILD=${PWD}
export GOPATH=${PWD}/gopath
export PATH=${GOPATH}:${PATH}
export PKG_CONFIG_PATH=/usr/lib64/pkgconfig
cd $GOPATH/src/github.com/redBorder/events-counter
mkdir -p %{buildroot}/usr/bin
prefix=%{buildroot}/usr PKG_CONFIG_PATH=/usr/lib/pkgconfig/ make install
mkdir -p %{buildroot}/usr/share/redborder-events-counter
mkdir -p %{buildroot}/etc/redborder-events-counter
install -D -m 644 redborder-events-counter.service %{buildroot}/usr/lib/systemd/system/redborder-events-counter.service
install -D -m 644 packaging/rpm/config.yml %{buildroot}/usr/share/redborder-events-counter

%clean
rm -rf %{buildroot}

%pre
getent group redborder-events-counter >/dev/null || groupadd -r redborder-events-counter
getent passwd redborder-events-counter >/dev/null || \
    useradd -r -g redborder-events-counter -d / -s /sbin/nologin \
    -c "User of redborder-events-counter service" redborder-events-counter
exit 0

%post -p /sbin/ldconfig
%postun -p /sbin/ldconfig
systemctl daemon-reload

%files
%defattr(755,root,root)
/usr/bin/redborder-events-counter
%defattr(644,root,root)
/usr/share/redborder-events-counter/config.yml
/usr/lib/systemd/system/redborder-events-counter.service

%changelog
* Wed Oct 04 2023 David Vanhoucke <dvanhoucke@redborder.com> - 2.0.0-1
- adapt for go mod
* Mon Oct 04 2021 Miguel Negrón <manegron@redborder.com> & David Vanhoucke <dvanhoucke@redborder.com> - 1.0.0-1
- first spec version