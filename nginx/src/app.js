// this has the definition of the new anfular js app
(function() {
    angular.module("eensyweb", ["ngRoute"]).config(function($routeProvider, $interpolateProvider, $provide, $locationProvider) {
        // with GO Lang frameworks this can help to have angular a distinct space 
        $interpolateProvider.startSymbol("{[")
        $interpolateProvider.endSymbol("]}")
        $locationProvider.html5Mode({
            enabled: true,
            requireBase: true
        });

        $routeProvider
            .when("/", {
                templateUrl: "/views/splash.html"
            })
            .when("/products", {
                templateUrl: "/views/products-list.html"
            })
            .when("/blogs", {
                templateUrl: "/views/blogs-list.html"
            })
            .when("/blogs/:id", {
                templateUrl: "/views/blogs-read.html"
            })
            .when("/about", {
                templateUrl: "/views/about.html"
            })
            .when("/products/:id", {
                templateUrl: "/views/product-detail.html"
            })
            .when("/order", {
                // This route has no mapping on theproxy/server
                // reason being : this route is accessible only from within the application 
                // directly typing /order in the url bar will not / shuld not result in a page 
                // /order has a predecessor settings for purchase - product name and rate of the product
                templateUrl: "/views/order.html"
            })
            .when("/order-done", {
                templateUrl: "/views/order-done.html"
            })
        $provide.provider("emailPattern", function() {
            this.$get = function() {
                // [\w] is the same as [A-Za-z0-9_-]
                // 3 groups , id, provider , domain also a '.' in between separated by @
                // we are enforcing a valid email id 
                // email id can have .,_,- in it and nothing more 
                return /^[\w-._]+@[\w]+\.[a-z]+$/
            }
        })
        $provide.provider("passwdPattern", function() {
            this.$get = function() {
                // here for the password the special characters that are not allowed are being singled out and denied.
                // apart form this all the characters will be allowed
                // password also has a restriction on the number of characters in there
                return /^[\w-!@#%&?_]{8,16}$/
            }
        })
        $provide.provider("rzpKey", function() {
            this.$get = function() {
                return {
                    // Rzp public key for test mode
                    test: "rzp_test_Z4AumzgwmBpgQv",
                    live: ""
                }
            }
        })
    }).filter("paiseAsRupee", function() {
        return function(paise) {
            // https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/Intl/NumberFormat
            // this formatting lets us put the number in a more currency readable format
            return new Intl.NumberFormat('en-IN', { maximumSignificantDigits: 3 }).format(paise / 100);
        }
    })
})()