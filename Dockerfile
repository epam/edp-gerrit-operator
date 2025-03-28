FROM alpine:3.18.9

ENV OPERATOR=/usr/local/bin/gerrit-operator \
    USER_UID=1001 \
    USER_NAME=gerrit-operator \
    HOME=/home/gerrit-operator

RUN apk add --no-cache ca-certificates==20241121-r1 \
                       openssh-client==9.3_p2-r3 \
                       openssl==3.1.8-r0 \
                       git==2.40.4-r0

# install operator binary
COPY ./dist/go-binary ${OPERATOR}

COPY build/bin /usr/local/bin
COPY build/configs /usr/local/configs

RUN chmod u+x /usr/local/bin/user_setup && \
    chmod ugo+x /usr/local/bin/entrypoint && \
    /usr/local/bin/user_setup

ENTRYPOINT ["/usr/local/bin/entrypoint"]

USER ${USER_UID}
