FROM debian
MAINTAINER Ronmi Ren <ronmi@patrolavia.com>
RUN apt-get update \
 && apt-get install -y --no-install-recommends ssh nginx php5-fpm \
 && mkdir -p /etc/ynit \
 && ln -sf /etc/init.d/nginx /etc/init.d/ssh /etc/init.d/php5-fpm /etc/ynit/
ADD test.sh /etc/ynit/
ADD ynit /usr/local/bin/
CMD ["/usr/local/bin/ynit", "-debug"]
