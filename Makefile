test:
	python -m unittest discover -vv . -p '*.py'
lint:
	flake8 .
apt-update:
	sudo apt-get -qq update
juju-ci-tools.common_0.1.0-0_all.deb: apt-update
	sudo apt-get install -y equivs
	equivs-build juju-ci-tools-common
install-deps: juju-ci-tools.common_0.1.0-0_all.deb apt-update
	sudo dpkg -i juju-ci-tools.common_0.1.0-0_all.deb || true
	sudo apt-get install -y -f
	sudo apt-get install -y juju-local juju
.PHONY: lint test apt-update install-deps
