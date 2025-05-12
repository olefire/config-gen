# config-gen

A command-line tool that generates Go realtime configuration code from a YAML schema.

Solution use etcd as a centralized key-value store to keep configuration synchronized across distributed services

## Installation
`go install github.com/olefire/config-gen/cmd/config-gen`

## Usage
`config-gen -schema "$(CONFIG_FILE)"`