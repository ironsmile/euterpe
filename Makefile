.PHONY : build compress release release-no-upx install dist-archive run

# Build a normal binary for development.
all: build

# Build a release binary which could be used in the distribution archive.
build:
	go build \
		--tags "sqlite_icu" \
		-ldflags "-X github.com/ironsmile/euterpe/src/version.Version=`git describe --tags --always`" \
		-o euterpe

# Compress it somewhat. It seems that the Euterpe binary gets significantly smaller
# using upx.
compress:
	upx euterpe

# Build a release binary which could be used in the distribution archive.
release: build compress

# Build a release binary which could be used in the distribution archive but don't
# compress it with upx.
release-no-upx: build

# Install in $GOPATH/bin.
install:
	go install \
		--tags "sqlite_icu" \
		-ldflags "-X github.com/ironsmile/euterpe/src/version.Version=`git describe --tags --always`"

# Build distribution archive.
dist-archive:
	./tools/build

# Start Euterpe after building it from source.
run:
	go run --tags "sqlite_icu" main.go -D -local-fs
