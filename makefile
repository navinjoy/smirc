#!make

.PHONY: build
build:
	CGO_ENABLED=0 go build -v -o ./bin/smirc ./smirc.go

.PHONY: run
run: build
	CONFIG_FILENAME=smirc.conf IRC_NICKNAME=HelloMyNameIsGNU IRC_REALNAME=GNU IRC_USERNAME=HelloMyNameIsGNU ./bin/smirc
