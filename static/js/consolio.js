angular.module("consolio", ['consAuth']).
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

function ConsolioCtrl($scope, $http) {
}


function LoginCtrl($scope, $http, $rootScope, consAuth) {
    $rootScope.$watch('loggedin', function() {
        $scope.auth = consAuth.get(); });

    $http.get("/api/me/").success(function(me) {
        $scope.me = me;
    });

    $scope.getAuthToken = function() {
        $http.get("/api/me/token/").
            success(function(res) {
                $scope.authtoken = res.token;
            });
    };

    $scope.logout = consAuth.logout;
    $scope.login = consAuth.login;
}
