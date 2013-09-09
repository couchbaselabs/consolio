angular.module('angularCodeMirror', [])
    .directive('cbMirror', ['$timeout', function ($timeout) {
        return {
            restrict: 'A',
            require: 'ngModel',
            link: function(scope, el, attrs, ngModel) {
                scope.setupMirror = function() {
                    scope.codeMirror = CodeMirror.fromTextArea(el[0], {
                        lineWrapping: true,
                        lineNumbers: true,
                        mode: 'javascript',
                        theme: 'night',
                        smartIndent: true,
                        onChange: scope.reParseInput,
                        onFocus: scope.formatInput
                    });
                    if(scope.live) {
                        scope.codeMirror.on("change", function(cm, change) {
                            var val = cm.getValue();
                            if(val !== ngModel.$viewValue) {
                                ngModel.$setViewValue(val);
                                scope.$apply();
                            }
                        });
                    }
                    ngModel.$render = function() {
                        var viewval = ngModel.$viewValue;
                        if(!viewval) {
                            viewval = "";
                            ngModel.$setViewValue(viewfal);
                        }
                        scope.codeMirror.setValue(viewval);
                    };
                };
                scope.tearDownMirror = function() {
                    scope.codeMirror.toTextArea();
                    scope.codeMirror = null;
                };
                scope.$watch(function() {
                    if(scope.editing && !scope.codeMirror) {
                        scope.setupMirror();
                    }
                    if(!scope.editing && scope.codeMirror) {
                        scope.tearDownMirror();
                    }
                });
            }
        };
    }])
    .directive('cbEditor', function () {
        var editortpl = '<div ng-class="{edithide: !editing}"><textarea ng-model="source" '+
            'cb-mirror></textarea></div>';
        return {
            restrict: 'E',
            scope: {
                source: '=',
                editing: '=',
                editfn: '=',
                live: '@'
            },
            compile: function(tElem, tAttrs) {
                tElem.prepend(editortpl);
                return {
                    post: function(scope, el, attrs) {
                        scope.editfn = function() {
                            scope.editing = true;
                        };
                        scope.save = function() {
                            scope.editing = false;
                            scope.source = scope.codeMirror.getValue();
                        };
                    }
                };
            }
        };
    });