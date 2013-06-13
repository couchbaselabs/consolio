angular.module("consolio", ['consAuth', 'consAlert']).
    filter('calDate', function() {
        return function(dstr) {
            return moment(dstr).calendar();
        };
    }).
    config(['$routeProvider', '$locationProvider',
            function($routeProvider, $locationProvider) {
                $routeProvider.
                    when('/index/', {templateUrl: '/static/partials/index.html',
                                     controller: 'ConsolioCtrl'}).
                    when('/db/:name/', {templateUrl: '/static/partials/db.html',
                                        controller: 'DBCtrl'}).
                    when('/admin/', {templateUrl: '/static/partials/admin.html',
                                     controller: 'AdminCtrl'}).
                    otherwise({redirectTo: '/index/'});
                $locationProvider.html5Mode(true);
                $locationProvider.hashPrefix('!');
            }]);

function ConsolioCtrl($scope, $http, $rootScope, consAuth, bAlert) {
    $rootScope.$watch('loggedin', function() { $scope.auth = consAuth.get(); });

    $http.get("/api/me/").success(function(me) { $scope.me = me; });

    $scope.databases = [];
    $http.get("/api/database/").success(function(databases) {
        $scope.databases = databases;
    });

    $scope.newbucket = "";

    $scope.newdb = function() {
        var dbname = $("#newbucketname").val();
        console.log("Adding a new thing");
        $http.post('/api/database/',
                   'name=' + encodeURIComponent(dbname),
                   {headers: {"Content-Type": "application/x-www-form-urlencoded"}})
            .error(function(data, code) {
                bAlert("Error " + code, "Couldn't create " + dbname +
                       ": " + data, "error");
            })
            .success(function(data) {
                console.log("Worked!");
                $("#newbucketname").val("");
                $scope.newbucket="";
                $scope.databases.push(data);
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
                $scope.webhooks.push([n, u]);
            })
            .error(function(data, code) {
                bAlert("Error " + code, "Couldn't create webhook: " + data, "error");
            });
    };

    $scope.delete = function(n) {
        $http.delete("/api/webhook/" + encodeURIComponent(n) + "/")
            .success(function(data) {
                $scope.webhooks = _.filter($scope.webhooks, function(e) {
                    return e[0] !== n;
                });
            })
            .error(function(data, code) {
                bAlert("Error " + code, "Couldn't delete webhook: " + data, "error");
            });
    };
}

function LoginCtrl($scope, $http, $rootScope, consAuth) {
    $rootScope.$watch('loggedin', function() { $scope.auth = consAuth.get(); });
    $http.get("/api/me/").success(function(me) { $scope.me = me; });

    $scope.logout = consAuth.logout;
    $scope.login = consAuth.login;
}
