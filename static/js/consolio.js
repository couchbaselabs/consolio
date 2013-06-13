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
                    otherwise({redirectTo: '/index/'});
                $locationProvider.html5Mode(true);
                $locationProvider.hashPrefix('!');
            }]);

function ConsolioCtrl($scope, $http, $rootScope, consAuth, bAlert) {
    $rootScope.$watch('loggedin', function() {
        $scope.auth = consAuth.get(); });

    $scope.databases = [];
    $http.get("/api/my/databases/").success(function(databases) {
        $scope.databases = databases;
    });

    $scope.newbucket = "";

    $scope.newdb = function() {
        var dbname = $("#newbucketname").val();
        console.log("Adding a new thing");
        $http.post('/api/database/new/',
                   'name=' + encodeURIComponent(dbname),
                   {headers: {"Content-Type": "application/x-www-form-urlencoded"}})
            .error(function(data, code) {
                bAlert("Error " + code, "Couldn't create " + dbname +
                       ": " + data, "error");
            })
            .success(function(data) {
                console.log("Worked!");
                $scope.newbucket="";
                $scope.databases.push(data);
            });
    };
}


function LoginCtrl($scope, $http, $rootScope, consAuth) {
    $rootScope.$watch('loggedin', function() {
        $scope.auth = consAuth.get(); });

    $scope.logout = consAuth.logout;
    $scope.login = consAuth.login;
}
