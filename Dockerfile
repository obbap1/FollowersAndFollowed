FROM golang:1.16.3-alpine AS builder
WORKDIR /app/
COPY go.mod .
COPY go.sum .
RUN go mod download
COPY . .

ARG MY_ID="" 
ARG STUB_TWEET="" 
ARG API_KEY="" 
ARG API_SECRET_KEY="" 
ARG ACCESS_TOKEN="" 
ARG ACCESS_TOKEN_SECRET="" 
ARG BEARER_TOKEN="" 

ENV MY_ID=${MY_ID}
ENV STUB_TWEET=${STUB_TWEET}
ENV API_KEY=${API_KEY}
ENV API_SECRET_KEY=${API_SECRET_KEY}
ENV ACCESS_TOKEN=${ACCESS_TOKEN}
ENV ACCESS_TOKEN_SECRET=${ACCESS_TOKEN_SECRET}
ENV BEARER_TOKEN=${BEARER_TOKEN} 

RUN go build -o executable .

FROM alpine:latest
WORKDIR /root/
RUN touch holder.data
COPY --from=builder /app/executable ./

CMD [ "./executable" ]


