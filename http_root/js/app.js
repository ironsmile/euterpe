/*
    HTTPMS javascript
*/

$(document).ready(function(){

    // The default setPlaylist method was calling _init which did 
    // jPlayerPlaylist.select for the first track. That resulted in jplayer stopping 
    // the played song. Now we make it never select. We also set current = -1 so
    // that it will not be equal to the song currently plaing since this is a new
    // playlist after all.
    jPlayerPlaylist.prototype.setPlaylist = function(playlist) {
        this._initPlaylist(playlist);
        this._refresh();
        this.current = -1;
    };

    var cssSelector = {
        jPlayer: "#jquery_jplayer_N",
        cssSelectorAncestor: "#jp_container_N"
    };

    var playlist = [];

    var options = {
        swfPath: "/js",
        supplied: "mp3, oga, m4a, wav, fla",
        preload: "none",
        playlistOptions: {
            autoPlay: false,
            displayTime: 0,
            addTime: 0,
            removeTime: 0,
            shuffleTime: 0
        }
    };

    pagePlaylist = new jPlayerPlaylist(cssSelector, playlist, options);

    _search_timeout = null;
    _last_search = null;

    var search_with_timeout = function () {
        var search_query = $('#search').val();
        if (search_query == _last_search) {
           return;
        }

        if (_search_timeout) {
            clearTimeout(_search_timeout);
        };

        _search_timeout = setTimeout(function() {
            search_database(search_query)
        }, 500);
    };

    var search_immediately_on_enter = function (e) {
        var search_query = $('#search').val();
        var code = e.keyCode || e.which;
        if(code != 13) {
           return;
        }

        if (_search_timeout) {
            clearTimeout(_search_timeout);
        };

        search_database(search_query);
    };

    // Used when typing in the search area - it should use timeout since more
    // typing can follow immediately
    $('#search').keyup(search_with_timeout);
    $('#search').change(search_with_timeout);

    // Used when Enter is clicked - it should immediately send a request
    $('#search').keypress(search_immediately_on_enter);

    $('#album').change(function () {
        filter_playlist();
    });

    $('#artist').change(function () {
        load_filters(found_songs, {selected_artist: $('#artist').val()});
        filter_playlist();
    });

    search_database($('#search').val());
});

_ajax_query = null;

function search_database (query) {
    if (_ajax_query) {
        _ajax_query.abort();
    };

    _last_search = query;

    _ajax_query = $.ajax({
        type: "GET",
        url: encodeURI("/search/" + query),
        success: function (msg) {
            load_playlist(msg);
            load_filters(msg);
        }
    });
}

function load_playlist (songs) {

    songs.sort(function (a, b) {
        if (a.track == b.track) {
            return 0;
        };
        if (a.track < b.track) {
            return -1;
        };
        return 1;
    });

    songs.sort(function (a, b) {
        if (a.album == b.album) {
            return 0;
        };
        if (a.album < b.album) {
            return -1;
        };
        return 1;
    });

    var new_playlist = []
    for (var i = 0; i < songs.length; i++) {
        new_playlist.push({
            title: songs[i].title,
            artist: songs[i].artist,
            mp3: "/file/"+songs[i].id,
            free: true
        })
        
    };

    pagePlaylist.setPlaylist(new_playlist);
}

found_songs = [];

function load_filters(songs, opts) {
    found_songs = songs;

    opts = opts || {};
    opts.selected_artist = opts.selected_artist || false;
    opts.selected_album = opts.selected_album || false;

    var artist_elem = $('#artist');
    var album_elem = $('#album');

    var all_artists = {}, all_artists_list = []
    var all_albums = {}, all_albums_list = []

    for (var i = 0; i < songs.length; i++) {
        if (all_artists[songs[i].artist] == undefined) {
            all_artists[songs[i].artist] = true;
            all_artists_list.push(songs[i].artist)    
        };
        
        if (all_albums[songs[i].album] == undefined) {
            if (opts.selected_artist && opts.selected_artist != songs[i].artist) {
                continue;
            };
            all_albums[songs[i].album] = true;
            all_albums_list.push(songs[i].album)    
        };
    };

    artist_elem.empty();
    album_elem.empty();

    var all_artists_opt = $('<option></option>').html("All").val("");
    if (!opts.selected_artist) {
        all_artists_opt.attr("selected", 1);
    };
    artist_elem.append(all_artists_opt);

    var all_albums_opt = $('<option></option>').html("All").val("");
    if (!opts.selected_album) {
        all_albums_opt.attr("selected", 1);
    };
    album_elem.append(all_albums_opt);

    all_artists_list.sort(alpha_sort)
    all_albums_list.sort(alpha_sort)
    
    for (var i = 0; i < all_artists_list.length; i++) {
        var artist = all_artists_list[i];
        var option = $('<option></option>');
        option.html(artist).val(artist);
        if (opts.selected_artist && opts.selected_artist == artist) {
            option.attr("selected", 1);
        };
        artist_elem.append(option);
    };
    
    for (var i = 0; i < all_albums_list.length; i++) {
        var album = all_albums_list[i];
        var option = $('<option></option>');
        option.html(album).val(album);
        if (opts.selected_album && opts.selected_album == album) {
            option.attr("selected", 1);
        };
        album_elem.append(option);
    };
}

function filter_playlist () {

    var artist_filter = function (song) {return true;};
    var album_filter = function (song) {return true;};

    var selected_artist = $('#artist :selected').val();
    if (selected_artist) {
        artist_filter = function (song) {
            if (selected_artist == song.artist) {
                return true
            };
            return false;
        }
    };

    var selected_album = $('#album :selected').val();
    if (selected_album) {
        album_filter = function (song) {
            if (selected_album == song.album) {
                return true
            };
            return false;
        }
    };

    var to_load = []
    for (var i = 0; i < found_songs.length; i++) {
        if (!artist_filter(found_songs[i])) {
            continue;
        }

        if (!album_filter(found_songs[i])) {
            continue;
        }

        to_load.push(found_songs[i]);
    };

    load_playlist(to_load);
}

function alpha_sort (a, b) {
    a = a.toLowerCase()
    b = b.toLowerCase()
    if (a == b) {
        return 0;
    };
    return (a < b) ? -1 : 1;
}
