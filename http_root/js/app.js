/*
    Euterpe javascript
*/

const serviceName = 'Euterpe';

$(document).ready(function(){
    switch (window.location.pathname) {
        case "/":
            playerPageInit();
            break;
        case "/login/":
            loginPageInit();
            break;
        case "/add_device/":
            addDevicePageInit();
            break;
    }
});

function playerPageInit() {
    // jPlayer defines the flac "codec" (mime) as "audio/x-flac". But for some
    // browsers, notably Brave, this is nonsense. They only know of "audio/flac".
    // So before initializing the jPlayer we check which type of browser is in use
    // and define the flac "codec" to "audio/flac" if the browser does not understand
    // "auido/x-flac". The latter is already out-of-use anyway.
    try {
        var auidoEl = document.createElement('audio');
        const canPlay = auidoEl.canPlayType("audio/x-flac");
        if (canPlay == '') {
            $.jPlayer.prototype.format.flac.codec = 'audio/flac';
        }
    } catch(err) {
        // Do nothing. This browser probably does not support the HTML <audio> tag.
        // Hope it does support Flash somehow then.
    }

    // The default setPlaylist method was calling _init which did
    // jPlayerPlaylist.select for the first track. That resulted in jplayer stopping
    // the played song. Now we make it never select. We also set current = undef so
    // that it will not be equal to the song currently plaing since this is a new
    // playlist after all.
    jPlayerPlaylist.prototype.setPlaylist = function(playlist) {
        this._initPlaylist(playlist);
        this._refresh(true);
    };

    // The default implementation of _updateControls was making a show/hide
    // on a class which is present in every entry in the playlist. This takes
    // too much time if the playlist has many items.
    jPlayerPlaylist.prototype._updateControls = function() {
        var shuffled = this.shuffled;

        if (this.realShuffled !== undefined) {
            shuffled = this.realShuffled;
        }

        if(shuffled) {
            $(this.cssSelector.shuffleOff).show();
            $(this.cssSelector.shuffle).hide();
        } else {
            $(this.cssSelector.shuffleOff).hide();
            $(this.cssSelector.shuffle).show();
        }

        this.shuffled = shuffled;
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
            listItem += "<span class='hidden-xs " + options.freeGroupClass;
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

        var timeString = 'n/a';
        if (media.duration && media.duration > 0) {
            timeString = intToDuration(media.duration);
        }
        listItem += " <span class='jp-duration hidden-xs " +
            options.freeGroupClass + "'>" + timeString + "</span>";

        if (media.album) {
            listItem += " <span class='hidden-xs " + options.freeGroupClass + "'>" +
                    "<a href='/v1/album/"+media.album_id+"' title='download album' "+
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

    // The override is needed because of a bug with loading different playlists
    // while media is still runnig. The previous click handler did not change the
    // media when the index of the clicked media matched the index of the currently
    // played media in the previous playlist.
    jPlayerPlaylist.prototype._createItemHandlers = function() {
        var self = this;
        var options = this.options.playlistOptions;
        // Create live handlers for the playlist items
        $(this.cssSelector.playlist).off("click", "a." + options.itemClass);
        $(this.cssSelector.playlist).on("click", "a." + options.itemClass, function() {
            var index = $(this).parent().parent().index();
            self.play(index);
            $(this).blur();
            return false;
        });

        // Create live handlers that disable free media links to force access via right click
        $(this.cssSelector.playlist).off("click", "a." + options.freeItemClass);
        $(this.cssSelector.playlist).on("click", "a." + options.freeItemClass, function() {
            $(this).parent().parent().find("." + options.itemClass).click();
            $(this).blur();
            return false;
        });

        // Create live handlers for the remove controls
        $(this.cssSelector.playlist).off("click", "a." + options.removeItemClass);
        $(this.cssSelector.playlist).on("click", "a." + options.removeItemClass, function() {
            var index = $(this).parent().parent().index();
            self.remove(index);
            $(this).blur();
            return false;
        });
    };

    // The default jPlayerPlaylist shuffle implementation is quite stupid and does
    // a LOAD of work for no good reason. It rearranges the DOM elements! Which
    // obviously takes a lot of time when there are many entries in the playlist.
    // Yet again, this plug-in is not meant to work with a particularly large
    // collections.
    //
    // The changed function does not touch the DOM. It just stores a flag that
    // playing must be shuffled. The actual shuffling is done by the "next"
    // function.
    jPlayerPlaylist.prototype.shuffle = function(shuffled, playNow) {
        if(shuffled === undefined) {
            shuffled = !this.shuffled;
        }

        this.shuffled = shuffled;

        // realShuffled is used for the actual value. The jPlayerPlaylist is
        // monkeying with this.shuffled in expected places, toggling it to false.
        // So we ignore it totally while trying to keep it in sync with this.realShuffled
        // when possible.
        this.realShuffled = shuffled;

        var shuffleOff = $(this.cssSelector.shuffleOff);
        var shuffle = $(this.cssSelector.shuffle);

        if (shuffled) {
            shuffleOff.show();
            shuffle.hide();
        } else {
            shuffleOff.hide();
            shuffle.show();
        }

        if (localStorage) {
            localStorage.shuffleOn = shuffled ? "true" : "false";
        }
    }

    // next is overwritten in order to support more efficient shuffling. See the comment
    // above shuffled.
    jPlayerPlaylist.prototype.next = function() {
        if (this.shuffled && this.playlist.length > 1) {
            var index = this.current;
            while (index == this.current) {
                index = Math.round(Math.random() * this.playlist.length - 1);
            }

            if (index >= 0) {
                this.play(index);
            }
            return;
        }

        var index = (this.current + 1 < this.playlist.length) ? this.current + 1 : 0;

        if(this.loop) {
            this.play(index);
        } else if(index > 0) {
            // The index will be zero if it just looped round
            this.play(index);
        }
    }

    // previous is overwritten in order to support more efficient shuffling. See the
    // comment above shuffled.
    jPlayerPlaylist.prototype.previous = function() {
        var index = (this.current - 1 >= 0) ? this.current - 1 : this.playlist.length - 1;

        if(this.loop && this.options.playlistOptions.loopOnPrevious || index < this.playlist.length - 1) {
            this.play(index);
        }
    }

    var loopState = false;
    if (localStorage) {
        loopState = (localStorage.repeatOn === "true") ? true : false;
    }

    var cssSelector = {
        jPlayer: "#jquery_jplayer_bootstrap",
        cssSelectorAncestor: "#jp_container_bootstrap"
    };

    var playlist = [];

    var options = {
        swfPath: "/js",
        supplied: "mp3, oga, m4a, wav, flac, webma, fla",
        preload: "none",
        playlistOptions: {
            autoPlay: false,
            displayTime: 0,
            addTime: 0,
            removeTime: 0,
            shuffleTime: 0
        },
        loop: loopState
    };

    pagePlaylist = new jPlayerPlaylist(cssSelector, playlist, options);

    _search_timeout = null;
    _last_search = null;

    var search_with_timeout = function () {
        var search_query = $('#search').val();

        if (_search_timeout) {
            clearTimeout(_search_timeout);
        }

        if (search_query == _last_search) {
           return;
        }

        if (search_query.length < 1) {
            return;
        }

        _search_timeout = setTimeout(function() {
            search_database(search_query);
        }, 500);
    };

    var search_immediately_on_enter = function (e) {
        var search_query = $('#search').val();
        var code = e.keyCode || e.which;
        if(code != 13 && e.type != 'click') {
           return;
        }

        if (_search_timeout) {
            clearTimeout(_search_timeout);
        }

        search_database(search_query);
    };

    // Show song title when one is playing
    $(cssSelector.jPlayer).bind($.jPlayer.event.play, function(event) {
        var media = event.jPlayer.status.media;

        if (!media) {
            return;
        }

        document.title = media.title + ' by ' + media.artist + ' | ' + serviceName;

        var artwork_el = $('#artwork');
        var artwork_url = '/v1/album/' + escape(media.album_id) + '/artwork';
        var current_artwork_url = artwork_el.css("background-image");

        if (artwork_url !== current_artwork_url) {
            artwork_el.css("background-image", 'url(' + artwork_url + ')');
        }

        if (history.pushState) {
            var uri = new URI(window.location);
            var query_data = uri.search(true);

            if (query_data.tr === undefined || query_data.tr != media.media_id) {
                history.pushState(
                    null,
                    null,
                    URI(window.location).setSearch('tr', media.media_id)
                );
            }
        }
    });

    // Restores the normal title when nothing is played
    var restore_title = function(event) {
        document.title = serviceName;
    };

    $(cssSelector.jPlayer).bind($.jPlayer.event.ended, restore_title);
    $(cssSelector.jPlayer).bind($.jPlayer.event.pause, restore_title);

    // Used when typing in the search area - it should use timeout since more
    // typing can follow immediately
    $('#search').keyup(search_with_timeout);
    $('#search').change(search_with_timeout);

    // Used when Enter is clicked - it should immediately send a request
    $('#search').keypress(search_immediately_on_enter);
    $('.search-form-button').click(search_immediately_on_enter);

    $('#album').change(function () {
        save_selected_album();
        filter_playlist();
    });

    $('#artist').change(function () {
        save_selected_artist();
        load_filters(found_songs, {selected_artist: $('#artist').val()});
        filter_playlist();
    });

    $('.load-all-btn').click(function(e) {
        search_immediately_on_enter(e);
    })

    $('#artwork').popover({
        container: '#artwork',
        placement: 'left',
        trigger: 'hover',
        viewport: '.container',
        html: true,
        content: function() {
            var bgImage = $('#artwork').css("background-image");
            if (bgImage == "none") {
                return;
            }

            var img = $('<img>');
            img.attr('src', bgImage.replace(/url\("(.+)"\)/, '$1'));
            img.attr('style', 'max-width: 50vw; max-height: 50vw;');
            return img;
        }
    });

    // This event handler stores the state of the "repeat" button.
    $(cssSelector.jPlayer).bind($.jPlayer.event.repeat, function(event) {
        if (!localStorage) {
            return;
        }

        localStorage.repeatOn = event.jPlayer.options.loop ? "true" : "false";
    });

    $(document).ajaxStop(function(){
        var btn = $('.search-form-button > .glyphicon-refresh');
        btn.removeClass('glyphicon-refresh anim-revolving').addClass('glyphicon-search');

        $('.load-all-btn > .glyphicon').
            removeClass('glyphicon-refresh anim-revolving').
            addClass('glyphicon-circle-arrow-down');
    });

    $(document).ajaxStart(function(){
        var btn = $('.search-form-button > .glyphicon-search');
        btn.removeClass('glyphicon-search').addClass('glyphicon-refresh anim-revolving');

        $('.load-all-btn > .glyphicon').
            removeClass('glyphicon-circle-arrow-down').
            addClass('glyphicon-refresh anim-revolving');
    });

    restore_last_saved_search();
    restore_shuffle_state(pagePlaylist);
}

function loginPageInit() {
    if (window.location.search.includes("wrongCreds=1")) {
        $('.wrong-creds').show();
    }
}

function addDevicePageInit() {
    var serverAddress = window.location.protocol + "//" + window.location.host;
    var img = $("<img>");
    img.attr("src", "/new_qr_token/?address=" + encodeURIComponent(serverAddress));
    img.attr("alt", "New Token QR Barcode");

    var barcode = $('.barcode');
    barcode.empty();
    barcode.append(img)
}

_ajax_query = null;

function search_database (query, opts) {
    if (_ajax_query) {
        _ajax_query.abort();
    }

    opts = opts || {};
    if (opts.async === undefined) {
        opts.async = true;
    }

    save_search_query(query);

    _last_search = query;

    _ajax_query = $.ajax({
        type: "GET",
        async: opts.async,
        url: "/v1/search?q=" + encodeURIComponent(query),
        success: function (msg) {
            load_filters(msg);
            filter_playlist();
        }
    });
}

// Saves this search in the localStorage so that it can be used on the next
// refresh of the page.
function save_search_query (query) {
    if (localStorage) {
        localStorage.last_search = query;
    }
    if (history.pushState) {
        history.pushState(null, null, URI(window.location).setSearch('q', query));
    }
}

// Saves the last selected album in the localStorage
function save_selected_album () {
    var album = $('#album').val();
    if (localStorage) {
        localStorage.last_album = album;
    }
     if (history.pushState) {
         history.pushState(null, null, URI(window.location).setSearch('al', album));
     }
}

// Saves the last selected artist in the localStorage
function save_selected_artist () {
    var artist = $('#artist').val();
    if (!localStorage) {
        localStorage.last_artist = artist;
    }
    if (history.pushState) {
        history.pushState(null, null, URI(window.location).setSearch('at', artist));
    }
}

// Restores the last search from the local storage. This should be used on startup.
// It was uses the laste selected artist and album and then filters the playlist
function restore_last_saved_search () {
    var last_search = null;

    var uri = new URI(window.location);
    var query_data = uri.search(true);

    if (query_data.q) {
        last_search = query_data.q;
    }

    if (last_search === null && localStorage) {
        if (localStorage.last_search && localStorage.last_search.length >= 1) {
            last_search = localStorage.last_search;
        }
    }

    if (last_search === null) {
        return;
    }

    $('#search').val(last_search);
    search_database(last_search);
}

// restore_shuffle_state restores the state of the shuffle button to what it was last
// set in the local storage.
function restore_shuffle_state(playlistObj) {
    if (!localStorage) {
        return;
    }

    if (localStorage.shuffleOn === undefined) {
        return;
    }

    if (localStorage.shuffleOn === "true") {
        playlistObj.shuffle(true, false);
    } else {
        playlistObj.shuffle(false, false);
    }
}

function load_playlist (songs) {

    songs.sort(function (a, b) {
        if (a.album == b.album) {
            return a.track < b.track ? -1 : 1;
        } else {
            return a.album < b.album ? -1 : 1;
        }
    });

    var selected_index = null;
    var currently_playing = null;

    var uri = new URI(window.location);
    var query_data = uri.search(true);
    if (query_data.tr !== undefined && query_data.tr !== "") {
        currently_playing = parseInt(query_data.tr, 10);
    }

    $('.empty-playlist').hide();

    var new_playlist = [];
    var songs_length = songs.length;
    for (var i = 0; i < songs_length; i++) {
        var song_url = "/v1/file/"+songs[i].id;
        var song = {
            title: songs[i].title,
            artist: songs[i].artist,
            album: songs[i].album,
            free: true,
            number: songs[i].track,
            album_id: songs[i].album_id,
            media_id: songs[i].id,
            // convert nanoseconds to seconds
            duration: songs[i].duration ? songs[i].duration / 1e3 : 0
        };

        // For certain formats the jPlayer does not use their file extension name so
        // we check for them explicitly here.
        var format = songs[i].format;
        if (format === 'ogg') {
            format = 'oga';
        }
        song[format] = song_url;

        new_playlist.push(song);

        if (currently_playing !== null && currently_playing == songs[i].id) {
            selected_index = i;
        }
    }

    if (songs_length == 0) {
        $('.no-songs-found').show();
    } else {
        $('.no-songs-found').hide();
    }

    pagePlaylist.setPlaylist(new_playlist);

    if (selected_index === null && songs.length > 0) {
        selected_index = 0;
    }

    if (selected_index !== null) {
        pagePlaylist._highlight(selected_index);
        pagePlaylist.current = selected_index;

        if (currently_playing === null) {
            $(pagePlaylist.cssSelector.jPlayer).jPlayer(
                "setMedia",
                pagePlaylist.playlist[selected_index]
            );
        }
    }
}

found_songs = [];

function load_filters(songs, opts) {
    found_songs = songs;

    opts = opts || {};
    opts.selected_artist = opts.selected_artist || false;
    opts.selected_album = opts.selected_album || false;

    var uri = new URI(window.location);
    var query_data = uri.search(true);

    if (opts.selected_artist === false && query_data.at !== undefined) {
        opts.selected_artist = query_data.at;
    }

    if (opts.selected_artist === false && localStorage.last_artist &&
                                            localStorage.last_artist.length >= 1) {
        opts.selected_artist = localStorage.last_artist;
    }

    if (opts.selected_album === false && query_data.al !== undefined) {
        opts.selected_album = query_data.al;
    }

    if (opts.selected_album === false && localStorage.last_album &&
                                            localStorage.last_album.length >= 1) {
        opts.selected_album = localStorage.last_album;
    }

    var artist_elem = $('#artist');
    var album_elem = $('#album');

    var all_artists = {}, all_artists_list = [];
    var all_albums = {}, all_albums_list = [];

    for (var i = 0; i < songs.length; i++) {
        if (all_artists[songs[i].artist] === undefined) {
            all_artists[songs[i].artist] = true;
            all_artists_list.push({
                name: songs[i].artist,
                id: songs[i].artist_id
            });
        }

        // There might be more than one album with the same name. So we are forced
        // to use album IDs instead of names. This is unfortunate since the URLs
        // would be worse!
        if (all_albums[songs[i].album_id] === undefined) {
            all_albums[songs[i].album_id] = true;
            all_albums_list.push({
                name: songs[i].album,
                id: songs[i].album_id,
                artist_id: songs[i].artist_id
            });
        }
    }

    artist_elem.empty();
    album_elem.empty();

    var all_artists_opt = $('<option></option>').html("All").val("");
    if (!opts.selected_artist) {
        all_artists_opt.attr("selected", 1);
    }
    artist_elem.append(all_artists_opt);

    var all_albums_opt = $('<option></option>').html("All").val("");
    if (!opts.selected_album) {
        all_albums_opt.attr("selected", 1);
    }
    album_elem.append(all_albums_opt);

    all_artists_list.sort(alpha_sort);
    all_albums_list.sort(alpha_sort);

    var option = null;

    var really_selected_artist = false;
    for (var i = 0; i < all_artists_list.length; i++) {
        var artist = all_artists_list[i].name;
        var option = $('<option></option>');
        option.html(artist).val(artist);
        if (opts.selected_artist && opts.selected_artist == artist) {
            really_selected_artist = all_artists_list[i];
            option.attr("selected", 1);
        }
        artist_elem.append(option);
    }

    var really_selected_album = false;
    for (var i = 0; i < all_albums_list.length; i++) {
        var album = all_albums_list[i].name;
        var album_id = all_albums_list[i].id;
        var artist_id = all_albums_list[i].artist_id;
        var option = $('<option></option>');
        option.html(album).val(album_id);

        if (opts.selected_album && opts.selected_album === album_id) {
            really_selected_album = album_id;
            option.attr("selected", 1);
        }

        if (really_selected_artist && really_selected_artist.id !== artist_id) {
            continue;
        }

        album_elem.append(option);
    }

    if (localStorage.last_artist && localStorage.last_artist.length >= 1 &&
        really_selected_artist.name != localStorage.last_artist)
    {
        artist_elem.val('');
        delete localStorage.last_artist;
    }

    if (localStorage.last_album && localStorage.last_album.length >= 1 &&
        really_selected_album != localStorage.last_album)
    {
        album_elem.val('');
        delete localStorage.last_album;
    }
}

function filter_playlist () {

    var artist_filter = function (song) {return true;};
    var album_filter = function (song) {return true;};

    var selected_artist = $('#artist :selected').val();
    if (selected_artist) {
        artist_filter = function (song) {
            if (selected_artist == song.artist) {
                return true;
            }
            return false;
        };
    }

    var selected_album = $('#album :selected').val();
    if (selected_album) {
        album_filter = function (song) {
            if (selected_album == song.album_id) {
                return true;
            }
            return false;
        };
    }

    var to_load = [];
    var found_songs_length = found_songs.length;
    for (var i = 0; i < found_songs_length; i++) {
        if (!artist_filter(found_songs[i])) {
            continue;
        }

        if (!album_filter(found_songs[i])) {
            continue;
        }

        to_load.push(found_songs[i]);
    }

    load_playlist(to_load);
}

function alpha_sort (a, b) {
    if (a.name) {
        return alpha_sort(a.name, b.name);
    }
    if (!a.toLowerCase) {
        return b;
    }
    if (!b.toLowerCase) {
        return a;
    }
    a = a.toLowerCase();
    b = b.toLowerCase();
    if (a == b) {
        return 0;
    }
    return (a < b) ? -1 : 1;
}

function intToDuration(dur) {
    if (dur === 0) {
        return "0";
    }

    const durs = [
        {n: 60, s: "s"},
        {n: 60, s: "m"},
        {n: 24, s: "h"},
        {n: null, s: "d"},
    ];

    var str = "";
    for (var i = 0; i < durs.length; i++) {
        const t = durs[i];
        if (t.n === null) {
            str = dur + t.s + str;
            break;
        }

        r = dur % t.n;
        if (r !== 0) {
            str = r + t.s + str;
        }
        dur = Math.floor(dur / t.n)

        if (dur === 0) {
            break;
        }
    }

    return str;
}
