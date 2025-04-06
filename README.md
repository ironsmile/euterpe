Euterpe
======

<img src="images/heavy-metal-128.png" alt="Euterpe Icon" align="left" style="margin-right: 20px" title="Euterpe Icon" />

Euterpe is self-hosted streaming service for music.

A way to listen to your music library from everywhere. Once set up you won't need anything but a browser. Think of it as your own Spotify service over which you have full control. Euterpe will let you browse through and listen to your music over HTTP(s). Up until now I've had a really bad time listening to my music which is stored back home. I would create a mount over ftp, sshfs or something similar and point the local player to the mounted library. Every time it resulted in some unpleasantry. Just imagine searching in a network mounted directory!

No more!

[![Build Status](https://github.com/ironsmile/euterpe/actions/workflows/build-test-lint.yml/badge.svg?branch=master)](https://github.com/ironsmile/euterpe/actions/workflows/build-test-lint.yml?query=branch%3Amaster) [![GoDoc](https://pkg.go.dev/badge/github.com/ironsmile/euterpe)](https://godoc.org/github.com/ironsmile/euterpe) [![Go Report Card](https://goreportcard.com/badge/github.com/ironsmile/euterpe)](https://goreportcard.com/report/github.com/ironsmile/euterpe) [![Coverage Status](https://coveralls.io/repos/github/ironsmile/euterpe/badge.svg?branch=master)](https://coveralls.io/github/ironsmile/euterpe?branch=master)

* [Web UI](#web-ui)
* [Features](#features)
* [Demo](#demo)
* [Requirements](#requirements)
* [Install](#install)
* [Docker Image](#docker)
* [Configuration](#configuration)
* [As an API](#as-an-api)
* [OSX Media Keys Control](#media-keys-control-for-osx)
* [Clients](#clients)
* [Change Log](CHANGELOG.md)
* [Name Change](#name-change)


Web UI
======

Have a taste of how its web interface looks like

![Euterpe Screenshot](images/euterpe-preview.webp)

It comes with a custom [jPlayer](https://github.com/happyworm/jPlayer) which can handle playlists with thousands of songs. Which is [an improvement](https://github.com/jplayer/jPlayer/pull/192) over the original which never included this performance patch.

I feel obliged to say that the music on the screenshot is written and performed by my close friend [Velislav Ivanov](http://www.progarchives.com/artist.asp?id=4264).


Features
======

* Simple. It is just one binary, that's it! You don't need to faff about with interpreters or web servers
* Fast. A typical response time on my more than a decade old mediocre computer is 26ms for a fairly large collection
* Supports the most common audio formats such as mp3, oga, ogg, wav, flac, opus, web and m4a audio formats
* Built-in fast and simple Web UI so that you can play your music on every device
* Media and UI could be served over HTTP(S) natively without the need for other software
* User authentication (HTTP Basic, query token, Bearer token)
* Media artwork from local files or automatically downloaded from the [Cover Art Archive](https://musicbrainz.org/doc/Cover_Art_Archive)
* Artist images could be downloaded automatically from [Discogs](https://www.discogs.com/)
* Search by track name, artist or album
* Download whole album in a zip file with one click
* Controllable via media keys in OSX with the help of [BeardedSpice](https://beardedspice.github.io/)
* Extensible via [stable API](#as-an-api)
* Multiple [clients and player plugins](#clients)
* Uses [jplayer](https://github.com/happyworm/jPlayer) to play your music on really old browsers

Demo
======

Just show, don't talk, will ya? I will! You may take the server a spin with the [live demo](https://listen-to-euterpe.eu/demo) if you would like to. Feel free to thank all the artists who made their music available for this!

Requirements
======
If you want to install it from source you will need:

* [Go](http://golang.org/) 1.21 or later [installed and properly configured](http://golang.org/doc/install).

* [taglib](https://taglib.org/) - Read the [install instructions](https://github.com/taglib/taglib/blob/master/INSTALL.md) or better yet the one inside your downloaded version. Most operating systems will have it in their package manager, though. Better use this one.

* [International Components for Unicode](http://site.icu-project.org/) - The Euterpe binary dynamically links to `libicu`. Your friendly Linux distribution probably already has a package. For other OSs one should [go here](http://site.icu-project.org/download).

Install
======

The safest route is installing [one of the releases](https://github.com/ironsmile/euterpe/releases).

#### Linux & macOS

If you have [one of the releases](https://github.com/ironsmile/euterpe/releases) (for example `euterpe_1.1.0_linux.tar.gz`) it includes an `install` script which would install Euterpe in `/usr/bin/euterpe`. You will have to uninstall any previously installed versions first. An `uninstall` script is provided as well.

#### Windows

Automatically creating a release version for Windows is in progress at the moment. For the time being check out the next section, "From Source". Pay attention to the [requirements](#requirements) section above. As of writing this the author hasn't been yet initiated in the secret art of building and installing libraries on Windows so you are on your own.

#### From Source (any OS)

If installing from source running `go install` in the project root directory will compile `euterpe` and move its binary in your `$GOPATH`. Releases from `v1.0.1` onward have their go dependencies vendored in.

So, to install the `master` branch, you can just run

```
go install github.com/ironsmile/euterpe
```

Or alternatively, if you want to produce a release version you will have to get the repository. Then in the root of the project run

```
make release
```

This will produce a binary `euterpe` which is ready for distribution. Check its version with

```
./euterpe -v
```

First Run
======

Once installed, you are ready to use your media server. After its initial run it will create a configuration file which you will have to edit to suit your needs.

1. Start it with ```euterpe```

2. [Edit the config.json](#configuration) and add your library paths to the "library" field. This is an *important* step. Without it, `euterpe` will not know where your media files are.


Docker
======

Alternatively to installing everything in your environment you can use the [Docker image](https://hub.docker.com/r/ironsmile/euterpe).

Start the server by running:

```sh
docker run -v "${HOME}/Music/:/root/Music" -p 8080:9996 -d ironsmile/euterpe:latest euterpe
```

Then point your browser to [https://localhost:8080](https://localhost:8080) and you will see the Euterpe web UI. The `-v` flag in the Docker command will mount your `$HOME/Music` directory to be discoverable by Euterpe.


### Building the Image Yourself

You can use the [Dockerfile](Dockerfile) in this repository to build the image yourself.

```docker build -t ironsmile/euterpe github.com/ironsmile/euterpe```

The `euterpe` binary there is placed in `/usr/local/bin/euterpe`.

Configuration
======

HTTPS configuration is saved in a JSON file, different for every user in the system. Its
location is as follows:

* Linux or BSD: ```$HOME/.euterpe/config.json```
* Windows: ```%APPDATA%\euterpe\config.json```

When started for the first time Euterpe will create one for you. Here is an example:

```js
{
    // Address and port on which Euterpe will listen. It is in the form hostname[:port]
    // For exact explanation see the Addr field in the Go's net.http.Server
    // Make sure the user running Euterpe have permission to bind on the specified
    // port number
    "listen": ":443",

    // true if you want to access Euterpe over HTTPS or false for plain HTTP.
    // If set to true the "ssl_certificate" field must be configured as well.
    "ssl": true,

    // Provides the paths to the certificate and key files. Must be full paths, not
    // relatives. If "ssl" is false this can be left out.
    "ssl_certificate": {
        "crt": "/full/path/to/certificate/file.crt",
        "key": "/full/path/to/key/file.key"
    },

    // true if you want the server to require HTTP basic authentication. Credentials
    // are set by the 'authentication' field below.
    "basic_authenticate": true,
    
    // User and password for the HTTP basic authentication.
    "authentication": {
        "user": "example",
        "password": "example"
    },

    // An array with all the directories which will be scanned for media. They must be
    // full paths and formatted according to your OS. So for example a Windows path
    // have to be something like "D:\Media\Music".
    // As expected Euterpe will need permission to read in the library folders.
    "libraries": [
        "/path/to/my/files",
        "/some/more/files/can/be/found/here"
    ],
    
    // Optional configuration on how to scan libraries. Note that this configuration
    // is applied to each library separately.
    "library_scan": {
        // Will wait this much time before actually starting to scan a library.
        // This might be useful when scanning is resource hungry operation and you
        // want to postpone it on start up.
        "initial_wait_duration": "1s",
        
        // With this option a "operation" is defined by this number of scanned files.
        "files_per_operation": 1500,

        // After each "operation", sleep this amount of time.
        "sleep_after_operation": "15ms"
    },

    // When true, Euterpe will search for images on the internet. This means album artwork
    // and artists images. Cover Art Archive is used for album artworks when none is
    // found locally. And Discogs for artist images. Anything found will be saved in
    // the Euterpe database and later used to prevent further calls to the archive.
    "download_artwork": true,

    // If download_artwork is true the server will try to find artist artwork in the
    // Discogs database. In order for this to work an authentication is required
    // with their API. This here must be a personal access token. In effect the server
    // will make requests on your behalf.
    //
    // See the API docs for more information:
    // https://www.discogs.com/developers/#page:authentication,header:authentication-discogs-auth-flow
    "discogs_auth_token": "some-personal-token",

    // If set to true, logs will include a line for every HTTP request handled by the
    // server.
    "access_log": false
}
```

List with all directives can be found in the [configuration wiki](https://github.com/ironsmile/euterpe/wiki/configuration#wiki-json-directives).

As an API
======

Check out the Euterpe API reference for:

* For the current branch at [API.md](API.md)
* The latest released Eupterpe version at the [website's docs](https://listen-to-euterpe.eu/docs/api/).

Media Keys Control For OSX
======

You can control your Euterpe web interface with the media keys the same way you can control any native media player. To achieve this a third-party program is required: [BearderSpice](https://beardedspice.github.io/). Sadly, Euterpe is [not included](https://github.com/beardedspice/beardedspice/pull/684) in the default web strategies bundled-in with the program. You will have to import the [strategy](https://github.com/beardedspice/beardedspice/tree/disco-strategy-web#writing-a-media-strategy) [file](tools/bearded-spice.js) included in this repo yourself.

How to do it:

1. Install BeardedSpice. Here's the [download link](https://beardedspice.github.io/#download)
2. Then go to BeardedSpice's Preferences -> General -> Media Controls -> Import
3. Select the [bearded-spice.js](tools/bearded-spice.js) strategy from this repo

Or with images:

BeardedSpice Preferences:

![BS Install Step 1](images/barded-spice-install-step1.png)

Select "Import" under General tab:

![BS Install Step 2](images/barded-spice-install-step2.png)

Select the [bearded-spice.js](tools/bearded-spice.js) file:

![BS Install Step 3](images/barded-spice-install-step3.png)

Then you are good to go. Smash those media buttons!


Clients
======

You are not restricted to using the web UI. The server has a RESTful API which can easily be used from other clients. I will try to keep a list with all of the known clients here:

* ~~[httpms-android](https://github.com/ironsmile/httpms-android) is a Android client for HTTPMS.~~ Long abandoned in favour of a React Native mobile client.
* [euterpe-mobile](https://github.com/ironsmile/euterpe-mobile) is an iOS/Android mobile client written with React Native.
* [euterpe-rhythmbox](https://github.com/ironsmile/euterpe-rhythmbox) is an Euterpe client plug-in for Gnome's Rhythmbox.
* [euterpe-gtk](https://github.com/ironsmile/euterpe-gtk) is a GTK client for mobile or desktop.
