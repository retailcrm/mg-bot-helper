FROM golang:1.11-stretch

WORKDIR /
ADD ./bin/bot /
ADD ./templates/ /templates/
ADD ./static/ /static/
ADD ./translate/ /translate/
ADD ./migrations/ /migrations/

EXPOSE 3001

ENTRYPOINT ["/bot"]

CMD ["run"]
