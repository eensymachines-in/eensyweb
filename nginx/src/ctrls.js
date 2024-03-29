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

    }).controller("testPayCtrl", function($scope, $http) {
        // A sample controller to see if we can trigger an order id creation and hence access API on RZP
        // for this when running will keep RZP in test mode
        alert("inside testPayCtrl")
        console.log("inside testPayCtrl")
        var options = {
            "key": "rzp_test_Z4AumzgwmBpgQv", // Enter the Key ID generated from the Dashboard
            "amount": "500", // Amount is in currency subunits. Default currency is INR. Hence, 50000 refers to 50000 paise
            "currency": "INR",
            "name": "Acme Corp",
            "description": "Test Transaction",
            "image": "",
            "order_id": "order_JwJWSlUBtlMzL6", //This is a sample Order ID. Pass the `id` obtained in the response of Step 1
            "handler": function(response) {
                console.log(response.razorpay_payment_id);
                console.log(response.razorpay_order_id);
                console.log(response.razorpay_signature)
            },
            "prefill": {
                "name": "Niranjan Awati",
                "email": "kneerunjun@gmail.com",
                "contact": "8390302622"
            },
            "notes": {
                "address": "Eensymachines, Pune, 411038"
            },
            "theme": {
                "color": "#3399cc"
            }
        };
        var rzp1 = new Razorpay(options);
        rzp1.on('payment.failed', function(response) {
            alert(response.error.code);
            alert(response.error.description);
            alert(response.error.source);
            alert(response.error.step);
            alert(response.error.reason);
            alert(response.error.metadata.order_id);
            alert(response.error.metadata.payment_id);
        });
        $scope.test_pay = function() {
            rzp1.open()
        }
    })
})()