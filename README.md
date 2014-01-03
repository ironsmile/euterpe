httpms
======

A way to listen your music library everywhere. Once set up you won't need anything but a browser.
HTTPMS will let you browse through and listen to your music over HTTP.
Up untill now I've had a really bad time listening to my music which is stored back home.
I would create a mount over ftp, sshfs or something similar and point the local player to
the mounted library. Every time it resulted in some upleasantries. Just imagine searching
in a network mounted directory!

No more!

Requirements
======
If you want to install it from source (from here) you will need [Go](http://golang.org/) 1.1.2 or later [installed and properly configured](http://golang.org/doc/install). For the moment I do not plan to distribute it any other way.


Install
======

1. Run ```go get https://github.com/ironsmile/httpms```

2. Create config.json (or copy config.example.json) to <gopath>/github.com/ironsmile/httpms/config.json and [edit it to your](https://github.com/ironsmile/httpms/wiki/HTTPMS-configuration) liking

3. Start it with ```httpms```

Features
======

* Uses [jplayer](https://github.com/happyworm/jPlayer) to play your music so it will probably work in every browser
* Interface and media via HTTPS
* HTTP Basic Authenticate
* Playlists
* Search by track name, artist or album
