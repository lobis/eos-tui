%global debug_package %{nil}

Name: eos-tui
Version: %{?pkg_version}%{!?pkg_version:0.0.0}
Release: %{?pkg_release}%{!?pkg_release:1}%{?dist}
Summary: Terminal UI for monitoring and managing EOS storage clusters
License: MIT
URL: https://github.com/lobis/eos-tui
Source0: %{name}-%{version}.tar.gz

Requires: openssh-clients

%description
EOS TUI is a terminal user interface for monitoring and managing EOS storage
clusters. It can inspect MGM, QDB, FST, filesystem, namespace, and IO shaping
state while also following logs and opening SSH sessions to cluster nodes.

%prep
%autosetup -n %{name}-%{version}

%build
CGO_ENABLED=0 /usr/local/go/bin/go build \
  -trimpath \
  -buildvcs=false \
  -ldflags="-s -w" \
  -o %{name} \
  .

%install
install -D -m 0755 %{name} %{buildroot}%{_bindir}/%{name}
install -D -m 0644 LICENSE %{buildroot}%{_licensedir}/%{name}/LICENSE
install -D -m 0644 README.md %{buildroot}%{_docdir}/%{name}/README.md

%files
%license %{_licensedir}/%{name}/LICENSE
%doc %{_docdir}/%{name}/README.md
%{_bindir}/%{name}

%changelog
* Thu Apr 09 2026 Codex <codex@openai.com> 0.0.0-1
- Initial RPM packaging for GitHub releases
