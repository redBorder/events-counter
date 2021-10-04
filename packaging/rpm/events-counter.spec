Name:    events-counter
Version: %{__version}
Release: %{__release}%{?dist}

License: GNU AGPLv3
URL: https://github.com/redBorder/events-counter
Source0: %{name}-%{version}.tar.gz

BuildRequires: go glide
BuildRequires:	rsync
BuildRequires: librdkafka-devel

Requires: librdkafka1

Summary: Counts bytes of kafka topics
Group:   Development/Libraries/Go

%description
%{summary}

%prep
%setup -qn %{name}-%{version}

%build
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
cd $GOPATH/src/github.com/redBorder/events-counter
mkdir -p %{buildroot}/usr/bin
prefix=%{buildroot}/usr make install
mkdir -p %{buildroot}/usr/share/events-counter
mkdir -p %{buildroot}/etc/events-counter
install -D -m 644 events-counter.service %{buildroot}/usr/lib/systemd/system/events-counter.service
install -D -m 644 packaging/rpm/config.yml %{buildroot}/usr/share/events-counter

%clean
rm -rf %{buildroot}

%pre
getent group events-counter >/dev/null || groupadd -r events-counter
getent passwd events-counter >/dev/null || \
    useradd -r -g events-counter -d / -s /sbin/nologin \
    -c "User of events-counter service" events-counter
exit 0

%post -p /sbin/ldconfig
%postun -p /sbin/ldconfig

%files
%defattr(755,root,root)
/usr/bin/events-counter
%defattr(644,root,root)
/usr/share/events-counter/config.yml
/usr/lib/systemd/system/events-counter.service

%changelog
* Mon Oct 04 2021 Miguel Negrón <manegron@redborder.com> & David Vanhoucke <dvanhoucke@redborder.com> - 1.0.0-1
- first spec version