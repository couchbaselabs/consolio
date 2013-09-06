FROM      ubuntu
MAINTAINER Dustin Sallings "dustin@spy.net"

ADD http://cbgb.io/cbgb.lin64.gz /usr/local/sbin/cbgb.gz

RUN gzip -d /usr/local/sbin/cbgb.gz
RUN chmod 755 /usr/local/sbin/cbgb

RUN mkdir -p /var/db/cbgb/
RUN chown nobody /var/db/cbgb/

EXPOSE 8091:8091 8092:8092 11210:11210
