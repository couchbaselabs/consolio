var app = angular.module("consolio", ['ui.codemirror', 'consAuth', 'consAlert', 'angularCodeMirror', 'ngRoute']).
    filter('calDate',function () {
        return function (dstr) {
            return moment(dstr).calendar();
        };
    }).
    config(['$routeProvider', '$locationProvider',
        function ($routeProvider, $locationProvider) {
            $routeProvider.
                when('/index/', {templateUrl: '/static/partials/index.html'}).
                when('/terms_of_service/', {templateUrl: '/static/partials/terms_of_service.html'}).
                when('/acceptable_use/', {templateUrl: '/static/partials/acceptable_use.html'}).
                when('/privacy_policy/', {templateUrl: '/static/partials/privacy_policy.html'}).
                when('/faq/', {templateUrl: '/static/partials/faq.html'}).
                when('/dashboard/', {templateUrl: '/static/partials/dashboard.html',
                    controller: 'DashCtrl'}).
                when('/db/:name/', {templateUrl: '/static/partials/db.html',
                    controller: 'DBCtrl'}).
                when('/sgw/:name/', {templateUrl: '/static/partials/sgw.html',
                    controller: 'SGWCtrl'}).
                when('/admin/', {templateUrl: '/static/partials/admin.html',
                    controller: 'AdminCtrl'}).
                otherwise({redirectTo: '/index/'});
            $locationProvider.html5Mode(true);
            $locationProvider.hashPrefix('!');
        }]);

app.directive('confirmDelete', function () {
    return {
        priority: 1,
        terminal: true,
        link: function (scope, element, attr) {
            var msg = attr.confirmDelete || "Are you sure?";
            var clickAction = attr.ngClick;
            element.bind('click',function () {
                if ( window.confirm(msg) ) {
                    scope.$eval(clickAction)
                }
            });
        }
    };
});

function SwitchCtrl($scope, $rootScope) {

    $rootScope.switch = { visible: false }

    $scope.switch_toggle = function () {
        if ($rootScope.switch.visible) {
            $rootScope.switch.visible = false;
        }
        else {
            $rootScope.switch.visible = true;
        }
    }
}

