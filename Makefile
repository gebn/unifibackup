OUT := unifibackup

# ensure these are set outside CI context
GOOS := $(shell go env GOOS)
GOARCH := $(shell go env GOARCH)
GOARM := $(shell go env GOARM)

VERSION := $(shell git describe --always --tags --dirty)
ARCHIVE_BASE := $(OUT)-$(VERSION:v%=%).$(GOOS)-$(GOARCH)

ifeq ($(GOOS), windows)
	OUT := $(OUT).exe
	ARCHIVE := $(ARCHIVE_BASE).zip
else
	ARCHIVE := $(ARCHIVE_BASE).tar.gz
endif

LDFLAGS := -ldflags=" \
-X 'github.com/gebn/go-stamp/v2.User=$(USER)' \
-X 'github.com/gebn/go-stamp/v2.Host=$(shell hostname)' \
-X 'github.com/gebn/go-stamp/v2.timestamp=$(shell date +%s)' \
-X 'github.com/gebn/go-stamp/v2.Commit=$(shell git rev-parse HEAD)' \
-X 'github.com/gebn/go-stamp/v2.Branch=$(shell git rev-parse --abbrev-ref HEAD)' \
-X 'github.com/gebn/go-stamp/v2.Version=$(VERSION)'"

# just build a binary
build:
	go build $(LDFLAGS) -o $(OUT)

# create the full-blown archive
dist: build
	mkdir $(ARCHIVE_BASE)
	mv $(OUT) $(ARCHIVE_BASE)/
	cp LICENSE $(ARCHIVE_BASE)/
ifeq ($(GOOS), windows)
	zip -r $(ARCHIVE) $(ARCHIVE_BASE)
else
	tar -czf $(ARCHIVE) $(ARCHIVE_BASE)
endif
	rm -r $(ARCHIVE_BASE)

# allows CI to get the path of the archive created by dist
distpath:
	@echo $(ARCHIVE)

clean:
	rm -f $(OUT) $(ARCHIVE)
