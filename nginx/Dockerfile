FROM nginx:1.19.10-alpine
EXPOSE 80
# for this new app we prefer to copy the configuration in  sites-enabled
# root folder creation
RUN mkdir -p /var/www/eensyweb
COPY ./webeensy.conf /etc/nginx/conf.d/default.conf

# COPY nginx.conf /etc/nginx/nginx.conf
# COPY .htpasswd /etc/nginx/.htpasswd
# # never use ./luminapp when you want address the folder,  its bad habit to use it 
# # luminapp/ works when you want to copy all the folders recursively 
# # luminapp/* will copy only one level of folder 
# # http://trust-me-i-am-an.engineer/blog/tips-tricks/dockerfile-copying-the-entire-folder-recursively/
# COPY luminapp/ /var/www/luminapp/
