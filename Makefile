# Build a normal binary for development.
all:
	go build -ldflags "-X github.com/ironsmile/httpms/src.Version=`git describe --tags --always`"

# Build a release binary which could be used in the distribution archive.
release:
	# Build with packer in order to produce a single binary
	packr build \
		-ldflags "-X github.com/ironsmile/httpms/src.Version=`git describe --tags --always`" \
		-o httpms

	# Compress it somewhat. It seems that the HTTPMS binary gets more than 3 times smaller
	# using upx.
	upx httpms

# Install in $GOPATH/bin.
install:
	packr install -ldflags "-X github.com/ironsmile/httpms/src.Version=`git describe --tags --always`"

# Build distribution archive.
dist-archive:
	./tools/build

# Start HTTPMS after building it from source.
run:
	go run main.go -D
