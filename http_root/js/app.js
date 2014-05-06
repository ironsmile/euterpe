/*
    HTTPMS javascript
*/

$(document).ready(function(){

    // The default setPlaylist method was calling _init which did 
    // jPlayerPlaylist.select for the first track. That resulted in jplayer stopping 
    // the played song. Now we make it never select. We also set current = undef so
    // that it will not be equal to the song currently plaing since this is a new
    // playlist after all.
    jPlayerPlaylist.prototype.setPlaylist = function(playlist) {
        this._initPlaylist(playlist);
        this._refresh(true);
        this.current = undefined;
    };

    // The default implementation of _updateControls was making a show/hide
    // on a class which is present in every entry in the playlist. This takes
    // too much time if the palylist has many items.
    jPlayerPlaylist.prototype._updateControls = function() {
        if(this.shuffled) {
            $(this.cssSelector.shuffleOff).show();
            $(this.cssSelector.shuffle).hide();
        } else {
            $(this.cssSelector.shuffleOff).hide();
            $(this.cssSelector.shuffle).show();
        }
    };

    // This function was was making repetitive DOM searches.
    // See my pull request for this: https://github.com/happyworm/jPlayer/pull/192
    jPlayerPlaylist.prototype._refresh = function(instant) {
        /* instant: Can be undefined, true or a function.
         *  undefined -> use animation timings
         *  true -> no animation
         *  function -> use animation timings and excute function at half way point.
         */
        var self = this;
        var playlist_el = $(self.cssSelector.playlist + " ul");

        if(instant && !$.isFunction(instant)) {
            $(playlist_el).empty();
            
            $.each(this.playlist, function(i) {
                playlist_el.append(self._createListItem(self.playlist[i]));
            });
            this._updateControls();
        } else {
            var displayTime = playlist_el.children().length ?
                    this.options.playlistOptions.displayTime : 0;
            playlist_el.slideUp(displayTime, function() {
                var $this = $(this);
                $(this).empty();
                
                $.each(self.playlist, function(i) {
                    $this.append(self._createListItem(self.playlist[i]));
                });
                self._updateControls();
                if($.isFunction(instant)) {
                    instant();
                }
                if(self.playlist.length) {
                    $(this).slideDown(self.options.playlistOptions.displayTime);
                } else {
                    $(this).show();
                }
            });
        }
    };

    /*
    *   My own create list item. I am adding track number, album and an album
    *   download button.
    */
    jPlayerPlaylist.prototype._createListItem = function(media) {
        var self = this;
        var options = this.options.playlistOptions;

        // Wrap the <li> contents in a <div>
        var listItem = "<li><div>";

        // Create links to free media
        if(media.free) {
            var first = true;
            listItem += "<span class='" + options.freeGroupClass;
            listItem += "'>(";
            $.each(media, function(property,value) {
                // Check property is a media format.
                if($.jPlayer.prototype.format[property]) {
                    if(first) {
                        first = false;
                    } else {
                        listItem += " | ";
                    }
                    listItem += "<a class='" + options.freeItemClass;
                    listItem += "' href='" + value + "' title='download media' >" +
                                    property + "</a>";
                }
            });
            listItem += ")</span>";
        }

        if (media.album) {
            listItem += " <span class='" + options.freeGroupClass + "'>" +
                    "<a href='/album/"+media.album_id+"' title='download album' "+
                    "target='_blank'>" +  media.album + "</a></span>";
        }
        

        // The title is given next in the HTML otherwise the float:right on the
        // free media corrupts in IE6/7
        listItem += "<a href='javascript:;' class='" + options.itemClass;
        listItem += "'>" + (media.number ? media.number + '. ' : "") + media.title;
        listItem += (media.artist ? 
                        " <span class='jp-artist'>by "+media.artist+"</span>" : "");
        listItem += "</a>";
        listItem += "</div></li>";

        return listItem;
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

        if (_search_timeout) {
            clearTimeout(_search_timeout);
        };

        if (search_query == _last_search) {
           return;
        }

        if (search_query.length < 1) {
            return;
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

    // Show song title when one is playing
    $("#jquery_jplayer_N").bind($.jPlayer.event.play, function(event) {
        var media = event.jPlayer.status.media;

        if (!media) {
            return;
        };

        _currently_playing = media.media_id;

        document.title = media.title + ' by ' + media.artist + ' | HTTPMS';
    });

    // Restores the normal title when nothing is played
    var restore_title = function(event) {
        document.title = 'HTTPMS';
    };

    $("#jquery_jplayer_N").bind($.jPlayer.event.ended, restore_title);
    $("#jquery_jplayer_N").bind($.jPlayer.event.pause, restore_title);

    // Used when typing in the search area - it should use timeout since more
    // typing can follow immediately
    $('#search').keyup(search_with_timeout);
    $('#search').change(search_with_timeout);

    // Used when Enter is clicked - it should immediately send a request
    $('#search').keypress(search_immediately_on_enter);

    $('#album').change(function () {
        save_selected_album();
        filter_playlist();
    });

    $('#artist').change(function () {
        save_selected_artist();
        load_filters(found_songs, {selected_artist: $('#artist').val()});
        filter_playlist();
    });

    restore_last_saved_search();
});

_currently_playing = null;
_ajax_query = null;

function search_database (query, opts) {
    if (_ajax_query) {
        _ajax_query.abort();
    };

    opts = opts || {};
    if (opts.async == undefined) {
        opts.async = true;    
    };
    

    save_search_query(query);

    _last_search = query;

    _ajax_query = $.ajax({
        type: "GET",
        async: opts.async,
        url: encodeURI("/search/" + query),
        success: function (msg) {
            load_filters(msg);
            filter_playlist();
        }
    });
}

// Saves this search in the localStorage so that it can be used on the next
// refresh of the page.
function save_search_query (query) {
    if (!localStorage) {
        return;
    };
    localStorage.last_search = query;
}

// Saves the last selected album in the localStorage
function save_selected_album () {
    if (!localStorage) {
        return;
    };
    localStorage.last_album = $('#album').val();
}

// Saves the last selected artist in the localStorage
function save_selected_artist () {
    if (!localStorage) {
        return;
    };
    localStorage.last_artist = $('#artist').val();
}

// Restores the last search from the local storage. This should be used on startup.
// It was uses the laste selected artist and album and then filters the playlist
function restore_last_saved_search () {
    if (!localStorage) {
        return;
    };
    if (!localStorage.last_search || localStorage.last_search.length < 1) {
        return;
    };

    $('#search').val(localStorage.last_search);
    search_database(localStorage.last_search);
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

    var selected_index = null;
    var new_playlist = []
    for (var i = 0; i < songs.length; i++) {
        var song_url = "/file/"+songs[i].id;

        new_playlist.push({
            title: songs[i].title,
            artist: songs[i].artist,
            album: songs[i].album,
            mp3: song_url,
            free: true,
            number: songs[i].track,
            album_id: songs[i].album_id,
            media_id: songs[i].id
        });

        if (_currently_playing == songs[i].id) {
            selected_index = i;
        }
    };

    pagePlaylist.setPlaylist(new_playlist);

    if (selected_index) {
        pagePlaylist._highlight(selected_index);
        pagePlaylist.current = selected_index;
    };
}

found_songs = [];

function load_filters(songs, opts) {
    found_songs = songs;

    opts = opts || {};
    opts.selected_artist = opts.selected_artist || false;
    opts.selected_album = opts.selected_album || false;

    if (opts.selected_artist == false && localStorage.last_artist &&
                                            localStorage.last_artist.length >= 1) {
        opts.selected_artist = localStorage.last_artist;
    };

    if (opts.selected_album == false && localStorage.last_album &&
                                            localStorage.last_album.length >= 1) {
        opts.selected_album = localStorage.last_album;
    };

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
    
    var really_selected_artist = false;
    for (var i = 0; i < all_artists_list.length; i++) {
        var artist = all_artists_list[i];
        var option = $('<option></option>');
        option.html(artist).val(artist);
        if (opts.selected_artist && opts.selected_artist == artist) {
            really_selected_artist = artist;
            option.attr("selected", 1);
        };
        artist_elem.append(option);
    };
    
    var really_selected_album = false;
    for (var i = 0; i < all_albums_list.length; i++) {
        var album = all_albums_list[i];
        var option = $('<option></option>');
        option.html(album).val(album);
        if (opts.selected_album && opts.selected_album == album) {
            really_selected_album = album;
            option.attr("selected", 1);
        };
        album_elem.append(option);
    };

    if (localStorage.last_artist && localStorage.last_artist.length >= 1 && 
        really_selected_artist != localStorage.last_artist)
    {
        artist_elem.val('');
        delete localStorage.last_artist;
    };

    if (localStorage.last_album && localStorage.last_album.length >= 1 &&
        really_selected_album != localStorage.last_album)
    {
        album_elem.val('');
        delete localStorage.last_album;
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
