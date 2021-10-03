GOARCH ?= amd64
GOOS ?= linux
VERSION ?= local

.PHONY: docker local clean skaffold

.DEFAULT: native

native: fluffy

skaffold: fluffy-linux-amd64

docker: fluffy-linux-amd64
	docker build -t fluffy:$(VERSION) .

publish:
	docker push fluffy:$(VERSION)

fluffy: main.go controller.go
	go build

fluffy-linux-amd64: main.go controller.go
	GOARCH=amd64 GOOS=linux go build -o fluffy-linux-amd64 -ldflags="-X 'main.Version=$(VERSION)'"

clean:
	rm -vf fluffy fluffy-linux-amd64

