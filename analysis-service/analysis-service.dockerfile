FROM alpine:latest

RUN mkdir /app

COPY analysisServiceApp /app

CMD [ "app/analysisServiceApp" ]