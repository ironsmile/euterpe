/*
    HTTPMS javascript
*/

$(document).ready(function(){
    
    var cssSelector = {
        jPlayer: "#jquery_jplayer_N",
        cssSelectorAncestor: "#jp_container_N"
    };

    var playlist = [];

    var options = {
        swfPath: "/js",
        supplied: "oga, mp3",
        playlistOptions: {
            displayTime: 0,
            addTime: 0,
            removeTime: 0,
            shuffleTime: 0
        }
    };

    pagePlaylist = new jPlayerPlaylist(cssSelector, playlist, options);

    _search_timeout = null;

    $('#search').keypress(function () {
        if (_search_timeout) {
            clearTimeout(_search_timeout);
        };

        _search_timeout = setTimeout(function() {
            search_database($('#search').val())
        }, 500);
    });

    $('#search').keypress(function (e) {
        var code = e.keyCode || e.which;
        if(code != 13) {
           return;
        }

        if (_search_timeout) {
            clearTimeout(_search_timeout);
        };

        search_database($('#search').val());
    });

    $('.filter-list').each(function (ind, el) {

        $(el).change(function () {
            filter_playlist();
        });

    });

});

ajax_query = null;

function search_database (query) {
    if (ajax_query) {
        ajax_query.abort();
    };

    ajax_query = $.ajax({
        type: "GET",
        url: encodeURI("/search/" + query),
        success: function (msg) {
            load_playlist(msg);
            load_filters(msg);
        }
    });
}

function load_playlist (songs) {
    pagePlaylist.remove();

    songs.sort(function (a, b) {
        if (a.track == b.track) {
            return 0;
        };
        if (a.track < b.track) {
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

function load_filters(songs) {
    found_songs = songs;
    var all_artists = {}
    var all_albums = {}

    for (var i = 0; i < songs.length; i++) {
        all_artists[songs[i].artist] = true;
        all_albums[songs[i].album] = true;
    };

    $('#artist').empty();
    $('#album').empty();

    $('#artist').append($('<option></option>').html("All").attr("selected", 1).val(""));
    $('#album').append($('<option></option>').html("All").attr("selected", 1).val(""));

    for (artist in all_artists) {
        var option = $('<option></option>');
        option.html(artist).val(artist);
        $('#artist').append(option);
    }

    for (album in all_albums) {
        var option = $('<option></option>');
        option.html(album).val(album);
        $('#album').append(option);
    }
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
