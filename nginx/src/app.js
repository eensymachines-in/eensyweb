// this has the definition of the new anfular js app
(function(){
    angular.module("eensyweb",["ngRoute"]).config(function($routeProvider,$interpolateProvider,$provide){
        // with GO Lang frameworks this can help to have angular a distinct space 
        $interpolateProvider.startSymbol("{[")
        $interpolateProvider.endSymbol("]}")
       
        $routeProvider
        .when("/", {
            templateUrl:"/views/splash.html"
        })
        .when("/products", {
            templateUrl:"/views/products-list.html"
        })
        .when("/about", {
            templateUrl:"/views/about.html"
        })
        $provide.provider("emailPattern", function(){
            this.$get = function(){
                // [\w] is the same as [A-Za-z0-9_-]
                // 3 groups , id, provider , domain also a '.' in between separated by @
                // we are enforcing a valid email id 
                // email id can have .,_,- in it and nothing more 
                return /^[\w-._]+@[\w]+\.[a-z]+$/
            }
        })
        $provide.provider("passwdPattern", function(){
            this.$get = function(){
                // here for the password the special characters that are not allowed are being singled out and denied.
                // apart form this all the characters will be allowed
                // password also has a restriction on the number of characters in there
                return /^[\w-!@#%&?_]{8,16}$/
            }
        })
    })
})()