FROM golang:1.16-alpine3.13 as builder

RUN apk add --update taglib-dev libc-dev icu-dev upx make gcc git zlib-dev

COPY . /src/euterpe
WORKDIR /src/euterpe

RUN make release
RUN mv euterpe /tmp/euterpe
RUN /tmp/euterpe -config-gen && sed -i 's/localhost:9996/0.0.0.0:9996/' /root/.euterpe/config.json

FROM alpine:3.13

RUN apk add --update taglib icu

COPY --from=builder /tmp/euterpe /usr/local/bin/euterpe
COPY --from=builder /root/.euterpe/config.json /root/.euterpe/config.json

ENV HOME /root
WORKDIR /root
EXPOSE 9996
CMD ["euterpe"]
