all: 
	sudo docker build -t quay.io/established/sget .

push:
	sudo docker push quay.io/established/sget:latest

.PHONY: all push
