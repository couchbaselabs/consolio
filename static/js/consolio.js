angular.module("consolio", ['ui.codemirror', 'consAuth', 'consAlert']).
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

function DashCtrl($scope, $http, $rootScope, consAuth, bAlert) {

    console.log("%c DashCtrl>>>>>>", 'background: #222; color: #bada55');
    console.log({ loggedin: $rootScope.loggedin, syncgws: $scope.syncgws, init: $scope.init, sgws_size: $scope.syncgws_size});
    $scope.init = $scope.init || false;

    if (!$scope.init) {
        $scope.databases = [];
        $scope.syncgws = [];
        $scope.syncgws_size = 0;
        $scope.init = true;
    }

    $rootScope.$watch('loggedin', function () {
        if ($rootScope.loggedin == true) {

            console.log("%c DashCtrl - $watch loggedin", 'background: #222; color: #bada55');
            console.log({ loggedin: $rootScope.loggedin, syncgws: $scope.syncgws, init: $scope.init, sgws_size: $scope.syncgws_size});

            $http.get("/api/me/").success(function (me) {
                $scope.me = me;
            });

            $http.get("/api/sgw/").success(function (sgws) {

                $scope.syncgws = sgws;
                //setTimeout($scope.setButtons(), 2000);
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

    $scope.wantnewsgw = false;

    $scope.newsgw = function () {
        var sgwname = $("#newsgwname").val();
        var guest = $("#newsgwguest").is(":checked");
        var dbname = $("#newsgwdb").val();
        var func = $("#newswsync").val();
        $http.post('/api/sgw/',
            'name=' + encodeURIComponent(sgwname) +
                '&guest=' + (guest ? "true" : "false") +
                '&syncfun=' + encodeURIComponent(func),
            {headers: {"Content-Type": "application/x-www-form-urlencoded"}})
            .error(function (data, code) {
                bAlert("Error " + code, "Couldn't create " + dbname +
                    ": " + data, "error");
            })
            .success(function (data) {
                $("#newsgwname").val("");
                $("#newsgwguest").val("");
                $("#newsgwdb").val("");
                var tmp = $scope.syncgws.slice(0);
                tmp.push(data);
                $scope.syncgws = tmp;
                $scope.wantnewsgw = false;
            });
    };

    $scope.delete = function (i) {
        //swgurl = $scope.syncgws[]
        console.log("DELETE");
        console.log("index = " + i);
//        $http.delete(sgwurl)
//            .success(function(data) {
//                $location.path("/dashboard/");
//            })
//            .error(function(data, code) {
//                bAlert("Error " + code, "Couldn't delete SGW: " + data, "error");
//            });
    };

    $scope.$watch("syncgws", function (value) {
        console.log("%c Watch syncgws[]", 'background: #222; color: #55daba');
        console.log({ loggedin: $rootScope.loggedin, syncgws: $scope.syncgws, init: $scope.init, sgws_size: $scope.syncgws_size});
        var val = value || null;
        if (val) {
            //console.log("values..." + $scope.syncgws_size + "/" + $scope.syncgws.length.toString());

            if ($scope.syncgws.length > 0 && $scope.syncgws_size != $scope.syncgws.length) {
                console.log("%c BOOM!", 'color: #ff0000');
                $scope.syncgws_size = $scope.syncgws.length
                console.log({ x: true, loggedin: $rootScope.loggedin, syncgws: $scope.syncgws, init: $scope.init, sgws_size: $scope.syncgws_size});
                setTimeout($scope.setButtons, 2000);
            }
        }
    });

    $scope.clickSyncUrlButton = function(i) {
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

    $scope.clickAdminSyncUrlButton = function(i) {
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

    $scope.setButtons = function() {

//        $("button.zerocopy-button").each(function (i, e) {
//            console.log($(this));
//            var clip = new ZeroClipboard($(this), { moviePath: '/static/swf/ZeroClipboard.swf' });
//            var txt = $(this).attr("data-clipboard-text");
//            console.log(txt);
//
//            clip.on( 'complete', function ( client, args ) {
//                alert("Copied text to clipboard: " + args.text );
//            });
//
//            clip.on( 'dataRequested', function ( client, args ) {
//                clip.setText(txt);
//                console.log(txt);
//            });
////            $(this).click(function (event) {
////                console.log("click");
////            });
//        });
    }

    $scope.cleanInput = function(editor){
        console.log("clean");
        var autoformat = function(editor) {
            var totalLines = editor.lineCount();
            var totalChars = editor.getTextArea().value.length;
            editor.autoFormatRange({line:0, ch:0}, {line:totalLines, ch:totalChars});
            editor.getDoc().setCursor({line:0, ch:0});
        }

        autoformat(editor);
    }

    $scope.reParseInput = function(editor){
        console.log("reparse");
        var doc = editor.getDoc();

        // Options
        doc.markClean()

        // Events
        editor.on("beforeChange", function(){
            console.log("before_change");
        });

        editor.on("change", function(){
            console.log("change");
        });
    }

    $scope.codemirrorLoaded = function(editor){

        console.log("codemirrorLoaded");

        var autoformat = function(editor) {
            var totalLines = editor.lineCount();
            var totalChars = editor.getTextArea().value.length;
            editor.autoFormatRange({line:0, ch:0}, {line:totalLines, ch:totalChars});
            editor.getDoc().setCursor({line:0, ch:0});
        }

        autoformat(editor);
    }

    $scope.editorOptions = {
        lineWrapping : true,
        lineNumbers: true,
        mode: 'javascript',
        theme: 'night',
        smartIndent: true,
        onChange: $scope.reParseInput,
        onFocus: $scope.cleanInput,
        onLoad: $scope.codemirrorLoaded
    }

    $scope.saveSyncFunction = function(i){
        console.log("saved - " + i.toString())
        console.log($scope.syncgws[i].extra.sync)
    }
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
    console.log("LoginCtrl");

    $rootScope.$watch('loggedin', function () {

        if ($rootScope.loggedin) {
            console.log("LoginCtrl - $watch loggedin");
            $scope.auth = consAuth.get();
        }
    });

    $http.get("/api/me/").success(function (me) {
        $scope.me = me;
    });

    $scope.logout = consAuth.logout;
    $scope.login = consAuth.login;
    $scope.authtoken = "";

    $scope.getAuthToken = function () {
        $http.get("/api/me/token/").
            success(function (res) {
                $scope.authtoken = res.token;
            });
    };
}
