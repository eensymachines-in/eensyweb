(function() {
    angular.module("eensyweb").controller("splashCtrl", function($http, $scope) {
        $http.get("/data/eensy.json").then(function(response) {
            console.info("We have eensyjson data ..");
            $scope.data = response.data.d;
        }, function(error) {
            // this is when we have an error fetching the data from the server
            console.error("Failed to get eensymachines data from server");
            $rootScope.err = error;
        })
    }).controller("productListCtrl", function($scope, srvHttp) {
        $scope.selc_prod = null;
        // gets the bunch of products and allows to select one product at a time 
        srvHttp.download_data("products.json").then(function(data) {
            $scope.products = data;
            if ($scope.products.length > 0) {
                $scope.selc_prod = $scope.products[0];
            }
        }, function(e) {

        });
    }).controller("prodDetailCtrl", function($scope, srvHttp, $routeParams) {
        srvHttp.download_data("products.json").then(function(data) {
            // Here we need to take details from one single product
            $scope.products = data;
            var filtered = $scope.products.filter(x => x.id == $routeParams.id);
            console.log(filtered);
            $scope.selcProd = filtered[0];
        }, function(e) {

        });
    }).controller("blogsListCtrl", function($scope, srvHttp) {
        $scope.activeIndex = 0;
        srvHttp.download_data("blogs.json").then(function(data) {
            $scope.blogs = data;
            // calculate pagination 
            var perPage = 5; // number of blogs per page
            // calculating total number of blog pages 
            var totalPages = Math.floor($scope.blogs.length / perPage);
            totalPages = $scope.blogs.length % perPage > 0 ? totalPages + 1 : totalPages;
            // incase of more than the number divisible by 3 one extra page is also needed
            $scope.pages = [];
            var i, j, temporary
                // splitting the pages in chunks of perPage
            for (i = 0, j = $scope.blogs.length; i < j; i += perPage) {
                temporary = $scope.blogs.slice(i, i + perPage);
                $scope.pages.push(temporary)
            }
            if ($scope.pages.length > 0) {
                $scope.activePage = $scope.pages[0];
            }
            $scope.goto_page = function(pageNo) {
                $scope.activePage = $scope.pages[pageNo];
                $scope.activeIndex = pageNo;
                console.log("Active index :" + $scope.activeIndex)
            }
        }, function(e) {

        });
    }).controller("blogReadCtrl", function($scope, srvHttp, $routeParams, $sce, $http) {
        var file = $routeParams.id + '.html';
        // reading the full length of the blog

        srvHttp.download_data("blogs.json").then(function(data) {
            console.log(data);
            console.log($routeParams.id);
            var filtered = data.filter(x => x.id == $routeParams.id);
            console.log(filtered);
            if (filtered.length > 0) {
                // selecting the blog from the route param id
                $scope.blog = filtered[0];
                console.log($scope.blog);
                console.log(file)
                $http.get("/templates/" + file).then(function(response) {
                    console.log(response.data);
                    $scope.rawHtml = response.data;
                }, function(response) {
                    console.error("Error getting the blog content");
                    console.error(response.error)
                });
                $scope.renderHtml = function() {
                    return $sce.trustAsHtml($scope.rawHtml);
                }
            }
        }, function(e) {

        });

    }).controller("testPayCtrl", function($scope, $http, rzpKey, srvPurchase) {
        srvPurchase.set_purchase("autolumin", 29500)
        $scope.orderFailed = null;
        $scope.units = 1;
        $scope.pinCode;
        $scope.location = {
            pin: 0,
            state: "",
            block: "",
            // this can let user select the broader area of delivery
            areasList: {
                disp: false,
                options: [],
                select: "", // the selected area
            },
            addressTxtBox: {
                addr: "",
                disp: false
            }
        };

        // somehow we have to see how we can send in rateOfUnit to this controller from another one
        // or is it possible to pass from routes?
        var rateOfUnit = srvPurchase.purchase.rate * 100;
        console.log("purchase for: " + srvPurchase.purchase.product);
        console.log("rate of the product :" + rateOfUnit);
        // create an order by hitting the url on eensy api  .. 
        // once we have he order we can then send the order id to razorpay for checkout.js

        $scope.invalidity = {
            email: false,
            name: false,
            contact: false,
            check: function() {
                // this shall check the validtiy of the payment fields entered by the user
                this.email = $scope.order.prefill.email == "";
                this.name = $scope.order.prefill.name == "";
                this.contact = $scope.order.prefill.contact == "";
                return this.email || this.name || this.contact;
            }
        }
        $scope.order = {
            // this has to come a provider
            key: rzpKey.test,
            // Amount is in currency subunits. Default currency is INR. Hence, 50000 refers to 50000 paise
            amount: rateOfUnit * $scope.units,
            currency: "INR",
            // this is generated first in 
            order_id: "",
            name: "Eensymachines",
            description: "Home automation purchase ",
            image: "/images/eensybright.png",
            handler: function(response) {
                // this is out of angularjs scope 
                // once we have the confirmation of payment - success/failure we go ahead to post the same to eensy server
                $scope.$apply(function() {
                    // for the url to be hit we need service support to form the base url
                    $http({
                        method: "POST",
                        url: "http://localhost/payments",
                        headers: {
                            'Content-Type': "application/json",
                        },
                        data: JSON.stringify({
                            "razorpay_payment_id": response.razorpay_payment_id,
                            "razorpay_order_id": response.razorpay_order_id,
                            "razorpay_signature": response.razorpay_signature
                        })

                    }).then(function(response) {
                        console.log("payment confirmed..")
                    }, function(data) {
                        console.log("Payment could be done, not confirmed")
                    })
                })
                console.log(response.razorpay_payment_id);
                console.log(response.razorpay_order_id);
                console.log(response.razorpay_signature)
            },
            prefill: {
                // this has to come from a form that user fills out 
                name: "",
                email: "",
                contact: ""
            },
            notes: {
                // order shipping address is basically from this
                address: "",
                pin: "",
                state: "",
            },
            theme: {
                color: "#3399cc"
            }
        };
        $scope.$watch("units", function(after, before) {
            $scope.order.amount = after * rateOfUnit;
        })
        $scope.init_rzp_pay = function() {
            // this would create the order first on the server 
            // order creation lets you have the order id and the amount of payment
            // then opens the rzp dialog 
            // completes the payment 
            // sends the payment confirmation to server again
            if ($scope.invalidity.check() == true) {
                console.log("one or more order fields are valid");
                console.warn("aborting order creation");
                console.log($scope.invalidity);
                return;
            };
            console.log($scope.order.notes);
            // $http({
            //     method: "POST",
            //     url: "http://localhost/orders",
            //     headers: {
            //         'Content-Type': "application/json",
            //     },
            //     data: JSON.stringify({
            //         "amount": $scope.order.amount,
            //         "partial_payment": false,
            //         "currency": "INR"
            //     })
            // }).then(function(response) {
            //     console.log("order has been created")
            //     $scope.order.order_id = response.data.id;
            //     var rzp1 = new Razorpay($scope.order);
            //     rzp1.on('payment.failed', function(response) {
            //         console.log(response.error.code);
            //         console.log(response.error.description);
            //         console.log(response.error.source);
            //         console.log(response.error.step);
            //         console.log(response.error.reason);
            //         console.log(response.error.metadata.order_id);
            //         console.log(response.error.metadata.payment_id);
            //     });
            //     rzp1.open();
            // }, function(data) {
            //     $scope.orderFailed = {
            //         title: "Failed",
            //         msg: data.err
            //     }
            // })

        }
    })
})()