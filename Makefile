PKG = github.com/k1LoW/runn
COMMIT = $$(git describe --tags --always)
OSNAME=${shell uname -s}
ifeq ($(OSNAME),Darwin)
	DATE = $$(gdate --utc '+%Y-%m-%d_%H:%M:%S')
else
	DATE = $$(date --utc '+%Y-%m-%d_%H:%M:%S')
endif

export GO111MODULE=on

BUILD_LDFLAGS = -X $(PKG).commit=$(COMMIT) -X $(PKG).date=$(DATE)

default: test

ci: depsdev integration

test: cert
	go test ./... -coverprofile=coverage.out -covermode=count

race:
	go test ./... -race

integration: cert
	go test ./... -tags=integration -coverprofile=coverage.out -covermode=count

lint:
	golangci-lint run ./...

build:
	go build -ldflags="$(BUILD_LDFLAGS)" -o runn cmd/runn/main.go

depsdev:
	go install github.com/Songmu/ghch/cmd/ghch@latest
	go install github.com/Songmu/gocredits/cmd/gocredits@latest
	go install github.com/securego/gosec/v2/cmd/gosec@latest

cert:
	rm -f testdata/*.pem testdata/*.srl
	openssl req -x509 -newkey rsa:4096 -days 365 -nodes -sha256 -keyout testdata/cakey.pem -out testdata/cacert.pem -subj "/C=UK/ST=Test State/L=Test Location/O=Test Org/OU=Test Unit/CN=*.example.com/emailAddress=k1lowxb@gmail.com"
	openssl req -newkey rsa:4096 -nodes -keyout testdata/key.pem -out testdata/csr.pem -subj "/C=JP/ST=Test State/L=Test Location/O=Test Org/OU=Test Unit/CN=*.example.com/emailAddress=k1lowxb@gmail.com"
	openssl x509 -req -sha256 -in testdata/csr.pem -days 60 -CA testdata/cacert.pem -CAkey testdata/cakey.pem -CAcreateserial -out testdata/cert.pem -extfile testdata/openssl.cnf
	openssl verify -CAfile testdata/cacert.pem testdata/cert.pem

prerelease:
	git pull origin main --tag
	go mod tidy
	ghch -w -N ${VER}
	gocredits -w .
	cat _EXTRA_CREDITS >> CREDITS
	git add CHANGELOG.md CREDITS go.mod go.sum
	git commit -m'Bump up version number'
	git tag ${VER}

release:
	git push origin main --tag
	goreleaser --rm-dist

.PHONY: default test
