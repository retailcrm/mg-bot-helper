FROM golang:1.13-stretch

WORKDIR /
ADD ./bin/bot /

EXPOSE 3001

ENTRYPOINT ["/bot"]

CMD ["run"]
