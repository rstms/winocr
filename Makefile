# go makefile

program := $(notdir $(shell pwd))
version := $(shell cat VERSION)

include make/common.mk

buildflags = -o $(binary) -buildmode exe

GOSRC = go.sum $(shell find . -type f -name '*.go')

default: build

build: $(binary)

go.mod:
	go mod init

go.sum: go.mod
	go mod tidy

$(binary): .fmt
	fix go build ./...
	go build $(buildflags) main.go

.fmt: $(GOSRC) 
	fix go fmt . ./...
	touch $@

fmt: .fmt

install: $(binary)
	go install 

test: fmt
	go test -v -failfast . ./...

debug: fmt
	go test -v -failfast -count=1 -run $(test) . ./...

release:
	$(gitclean)
	@$(if $(update),gh release delete -y v$(version),)
	gh release create v$(version) --notes "v$(version)"
	$(MAKE) build

dist: dist/$(release_binary)

dist/$(release_binary): $(binary)
	mkdir -p dist
	scp $< $(dist_target)/$(release_binary)
	scp $< $(dist_target)/$(dist_binary)
	cp $< $@

release-upload: dist
	cd dist; gh release upload $(latest_release) $(release_binary) $(CLOBBER)

update-modules:
	@echo checking dependencies for updated versions 
	@$(foreach module,$(rstms_modules),go get $(module)@$(call latest_module_release,$(module)) ;)
	curl -Lso .proxy https://raw.githubusercontent.com/rstms/go-common/master/proxy_common_go
	@$(foreach s,$(common_go),sed <.proxy >$(s) 's/^package cmd/package $(lastword $(subst /, ,$(dir $(s))))/' ;)
	-rm .proxy

clean:
	rm -f $(binary) *.core .fmt
	go clean
	rm -rf dist && mkdir dist
	rm -f go.mod go.sum

sterile: clean
	go clean -i || true
	go clean
	go clean -cache
	go clean -modcache

show-vars:
	@$(foreach var,$(all_variables),echo $(var)=$($(var));)
