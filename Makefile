.PHONY: all controller

all: controller

controller: 
	go build -o _output/bin/snapshot-controller cmd/snapshot-controller/snapshot-controller.go
