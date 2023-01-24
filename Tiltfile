# -*- mode: Python -*-

load('ext://restart_process', 'docker_build_with_restart')
load('ext://helm_resource', 'helm_resource', 'helm_repo')

compile_opt = 'GO111MODULE=on CGO_ENABLED=0 GOOS=linux GOARCH=amd64 '

##################### GreenMail #####################

# Spin up greenmail
helm_resource(
  'greenmail',
  'greenmail',
  port_forwards=["3993:3993", "8080:8080"],
  labels='greenmail',
)