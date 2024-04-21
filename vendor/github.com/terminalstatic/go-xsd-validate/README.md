# xsdvalidate
[![GoDoc](https://godoc.org/github.com/terminalstatic/go-xsd-validate?status.svg)](https://godoc.org/github.com/terminalstatic/go-xsd-validate)

The goal of this package is to preload xsd files into memory and to validate xml (fast) using libxml2, like post bodys of xml service endpoints or api routers. At the time of writing, similar packages I found on github either didn't provide error details or got stuck under load. In addition to providing error strings it also exposes some fields of libxml2 return structs. 

# Api Reference
[https://godoc.org/github.com/terminalstatic/go-xsd-validate](https://godoc.org/github.com/terminalstatic/go-xsd-validate)

# Install
Install libxml2 dev via distribution package manager or from source, below an example how to install the latest libxml2 from source on linux(Debian/Ubuntu): 

	curl -L ftp://xmlsoft.org/libxml2/LATEST_LIBXML2 -o ./LIBXML2_LATEST.tar.gz
	tar -xf ./LIBXML2_LATEST.tar.gz
	cd ./libxml2*
	./configure --prefix=/usr  --enable-static --with-threads --with-history
	make
	sudo make install
	
Go get the package:

	go get github.com/terminalstatic/go-xsd-validate
	
# Examples
Check [this](./examples/_server/simple/simple.go) for a simple http server example and [that](./examples/_server/simpler/simpler.go) for an even simpler one. Look at [this](./examples/_server/simpler_mem/simpler_mem.go) for an example using Go's `embed` package to bake an XML schema into a simple http server.
To see how this could be plugged into middleware see the [go-chi](https://github.com/go-chi/chi) [example](./examples/_server/chi/chi.go) I came up with. 

```go
	xsdvalidate.Init()
	defer xsdvalidate.Cleanup()
	xsdhandler, err := xsdvalidate.NewXsdHandlerUrl("examples/test1_split.xsd", xsdvalidate.ParsErrDefault)
	if err != nil {
		panic(err)
	}
	defer xsdhandler.Free()

	xmlFile, err := os.Open("examples/test1_fail2.xml")
	if err != nil {
		panic(err)
	}
	defer xmlFile.Close()
	inXml, err := ioutil.ReadAll(xmlFile)
	if err != nil {
		panic(err)
	}

	// Option 1:
	xmlhandler, err := xsdvalidate.NewXmlHandlerMem(inXml, xsdvalidate.ParsErrDefault)
	if err != nil {
		panic(err)
	}
	defer xmlhandler.Free()

	err = xsdhandler.Validate(xmlhandler, xsdvalidate.ValidErrDefault)
	if err != nil {
		switch err.(type) {
		case xsdvalidate.ValidationError:
			fmt.Println(err)
			fmt.Printf("Error in line: %d\n", err.(xsdvalidate.ValidationError).Errors[0].Line)
			fmt.Println(err.(xsdvalidate.ValidationError).Errors[0].Message)
		default:
			fmt.Println(err)
		}
	}

	// Option 2:
	err = xsdhandler.ValidateMem(inXml, xsdvalidate.ValidErrDefault)
	ifT err != nil {
		switch err.(type) {
		case xsdvalidate.ValidationError:
			fmt.Println(err)
			fmt.Printf("Error in line: %d\n", err.(xsdvalidate.ValidationError).Errors[0].Line)
			fmt.Println(err.(xsdvalidate.ValidationError).Errors[0].Message)
		default:
			fmt.Println(err)
		}
	}
```

# Licence
[MIT](./LICENSE)