function DashCtrl($scope, $http, $rootScope, consAuth, bAlert, $location) {

    $scope.logout = consAuth.logout;
    $scope.login = consAuth.login;
    $scope.prettySize = prettySize;

    $scope.switch = $rootScope.switch;
    $scope.init = $scope.init || false;

    if (!$scope.init) {
        $scope.databases = [];
        $scope.syncgws = [];
        $scope.syncgws_size = 0;
        $scope.init = true;
    }

    $rootScope.$watch('loggedin', function () {
        if ($rootScope.loggedin == true) {

            $http.get("/api/me/").success(function (me) {
                $scope.me = me;
            });

            $http.get("/api/sgw/").success(function (sgws) {

                $scope.syncgws = sgws;

                if ($scope.syncgws.length > 0) {

                    $http.get("/api/database/").success(function(dbs) {
                       $scope.databases = dbs;
                        // For the cancel edit feature on sync
                        // functions, we edit a copy (so a copy needs
                        // to be created)
                        angular.forEach($scope.syncgws, function (i) {
                            i.stats = _.detect($scope.databases, function(db) {
                                if (i.extra.dbname == db.name)
                                    return db.stats;
                            });
                            i.extra.sync_copy = i.extra.sync;
                        });
                    });

                }

            });
        }
    });

    $scope.availableDBs = function () {
        return _.filter($scope.databases, function (db) {
            return !_.any($scope.syncgws, function (sgw) {
                return db.name === sgw.extra.dbname;
            });
        });
    };

    $scope.modal_sgw_name = "sgw_name";
    $scope.modal_api_url = "api_url";
    $scope.modal_admin_url = "admin_url";

    $scope.setModalContent = function (i) {
        if ($scope.syncgws[i]) {

            var sgw = $scope.syncgws[i]
            $scope.modal_sgw_name = sgw.name;

            if (sgw.extra.url) {
                $scope.modal_api_url = sgw.url
            }
            else {
                $scope.modal_api_url = "http://sync.couchbasecloud.com/" + sgw.name + "/";
            }

            if ($scope.authuser == null || $scope.authtoken == null) {
                $http.get("/api/me/token/").
                    success(function (res) {
                        // This isn't exactly right, but it's pretty close
                        $scope.authuser = encodeURIComponent(sgw.owner);
                        $scope.authtoken = res.token;


                        $scope.modal_admin_url = "http://" + $scope.authuser + ":" + $scope.authtoken + "@syncadm.couchbasecloud.com:8083/" + sgw.name + "/"
                    });
            }
            else {
                $scope.modal_admin_url = "http://" + $scope.authuser + ":" + $scope.authtoken + "@syncadm.couchbasecloud.com:8083/" + sgw.name + "/"
            }


        }
    }

    $scope.wantNewSGW = false;
    $scope.newSGW_name = "";
    $scope.newSGW_guest = false;
    $scope.newSGW_sync = "function(doc) { \n\tchannel(doc.channels);\n}";
    $scope.newSGW_error = false;

    $scope.newSGW_name_changed = function () {
        $scope.newSGW_name = $scope.newSGW_name.replace(" ", "_");

        if ($scope.newSGW_name.length < 5) {
            $scope.newSGW_error = true;
        }
        else {
            $scope.newSGW_error = false;
        }
    }

    $scope.createNewSGW = function () {
        console.log("**** Creating sync gateway...");
        console.log($scope.newSGW_name);
        console.log($scope.newSGW_guest);
        console.log($scope.newSGW_sync);
        console.log("****");
        if ($scope.newSGW_error) {
            return false;
        }
        $http.post('/api/sgw/',
            'name=' + encodeURIComponent($scope.newSGW_name) +
                '&guest=' + ($scope.newSGW_guest ? "true" : "false") +
                '&syncfun=' + encodeURIComponent($scope.newSGW_sync),
            {headers: {"Content-Type": "application/x-www-form-urlencoded"}})
            .error(function (data, code) {
                bAlert("Error " + code, "Couldn't create " + dbname +
                    ": " + data, "danger");
            })
            .success(function (data) {
                console.log("SUCCESS");
                console.log(data);
                $scope.wantNewSGW = false;
                $scope.newSGW_name = "";
                $scope.newSGW_guest = false;
                $scope.newSGW_sync = "function(doc) { \n\tchannel(doc.channels);\n}";
                var tmp = $scope.syncgws.slice(0);
                data.extra.sync_copy = data.extra.sync;
                tmp.push(data);
                $scope.syncgws = tmp;
                $scope.wantnewsgw = false;
            });
    };

    $scope.delete = function (i) {
        var sgwurl = "/api/sgw/" + $scope.syncgws[i].name + "/";
        console.log("DELETE");
        console.log("index = " + i);
        console.log(sgwurl);
        $http.delete(sgwurl)
            .success(function(data) {
                $scope.syncgws = _.filter($scope.syncgws, function(e){
                    return e.name != $scope.syncgws[i].name;
                });
            })
            .error(function(data, code) {
                bAlert("Error " + code, "Couldn't delete SGW: " + data, "error");
            });
    };

    $scope.$watch("syncgws", function (value) {
        var val = value || null;
        if (val) {
            if ($scope.syncgws.length > 0 && $scope.syncgws_size != $scope.syncgws.length) {
                $scope.syncgws_size = $scope.syncgws.length
            }
        }
    });

    $scope.clickSyncUrlButton = function (i) {
        console.log($scope.syncgws[i]);
        var sgw = $scope.syncgws[i];
        var btn = $("sgw-" + i.toString());
        var clip = new ZeroClipboard(btn, { moviePath: '/static/swf/ZeroClipboard.swf' });
        clip.on('complete', function (client, args) {
            alert("Copied text to clipboard: " + args.text);
        });

        clip.on('dataRequested', function (client, args) {
            clip.setText("http://sync.couchbasecloud.com/" + sgw.name);
        });
    }

    $scope.clickAdminSyncUrlButton = function (i) {
        console.log($scope.syncgws[i]);
        var sgw = $scope.syncgws[i];
        var btn = $("sgwa-" + i.toString());
        var clip = new ZeroClipboard(btn, { moviePath: '/static/swf/ZeroClipboard.swf' });
        clip.on('complete', function (client, args) {
            alert("Copied text to clipboard: " + args.text);
        });

        clip.on('dataRequested', function (client, args) {
            clip.setText(txt);
            console.log(txt);
        });

    }

    $scope.formatInput = function (editor) {

        console.log("%c ******* formatInput()", 'color: #0000ff');

        var autoformat = function (editor) {
            var totalLines = editor.lineCount();
            var totalChars = editor.getTextArea().value.length;
            editor.autoFormatRange({line: 0, ch: 0}, {line: totalLines, ch: totalChars});
            editor.getDoc().setCursor({line: 0, ch: 0});
        }

        autoformat(editor);

        $scope.editor = editor;
    }

    $scope.reParseInput = function (editor) {
        console.log("%c ******* reParseInput()", 'color: #0000ff');
        var doc = editor.getDoc();

        // Options
        doc.markClean()

    }

    $scope.cmLoaded = function (editor) {}

    $scope.editorOptions = {
        lineWrapping: true,
        lineNumbers: true,
        mode: 'javascript',
        theme: 'night',
        smartIndent: true,
        onChange: $scope.reParseInput,
        onFocus: $scope.formatInput,
        onLoad: function (editor) {
            console.log("%c ******* CodeMirror Loaded", 'color: #0000ff');
        }
    }

    $scope.saveSyncFunction = function (i) {
        $scope.syncgws[i].extra.sync = $scope.syncgws[i].extra.sync_copy
        // Here we actually save the sync function through http post
        alert('Coming Soon!');
    }

    $scope.cancelSaveSyncFunction = function (i) {
        $scope.syncgws[i].extra.sync_copy = $scope.syncgws[i].extra.sync
    }


    ZeroClipboard.setDefaults({ moviePath: '/static/swf/ZeroClipboard.swf' });

    var clip1 = makeZero($("#copy-api-url"));
    var clip2 = makeZero($("#copy-admin-url"));

}

