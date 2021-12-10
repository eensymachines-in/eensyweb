(function () {
    angular.module("eensyweb").controller("splashCtrl", function ($http, $scope) {
        $http.get("/data/eensy.json").then(function (response) {
            console.info("We have eensyjson data ..");
            $scope.data = response.data.d;
        }, function (error) {
            // this is when we have an error fetching the data from the server
            console.error("Failed to get eensymachines data from server");
            $rootScope.err = error;
        })
    }).controller("productListCtrl", function ($scope,srvHttp) {
        $scope.selc_prod = null;
        // gets the bunch of products and allows to select one product at a time 
        srvHttp.download_data("products.json").then(function(data){
            $scope.products = data;
            if ($scope.products.length >0) {
                $scope.selc_prod = $scope.products[0];
            }
        }, function(e){

        });
    })
})()