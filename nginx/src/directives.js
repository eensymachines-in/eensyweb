(function() {
    angular.module("eensyweb").directive("waButton", function() {
            // this can induce a whatsapp redirection button anywhere the button isplaced
            return {
                restrict: "EA",
                replace: true,
                scope: {},
                templateUrl: "/templates/wa-button.html",
            }
        }).directive("socialStrip", function() {
            // this can induce a whatsapp redirection button anywhere the button isplaced
            return {
                restrict: "EA",
                replace: true,
                scope: {
                    item: "<"
                },
                templateUrl: "/templates/social-strip.html",
            }
        }).directive("companyIntro", function() {
            return {
                restrict: "EA",
                replace: true,
                scope: {},
                templateUrl: "/templates/company-intro.html",
            }
        })
        .directive("aboutCard", function() {
            return {
                restrict: "EA",
                replace: true,
                scope: {
                    data: "@"
                },
                templateUrl: "/templates/about-card.html",
                controller: function($scope, srvHttp) {
                    srvHttp.download_data($scope.data).then(function(data) {
                        $scope.about = data;
                    }, function() {

                    })
                }
            }
        })
        .directive("pinLocation", function() {
            /* While placing an order the deliverable location is an user input
            - User enters a pin code and checks it 
            - This pin code is verified/detailed with api.postalpincode.in
            - state, pin, address are attached to the order in notes 
            - for pin code the user needs to select a PO in the zone
            - for a zone there could be multiple POs and not all are deliverable locations
            - after selecting the PO the complete address is also sought from the user 
            */
            return {
                restrict: "EA",
                replace: true,
                scope: {
                    pin: "=",
                    state: "=",
                    addr: "="
                },
                templateUrl: "/templates/pin-location.html",
                controller: function($scope, $http, $sce) {
                    $scope.checkButton = $sce.trustAsHtml("Check");
                    var reset = function() {
                        console.log("resetting variables..")
                        $scope.areaOptions = [];
                        $scope.blockArea = null; // view model for 
                        $scope.err = null; // this will highlight the pin code input
                    };
                    reset();
                    // This will verify the pincode with 3rd party and check for delivery suitability
                    $scope.check_pincode = function() {
                        $scope.checkButton = $sce.trustAsHtml('<i class="fas fa-circle-notch load-animate fa-spin"></i>');
                        reset();
                        $http({
                            method: "GET",
                            url: "https://api.postalpincode.in/pincode/" + $scope.pin,
                            headers: {
                                'Content-Type': "application/json",
                            },
                        }).then(function(response) {
                            $scope.checkButton = $sce.trustAsHtml("Check");
                            var data = response.data[0];
                            if (data.PostOffice != null) {
                                $scope.state = data.PostOffice[0].State;
                                data.PostOffice.forEach(function(po) {
                                    $scope.areaOptions.push({
                                        title: po.Block + ", " + po.Name,
                                        select: function(bl) {
                                            $scope.blockArea = {
                                                title: bl
                                            }; // zone options can be selected 
                                        }
                                    });
                                }); //options for zone selection
                            } else {
                                // When the pin is not found this api does not emit 404
                                // instead sends back the error in the same format 
                                // but with field PostOffice == null 
                                $scope.err = {
                                    msg: "failed to get pin code details"
                                }
                            }
                        }, function(response) {
                            // TODO: handle the error gracefully 
                            $scope.checkButton = $sce.trustAsHtml("Check");
                            console.error("failed to get pin code details ..");
                            $scope.err = {
                                msg: "failed to get pin code details"
                            }

                        })
                    }
                }
            }
        })
})()