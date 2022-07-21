
#### Front facing website for Eensymachines :
----

For customers visiting us on the web we need a landing website. 


#### Adding new routes to server and proxy:
------
As the website expands, we would need to addin more routes and hence pages attached to them. The question is : _How is the entire server configured so that I can plug in the new routes without affecting the old ones_ Here its worhtwhile to note that `nginx` is the reverse proxy to the original `GO` server. Hence routes would have to be added to `nginx` prior to making changes to the GOLang server 


Add to the nginx rulesets

```
server {
    listen  80;
    server_name web.eensymachines.in;
    root /var/www/eensyweb;
    access_log /var/log/nginx/eensyweb.access.log;
    location ~* /(src|templates|views|data|images)/{
        # delivering static files 
        try_files $uri $uri/ /index.html =404;
    }
    location ~* /(blogs|products)/(?<bid>[0-9a-zA-Z]*) {
        # for specific blog read 
        # here is our chance to send blog id from the client to the proxy
        # the proxy shall modify the index page for ist meta tags 
        proxy_set_header X-Forwarded-For $remote_addr;
        proxy_set_header X-Host $host;
        proxy_pass http://goproxy:8080$uri;
    }
    location ~* /(blogs|products|about|error) {
        proxy_set_header X-Forwarded-For $remote_addr;
        proxy_set_header X-Host $host;
        proxy_pass http://goproxy:8080$uri;
    }
    location / {
        proxy_set_header X-Forwarded-For $remote_addr;
        proxy_set_header X-Host $host;
        proxy_pass http://goproxy:8080;
    }
   
}

```

#### Integrating RazorPay page:
-------
After exploring the various ways to __integrate__ RazorPay I have kind of got a fair idea on how this works

There are about 3 ways we can get razor Pay integrated into eensymachines website.

1.  Access the API gateway using the keys. This would give us complete control (to/fro) over purchases 
2.  Using the button is the easiest, but buttons have no way of letting you (your application) know if the payment was complete. 
3.  Using a payment page instead of a button. Again the only way out is the webhook to know if the purchase was complete 
4.  Using invoicing - this is the best way : 
    1.  Client hits to generate the invoice, or requests us to generate invoice
    2.  RAzorpay generates a professional invoice and sends it to the client over email
    3.  Client can pay via the invoice 
    4.  Webhooks can detect the payment done and notify us