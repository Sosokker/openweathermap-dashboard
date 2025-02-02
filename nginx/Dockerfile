FROM nginx:alpine

WORKDIR /etc/nginx

COPY ./nginx.conf ./conf.d/default.conf

COPY ./web/index.html ./html/report/index.html  
COPY ./web/index.css ./html/report/index.css

EXPOSE 80

ENTRYPOINT [ "nginx" ]

CMD [ "-g", "daemon off;" ]