function makeZero(item) {

    var clip = new ZeroClipboard(item, { moviePath: '/static/swf/ZeroClipboard.swf' });

    clip.on('load', function (client) {
    });

    clip.on('complete', function (client, args) {
    });

    clip.on('mouseover', function (client) {
    });

    clip.on('mouseout', function (client) {
        var x = $(this);
        setTimeout(function() {
            x.addClass("btn-info");
            x.removeClass("btn-success");
            $("#notify-copied").fadeOut('slow', function(){
                $("#notify-copied").removeClass("in");
            })
        }, 2000);
    });

    clip.on('mousedown', function (client) {});

    clip.on('mouseup', function (client) {
        $(this).addClass("btn-success");
        $(this).removeClass("btn-info");
        $("#notify-copied").show().addClass("in");
    });

    return clip;
}


function DBCtrl($scope, $http, $routeParams, $location, bAlert) {
    console.log("DBCtrl");

    $scope.dbname = $routeParams.name;
    var dburl = "/api/database/" + $scope.dbname + "/";
    $http.get(dburl)
        .success(function (data) {
            $scope.db = data;
        })
        .error(function (data, code) {
            bAlert("Error " + code, "Couldn't get DB: " + data, "error");
        });

    $scope.delete = function () {
        $http.delete(dburl)
            .success(function (data) {
                $location.path("/index/");
            })
            .error(function (data, code) {
                bAlert("Error " + code, "Couldn't delete DB: " + data, "error");
            });
    };
}

function SGWCtrl($scope, $http, $routeParams, $location, bAlert) {
    console.log("SGWCtrl");

    $scope.sgwname = $routeParams.name;
    var sgwurl = "/api/sgw/" + $scope.sgwname + "/";
    $http.get(sgwurl)
        .success(function (data) {
            $scope.sgw = data;
        })
        .error(function (data, code) {
            bAlert("Error " + code, "Couldn't get SGW: " + data, "error");
        });

    $scope.delete = function () {
        $http.delete(sgwurl)
            .success(function (data) {
                $location.path("/index/");
            })
            .error(function (data, code) {
                bAlert("Error " + code, "Couldn't delete SGW: " + data, "error");
            });
    };

    $scope.authtoken = "";

    $scope.getAuthToken = function () {
        $http.get("/api/me/token/").
            success(function (res) {
                // This isn't exactly right, but it's pretty close
                $scope.authuser = encodeURIComponent($scope.sgw.owner);
                $scope.authtoken = res.token;
            });
    };
}

function AdminCtrl($scope, $http, $rootScope, $location, bAlert) {
    console.log("AdminCtrl");

    $http.get("/api/me/")
        .success(function (data) {
            $scope.me = data;
        })
        .error(function (data, error) {
            $location.path("/index/");
        });
}

function LoginCtrl($scope, $http, $rootScope, consAuth) {
    $rootScope.$watch('loggedin', function () {
        if ($rootScope.loggedin) {
            $scope.auth = consAuth.get();
        }
    });

    $http.get("/api/me/").success(function (me) {
        $scope.me = me;
    });

    $scope.login = function(){
        $('#signupModal').modal('hide')
        consAuth.login();
    }

    $scope.logout = consAuth.logout;
    $scope.authtoken = "";

    $scope.getAuthToken = function () {
        $http.get("/api/me/token/").
            success(function (res) {
                $scope.authtoken = res.token;
            });
    };
}

function prettySize(s) {
    if (s < 10) {
        return s + "B";
    }
    var e = parseInt(Math.floor(Math.log(s) / Math.log(1024)));
    var sizes = ["B", "KB", "MB", "GB", "TB", "PB", "EB"];
    var suffix = sizes[parseInt(e)];
    var val = s / Math.pow(1024, Math.floor(e));
    x = val.toFixed(2) + " " + suffix;
    return x;
}
