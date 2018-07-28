all:
	go build -ldflags "-X github.com/ironsmile/httpms/src.Version=`git describe --tags --always`"

release:
	go build -ldflags "-X github.com/ironsmile/httpms/src.Version=`git describe --tags --always`"

install:
	go install -ldflags "-X github.com/ironsmile/httpms/src.Version=`git describe --tags --always`"

run:
	go run main.go -D
