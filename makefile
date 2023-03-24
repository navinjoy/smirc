#!make

.PHONY: build
build:
	CGO_ENABLED=0 go build -v -o ./bin/minirc ./minirc.go

.PHONY: run
run: build
	CONFIG_FILENAME=config.json IRC_NICKNAME=HelloMyNameIsGNU IRC_REALNAME=GNU IRC_USERNAME=HelloMyNameIsGNU ./bin/minirc
