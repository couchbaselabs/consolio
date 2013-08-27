angular.module("consolio", ['consAuth', 'consAlert']).
    filter('calDate', function() {
        return function(dstr) {
            return moment(dstr).calendar();
        };
    }).
    config(['$routeProvider', '$locationProvider',
            function($routeProvider, $locationProvider) {
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

function ConsolioCtrl($scope, $http, $rootScope, consAuth, bAlert) {

}

function DashCtrl($scope, $http, $rootScope, consAuth, bAlert) {
    $rootScope.$watch('loggedin', function() {
        $scope.auth = consAuth.get();

        $http.get("/api/me/").success(function(me) { $scope.me = me; });

        $http.get("/api/sgw/").success(function(sgws) {
            console.log(sgws);
            $scope.syncgws = sgws;
        });

    });

    $http.get("/api/me/").success(function(me) { $scope.me = me; });

    $scope.databases = [];
    $scope.syncgws = [];

    $scope.availableDBs = function() {
        return _.filter($scope.databases, function(db) {
            return !_.any($scope.syncgws, function(sgw) {
                return db.name === sgw.extra.dbname;
            });
        });
    };

    $http.get("/api/sgw/").success(function(sgws) {
        console.log(sgws);
        $scope.syncgws = sgws;
    });

    $scope.wantnewsgw = false;

    $scope.newsgw = function() {
        var sgwname = $("#newsgwname").val();
        var guest = $("#newsgwguest").is(":checked");
        var dbname = $("#newsgwdb").val();
        var func = $("#newswsync").val();
        $http.post('/api/sgw/',
            'name=' + encodeURIComponent(sgwname) +
                '&guest=' + (guest?"true":"false") +
                '&syncfun=' + encodeURIComponent(func),
            {headers: {"Content-Type": "application/x-www-form-urlencoded"}})
            .error(function(data, code) {
                bAlert("Error " + code, "Couldn't create " + dbname +
                    ": " + data, "error");
            })
            .success(function(data) {
                $("#newsgwname").val("");
                $("#newsgwguest").val("");
                $("#newsgwdb").val("");
                var tmp = $scope.syncgws.slice(0);
                tmp.push(data);
                $scope.syncgws = tmp;
                $scope.wantnewsgw = false;
            });
    };
}


function DBCtrl($scope, $http, $routeParams, $rootScope, $location, consAuth, bAlert) {
    $scope.dbname = $routeParams.name;
    var dburl = "/api/database/" + $scope.dbname + "/";
    $http.get(dburl)
        .success(function(data) {
            $scope.db = data;
        })
        .error(function(data, code) {
            bAlert("Error " + code, "Couldn't get DB: " + data, "error");
        });

    $scope.delete = function() {
        $http.delete(dburl)
            .success(function(data) {
                $location.path("/index/");
            })
            .error(function(data, code) {
                bAlert("Error " + code, "Couldn't delete DB: " + data, "error");
            });
    };
}

function SGWCtrl($scope, $http, $routeParams, $rootScope, $location, consAuth, bAlert) {

    $scope.sgwname = $routeParams.name;
    var sgwurl = "/api/sgw/" + $scope.sgwname + "/";
    $http.get(sgwurl)
        .success(function(data) {
            $scope.sgw = data;
        })
        .error(function(data, code) {
            bAlert("Error " + code, "Couldn't get SGW: " + data, "error");
        });

    $scope.delete = function() {
        $http.delete(sgwurl)
            .success(function(data) {
                $location.path("/index/");
            })
            .error(function(data, code) {
                bAlert("Error " + code, "Couldn't delete SGW: " + data, "error");
            });
    };

    $scope.authtoken = "";

    $scope.getAuthToken = function() {
        $http.get("/api/me/token/").
            success(function(res) {
                // This isn't exactly right, but it's pretty close
                $scope.authuser = encodeURIComponent($scope.sgw.owner);
                $scope.authtoken = res.token;
            });
    };
}

function AdminCtrl($scope, $http, $rootScope, $location, bAlert) {
    $http.get("/api/me/")
        .success(function(data) {
            $scope.me = data;
        })
        .error(function(data, error) {
            $location.path("/index/");
        });

    $scope.webhooks = [];
    $http.get("/api/webhook/")
        .success(function(data) {
            $scope.webhooks = data;
        })
        .error(function(data, error) {
            bAlert("Error " + code, "Couldn't get webhooks: " + data, "error");
        });

    $scope.add = function() {
        var n = $("#newhookname").val();
        var u = $("#newhookurl").val();
        $http.post("/api/webhook/",
                   'name=' + encodeURIComponent(n) +
                   '&url=' + encodeURIComponent(u),
                   {headers: {"Content-Type": "application/x-www-form-urlencoded"}})
            .success(function(data) {
                $("#newhookname").val("");
                $("#newhookurl").val("");
                $scope.webhooks.push(data);
            })
            .error(function(data, code) {
                bAlert("Error " + code, "Couldn't create webhook: " + data, "error");
            });
    };

    $scope.delete = function(n) {
        $http.delete("/api/webhook/" + encodeURIComponent(n) + "/")
            .success(function(data) {
                $scope.webhooks = _.filter($scope.webhooks, function(e) {
                    return e.name !== n;
                });
            })
            .error(function(data, code) {
                bAlert("Error " + code, "Couldn't delete webhook: " + data, "error");
            });
    };
}

function LoginCtrl($scope, $http, $rootScope, $location, consAuth) {
    $rootScope.$watch('loggedin', function() {
        $scope.auth = consAuth.get();
    });

    $http.get("/api/me/").success(function(me) { $scope.me = me; });

    $scope.logout = consAuth.logout;
    $scope.login = consAuth.login;
    $scope.authtoken = "";

    $scope.getAuthToken = function() {
        $http.get("/api/me/token/").
            success(function(res) {
                $scope.authtoken = res.token;
            });
    };
}
