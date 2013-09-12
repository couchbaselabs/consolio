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
        //console.log($rootScope.switch.visible);
    }
}

function DashCtrl($scope, $http, $rootScope, consAuth, bAlert, $location) {

    $scope.logout = consAuth.logout;
    $scope.login = consAuth.login;

    $scope.switch = $rootScope.switch;

    //console.log("%c DashCtrl>>>>>>", 'background: #222; color: #bada55');
    //console.log({ loggedin: $rootScope.loggedin, syncgws: $scope.syncgws, init: $scope.init, sgws_size: $scope.syncgws_size});
    $scope.init = $scope.init || false;

    if (!$scope.init) {
        $scope.databases = [];
        $scope.syncgws = [];
        $scope.syncgws_size = 0;
        $scope.init = true;
    }

    $rootScope.$watch('loggedin', function () {
        if ($rootScope.loggedin == true) {

            //console.log("%c DashCtrl - $watch loggedin", 'background: #222; color: #bada55');
            //console.log({ loggedin: $rootScope.loggedin, syncgws: $scope.syncgws, init: $scope.init, sgws_size: $scope.syncgws_size});

            $http.get("/api/me/").success(function (me) {
                $scope.me = me;
            });

            $http.get("/api/sgw/").success(function (sgws) {

                $scope.syncgws = sgws;
                console.log(sgws);

                if ($scope.syncgws.length > 0) {
                    // For the cancel edit feature on sync functions, we edit a copy (so a copy needs to be created)
                    angular.forEach($scope.syncgws, function (i) {
                        //console.log(i);
                        i.extra.sync_copy = i.extra.sync;
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
        //console.log("%c Watch syncgws[]", 'background: #222; color: #55daba');
        //console.log({ loggedin: $rootScope.loggedin, syncgws: $scope.syncgws, init: $scope.init, sgws_size: $scope.syncgws_size});
        var val = value || null;
        if (val) {
            //console.log("values..." + $scope.syncgws_size + "/" + $scope.syncgws.length.toString());

            if ($scope.syncgws.length > 0 && $scope.syncgws_size != $scope.syncgws.length) {
                //console.log("%c BOOM!", 'color: #ff0000');
                $scope.syncgws_size = $scope.syncgws.length
                //console.log({ x: true, loggedin: $rootScope.loggedin, syncgws: $scope.syncgws, init: $scope.init, sgws_size: $scope.syncgws_size});
                //setTimeout($scope.setButtons, 2000);
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

    // Not used for now
    $scope.setButtons = function () {

        $("button.zerocopy-button").each(function (i, e) {
            console.log($(this));
            var clip = new ZeroClipboard($(this), { moviePath: '/static/swf/ZeroClipboard.swf' });
            var txt = $(this).attr("data-clipboard-text");
            console.log(txt);

            clip.on('complete', function (client, args) {
                alert("Copied text to clipboard: " + args.text);
            });

            clip.on('dataRequested', function (client, args) {
                clip.setText(txt);
                console.log(txt);
            });
//            $(this).click(function (event) {
//                console.log("click");
//            });
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

        // Events
//        editor.on("beforeChange", function () {
//            console.log("editor BeforeChange");
//        });
//
//        editor.on("change", function () {
//            console.log("editor OnChange");
//        });
    }

    $scope.cmLoaded = function (editor) {

//          var autoformat = function (editor) {
//            var totalLines = editor.lineCount();
//            var totalChars = editor.getTextArea().value.length;
//            editor.autoFormatRange({line: 0, ch: 0}, {line: totalLines, ch: totalChars});
//            editor.getDoc().setCursor({line: 0, ch: 0});
//        }
//
//        autoformat(editor);
    }

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
        //

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

    clip.on('mousedown', function (client) {
        // alert("mouse down");
    });

    clip.on('mouseup', function (client) {
        $(this).addClass("btn-success");
        $(this).removeClass("btn-info");
        $("#notify-copied").show().addClass("in");
//        item.tooltip({
//            title: 'copied!'
//        });
        // alert("mouse up");
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

    $scope.webhooks = [];
    $http.get("/api/webhook/")
        .success(function (data) {
            $scope.webhooks = data;
        })
        .error(function (data, error) {
            bAlert("Error " + code, "Couldn't get webhooks: " + data, "error");
        });

    $scope.add = function () {
        var n = $("#newhookname").val();
        var u = $("#newhookurl").val();
        $http.post("/api/webhook/",
            'name=' + encodeURIComponent(n) +
                '&url=' + encodeURIComponent(u),
            {headers: {"Content-Type": "application/x-www-form-urlencoded"}})
            .success(function (data) {
                $("#newhookname").val("");
                $("#newhookurl").val("");
                $scope.webhooks.push(data);
            })
            .error(function (data, code) {
                bAlert("Error " + code, "Couldn't create webhook: " + data, "error");
            });
    };

    $scope.delete = function (n) {
        $http.delete("/api/webhook/" + encodeURIComponent(n) + "/")
            .success(function (data) {
                $scope.webhooks = _.filter($scope.webhooks, function (e) {
                    return e.name !== n;
                });
            })
            .error(function (data, code) {
                bAlert("Error " + code, "Couldn't delete webhook: " + data, "error");
            });
    };
}

function LoginCtrl($scope, $http, $rootScope, consAuth) {
    //console.log("LoginCtrl");

    $rootScope.$watch('loggedin', function () {

        if ($rootScope.loggedin) {
            //console.log("LoginCtrl - $watch loggedin");
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
