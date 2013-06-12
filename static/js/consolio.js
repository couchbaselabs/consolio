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

function ConsolioCtrl($scope, $http, $rootScope, consAuth) {
    $rootScope.$watch('loggedin', function() {
        $scope.auth = consAuth.get(); });
}


function LoginCtrl($scope, $http, $rootScope, consAuth) {
    $rootScope.$watch('loggedin', function() {
        $scope.auth = consAuth.get(); });

    $scope.logout = consAuth.logout;
    $scope.login = consAuth.login;
}
