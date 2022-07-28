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
                controller: function($scope, $http) {
                    var reset = function() {
                        $scope.areaOptions = [];
                        $scope.blockArea = null; // view model for 
                    };
                    reset();
                    // This will verify the pincode with 3rd party and check for delivery suitability
                    $scope.check_pincode = function() {
                        reset();
                        $http({
                            method: "GET",
                            url: "https://api.postalpincode.in/pincode/" + $scope.pin,
                            headers: {
                                'Content-Type': "application/json",
                            },
                        }).then(function(response) {
                            if (response.data[0].PostOffice.length > 0) {
                                var data = response.data[0];
                                $scope.state = data.PostOffice[0].State;
                                data.PostOffice.forEach(function(po) {
                                    // User options for the POs in the zone 
                                    $scope.areaOptions.push({
                                        title: po.Block + ", " + po.Name,
                                        select: function(bl) {
                                            // When user selects a PO
                                            $scope.blockArea = {
                                                title: bl
                                            }
                                        }
                                    });
                                });
                            }
                        }, function(response) {
                            // TODO: handle the error gracefully 
                            console.error("failed to verify the pin code location for delivery")
                        })
                    }
                    console.log("now inside the pinLocation directive")
                }
            }
        })
})()