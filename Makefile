MKL_RED?=	\033[031m
MKL_GREEN?=	\033[032m
MKL_YELLOW?=	\033[033m
MKL_BLUE?=	\033[034m
MKL_CLR_RESET?=	\033[0m

BIN=      redborder-events-counter
prefix?=  /usr/local
bindir?=	$(prefix)/bin

VERSION?= $(shell git describe --tags --always --dirty=-dev)

LDFLAGS+=	'-X "main.version=$(VERSION)"
LDFLAGS+=	-X "main.PubKey=$(PUBLIC_KEY)"'

build: vendor
	@printf "$(MKL_YELLOW)[BUILD]$(MKL_CLR_RESET)    Building project\n"
	@go build -ldflags $(LDFLAGS) -o $(BIN) ./cmd
	@printf "$(MKL_YELLOW)[BUILD]$(MKL_CLR_RESET)    $(BIN) created\n"

install: build
	@printf "$(MKL_YELLOW)[INSTALL]$(MKL_CLR_RESET)  Installing  $(BIN) to $(bindir)\n"
	@install $(BIN) $(bindir)
	@printf "$(MKL_YELLOW)[INSTALL]$(MKL_CLR_RESET)  Installed\n"

uninstall:
	@printf "$(MKL_RED)[UNINSTALL]$(MKL_CLR_RESET)  Remove $(BIN) from $(bindir)\n"
	@rm $(bindir)/$(BIN)
	@printf "$(MKL_RED)[UNINSTALL]$(MKL_CLR_RESET)  Removed\n"

PACKAGE_LIST := $(shell glide novendor)
tests: vendor
	@printf "$(MKL_GREEN)[TESTING]$(MKL_CLR_RESET)  Running tests\n"
	@go test -race -v $(PACKAGE_LIST)

coverage:
	@printf "$(MKL_BLUE)[COVERAGE]$(MKL_CLR_RESET) Computing coverage\n"
	@echo "mode: count" > coverage.out
	@go test -covermode=count -coverprofile=counter.part ./counter
	@go test -covermode=count -coverprofile=monitor.part ./monitor
	@go test -covermode=count -coverprofile=producer.part ./producer
	@grep -h -v "mode: count" *.part >> coverage.out
	@go tool cover -func coverage.out

GLIDE := $(shell command -v glide 2> /dev/null)
vendor:
ifndef GLIDE
	$(error glide is not installed)
endif
	@printf "$(MKL_BLUE)[DEPS]$(MKL_CLR_RESET)  Resolving dependencies\n"
	@glide install

clean:
	rm -f $(BIN)
	rm -f coverage.out

vendor-clean:
	rm -rf vendor/

all: build tests coverage

rpm: clean
	$(MAKE) -C packaging/rpm
