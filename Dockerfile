#
# Dockerfile for the prototype
#
#FROM ubuntu:latest
#LABEL author="Ziad Khalaf"
FROM alpine:latest

RUN apk add --no-cache git make musl-dev go

# Configure Go
ENV GOROOT /usr/lib/go
ENV GOPATH /go
ENV PATH /go/bin:$PATH

RUN mkdir -p ${GOPATH}/src ${GOPATH}/bin
RUN mkdir /home/api/
RUN mkdir /home/api/p3
WORKDIR /home/api

ADD . /home/api/p3
# RUN cd p3 && go mod init p3


# Install Dependencies
RUN go get -u github.com/gorilla/mux
RUN go get -u github.com/jinzhu/gorm
RUN go get -u github.com/jinzhu/gorm/dialects/postgres
RUN go get -u github.com/dgrijalva/jwt-go
RUN go get -u github.com/lib/pq
RUN go get -u github.com/joho/godotenv
RUN go get -u golang.org/x/crypto/bcrypt

#WORKDIR $GOPATH

#CMD ["make"]

RUN cd p3 && go build main.go