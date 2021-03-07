FROM golang:1.16-buster as builder

RUN apt-get update && apt-get install -y \
    libtagc0-dev upx-ucl libicu-dev

COPY . /src/httpms
WORKDIR /src/httpms

RUN make release
RUN mv httpms /tmp/httpms

FROM debian:buster

RUN apt-get update && apt-get install -y libtagc0 libicu63

COPY --from=builder /tmp/httpms /usr/local/bin/httpms

ENV HOME /root
WORKDIR /root
EXPOSE 9996
CMD ["httpms"]
