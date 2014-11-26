var plotPolygon = function(map, geom) {
    console.log("plotpolygon");
    for(var i = 0; i < geom.coordinates.length; i++) {
        for(var j = 0; j < geom.coordinates[i].length; j++) {
            var coords = [];

            for(var k = 0; k < geom.coordinates[i][j].length; k++) {
                coords.push(
                    new google.maps.LatLng(geom.coordinates[i][j][k][1],
                                           geom.coordinates[i][j][k][0]));

            }

            // Construct the polygon.
            polygon = new google.maps.Polygon({
                paths: coords,
                strokeColor: '#FF0000',
                strokeOpacity: 0.8,
                strokeWeight: 2,
                fillColor: '#FF0000',
                fillOpacity: 0.35
            });

            polygon.setMap(map);
        }
    }
}


angular.module('treelyApp', ['ngRoute', 'chieffancypants.loadingBar', 'ngAnimate'])
    .config(function($routeProvider) {
        $routeProvider
            .when('/', {
                redirectTo: "/trees"
            })
            .when('/trees', {
                controller:'TreesCtrl',
                templateUrl:'../trees.html'
            })
            .when('/trees/:treeId', {
                controller:'ShowTreeCtrl',
                templateUrl:'../show.html'
            })
            .when('/parks', {
                controller:'ParksCtrl',
                templateUrl:'../parks.html'
            })
            .otherwise({
                redirectTo:'/'
            });
    })
    .config(function(cfpLoadingBarProvider) {
        cfpLoadingBarProvider.includeSpinner = true;
    })
    .controller('TreesCtrl', function($scope, $http) {
        $scope.trees = [];

        $http.get('/trees').
            success(function(data, status, headers, config) {
                $scope.trees = data;
            }).
            error(function(data, status, headers, config) {
            });

    })
    .controller('ShowTreeCtrl', function($scope, $http, $routeParams) {
        $scope.tree = {}

        var mapOptions = {
            zoom: 4,
            center: new google.maps.LatLng(37.09024, -95.712891),
            mapTypeId: google.maps.MapTypeId.TERRAIN
        };

        $scope.map = new google.maps.Map(document.getElementById('map-container'), mapOptions);

        $http.get('/trees/' + $routeParams.treeId).
            success(function(data, status, headers, config) {
                $scope.tree = data;

                for(var i = 0; i < $scope.tree.geom.length; i++) {
                    console.log(i);
                    plotPolygon($scope.map, JSON.parse($scope.tree.geom[i]));
                }
            }).
            error(function(data, status, headers, config) {});
    })
    .controller('ParksCtrl', function($scope, $http, $routeParams, cfpLoadingBar) {
        $scope.parks = {}

        var mapOptions = {
            zoom: 4,
            center: new google.maps.LatLng(37.09024, -95.712891),
            mapTypeId: google.maps.MapTypeId.TERRAIN
        };

        $scope.map = new google.maps.Map(document.getElementById('map-container'), mapOptions);

        $http.get('/parks').
            success(function(data, status, headers, config) {
                cfpLoadingBar.start();
                $scope.parks = data;

                for(var i = 0; i < $scope.parks.geom.length; i++) {
                    cfpLoadingBar.inc();

                    console.log(i);
                    plotPolygon($scope.map, JSON.parse($scope.parks.geom[i]));
                }
                cfpLoadingBar.complete()

            }).
            error(function(data, status, headers, config) {});
    });
