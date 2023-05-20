FROM alpine:3.2
RUN apk add --update ca-certificates && \
    rm -rf /var/cache/apk/* /tmp/*
ADD emque /emque
WORKDIR /
ENTRYPOINT [ "/emque" ]
