var consAuth = angular.module('consAuth', []);

consAuth.factory('consAuth', function ($rootScope, $http, $location) {

    var auth = {
        loggedin: false,
        username: "",
        gravatar: "",
        authtoken: "",
        checked: false,
        redirected: false,
        userPrefs: {}
    };

    navigator.id.watch({
        onlogin: function (assertion) {
            //console.log("navigator.id.onLogin()");
            $http.post('/auth/login', "assertion=" + assertion + "&audience=" +
                encodeURIComponent(location.protocol + "//" + location.host),
                {headers: {"Content-Type": "application/x-www-form-urlencoded"}}).
                success(function (res) {
                    auth.loggedin = true;
                    auth.username = res.email;
                    auth.gravatar = res.emailmd5;
                    if (res.prefs) {
                        // some users have prefs: null
                        // in which case they should keep the defaults
                        auth.userPrefs = res.prefs;
                    }
                    auth.authtoken = "";
                    auth.checked = true;
                    $rootScope.loggedin = true;
//                    if (auth.loggedin && !auth.redirected && false) {
//                        auth.redirected = true;
//                        $location.url('/dashboard/');
//                    }
                }).
                error(function (res, err) {
                    bAlert("Error", "Couldn't log you in.", "error");
                });
        },
        onlogout: function () {
            //console.log("navigator.id.onLogout()");
            $http.post('/auth/logout').
                success(function (res) {
                    $rootScope.loggedin = false;
                    auth.loggedin = false;
                    auth.authtoken = "";
                    auth.username = "";
                    auth.gravatar = "";
                    auth.userPrefs = {};
                }).
                error(function (res) {
                    bAlert("Error", "Problem logging out.", "error");
                    // we failed to log out, do not pretend to have succeeded
                });
        },
        onready: function () {
            //console.log("navigator.id.onReady()");
            auth.checked = true;
        }
    });

    function fetchAuthToken() {
        console.log("fetch")
        $http.get("/api/me/token/").
            success(function (res) {
                auth.authtoken = res.token;
            });
    }

    function logout() {
        navigator.id.logout();
    }

    function login() {
        navigator.id.request();
    }

    function getAuth() {
        return auth;
    }

    $rootScope.auth = auth;

    return {
        login: login,
        logout: logout,
        get: getAuth
    };
});
