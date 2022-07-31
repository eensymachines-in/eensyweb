(function() {
    angular.module("eensyweb").service("srvHttp", function($http, $q, $rootScope) {
        // A common function that helps to read the error details from http response
        // this has specific affinity to how the error is packed from the GO backend
        var err_message = function(response) {
                // err_message : breaks down the error response as required for modals / warning
                var m = "Server unreachable, or responded invalid. Kindly wait for admins to fix this";
                var l = "";
                if (response.data !== null && response.data !== undefined) {
                    if (response.data.message !== null && response.data.message !== undefined) {
                        m = response.data.message.split("\n")[0];
                        l = response.data.message.split("\n")[1];
                    }
                }
                return {
                    "status": response.status,
                    "statusText": response.statusText,
                    "message": m,
                    "logid": l
                }
            }
            // this helps to make a http call to a server json file to download data as a dump
            // make sure you pass the correct file name
        this.download_data = function(fileName) {
            var defered = $q.defer();
            $http.get("/data/" + fileName).then(function(response) {
                console.log("Downloaded data: " + fileName);
                console.log(response.data.d)
                defered.resolve(response.data.d);
            }, function(response) {
                console.error("Failed to download data: " + fileName);
                $rootScope.err = err_message(response);
                defered.reject($rootScope.err)
            })
            return defered.promise;
        }
    }).service("srvPurchase", function() {
        // setting the purchase field is very important
        // this is the distinguishing factor between if order page has been arrived at from valid predecessor
        // we do not want to let the user directly type in order url for obvious reasons
        // when purchase is null we know there is a un-natural way of arriving at the order page
        this.purchase = null;
        this.set_purchase = function(product, rate) {
            this.purchase = { product: product, rate: rate };
        }
        this.unset_purchase = function() {
            this.purchase = null;
        }
        return this
    })
})()