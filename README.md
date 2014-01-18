HTTP Media Server
======

A way to listen your music library everywhere. Once set up you won't need anything but a browser.
HTTPMS will let you browse through and listen to your music over HTTP.
Up until now I've had a really bad time listening to my music which is stored back home.
I would create a mount over ftp, sshfs or something similar and point the local player to
the mounted library. Every time it resulted in some upleasantries. Just imagine searching
in a network mounted directory!

No more!

Requirements
======
If you want to install it from source (from here) you will need:

* [Go](http://golang.org/) 1.1.2 or later [installed and properly configured](http://golang.org/doc/install).

* [go-taglib](https://github.com/landr0id/go-taglib) - Read the [install notes](https://github.com/landr0id/go-taglib#install)

* [go-sqlite3](https://github.com/mattn/go-sqlite3) - `go get github.com/mattn/go-sqlite3`

For the moment I do not plan to distribute it any other way.


Install
======

1. Run ```go get https://github.com/ironsmile/httpms```

2. Start it with ```httpms```

3. [Edit the config.json](#configuration) and add your library paths to the "library" field

Features
======

* Uses [jplayer](https://github.com/happyworm/jPlayer) to play your music so it will probably work in every browser
* jplayer supports mp3 and oga
* Interface and media via HTTPS
* HTTP Basic Authenticate
* Playlists
* Search by track name, artist or album

Configuration
======

HTTPS configuration is saved in a json file, different for every user in the system. Its
location is as follows:

* Linux or BSD: ```$HOME/.httpms/config.json```
* Windows: ```%APPDATA%/httpms/config.json```

When started for the first time HTTPMS will create one for you. It will be a copy of the
default configuration with all possible fields in it. Example with all the fields explained follows:

```javascript
{
    // Address and port on which HTTPMS will listen. It is in the form hostname[:port]
    // For exact explanation see the Addr field in the Go's net.http.Server
    // Make sure the user running HTTPMS have permission to bind on the specified
    // port number
    "listen": ":443",

    // true if you want to access HTTPMS over HTTPS or false for plain HTTP.
    // If set to true the "ssl_certificate" field must be configured as well.
    "ssl": true,

    // Provides the paths to the certificate and key files. Must be full paths, not
    // relatives. If "ssl" is false this can be left out.
    "ssl_certificate": {
        "crt": "/full/path/to/certificate/file.crt",
        "key": "/full/path/to/key/file.key"
    },

    // User and password for the HTTP basic authentication. If removed no authentication
    // will be used.
    "authenticate": {
        "user": "example",
        "password": "example"
    },

    // An array with all the directories which will be scanned for media. They must be
    // full paths and formatted according to your OS. So for example a Windows path
    // have to be something like "D:\Media\Music".
    // As expected HTTPMS will need permission to read in the library folders.
    "libraries": [
        "/path/to/my/files",
        "/some/more/files/can/be/found/here"
    ]
}
```

Known Issues
======

* **Currently played song stops when new playlist is loaded** - it seems this is the designed behaviour of jplayer. No sign of option for this is in sight. If nothing changes soon I will probably patch it myself. But for now it is as it is.

* **HTTPMS will not show/play media which does not have tags** - I honestly do not know what to do with this. I lean towards not doing anything about it since it will be a hassle. Just tag your music - title and artist are the bare minimum.
