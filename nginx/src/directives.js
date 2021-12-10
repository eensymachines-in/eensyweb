(function(){
    angular.module("eensyweb").directive("waButton", function(){
        // this can induce a whatsapp redirection button anywhere the button isplaced
        return {
            restrict :"EA", 
            replace: true, 
            scope: {},
            templateUrl:"/templates/wa-button.html",
        }
    }).directive("socialStrip", function(){
        // this can induce a whatsapp redirection button anywhere the button isplaced
        return {
            restrict :"EA", 
            replace: true, 
            scope: {
                item:"<"
            },
            templateUrl:"/templates/social-strip.html",
        }
    }).directive("companyIntro", function(){
        return {
            restrict :"EA", 
            replace: true, 
            scope: {},
            templateUrl:"/templates/company-intro.html",
        }
    })
    .directive("aboutCard", function(){
        return {
            restrict: "EA",
            replace: true,
            scope: {
                data:"@"
            }, 
            templateUrl: "/templates/about-card.html",
            controller: function($scope, srvHttp){
                srvHttp.download_data($scope.data).then(function(data){
                    $scope.about = data;
                }, function(){

                })
            }
        }
    })
})()