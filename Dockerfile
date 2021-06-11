FROM golang:1.16-buster as builder

RUN apt-get update && apt-get install -y \
    libtagc0-dev upx-ucl libicu-dev

COPY . /src/euterpe
WORKDIR /src/euterpe

RUN make release
RUN mv euterpe /tmp/euterpe

FROM debian:buster

RUN apt-get update && apt-get install -y libtagc0 libicu63

COPY --from=builder /tmp/euterpe /usr/local/bin/euterpe

ENV HOME /root
WORKDIR /root
EXPOSE 9996
CMD ["euterpe"]
