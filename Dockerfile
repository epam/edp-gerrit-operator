FROM alpine:3.24

ENV OPERATOR=/usr/local/bin/gerrit-operator \
    USER_UID=1001 \
    USER_NAME=gerrit-operator \
    HOME=/home/gerrit-operator

# pkg/client/git runs the git CLI; ~=2.54 tracks the git minor alpine:3.24 ships, so bump both together.
# maintenance.auto=false: git detaches maintenance after a merge, and the operator, as PID 1, does not reap it.
RUN apk upgrade --no-cache && apk add --no-cache git~=2.54 && git config --system maintenance.auto false

# install operator binary
COPY ./dist/go-binary ${OPERATOR}

COPY build/bin /usr/local/bin
COPY build/configs /usr/local/configs

RUN chmod u+x /usr/local/bin/user_setup && \
    chmod ugo+x /usr/local/bin/entrypoint && \
    /usr/local/bin/user_setup

ENTRYPOINT ["/usr/local/bin/entrypoint"]

USER ${USER_UID}
