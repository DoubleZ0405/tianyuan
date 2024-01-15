#######################
# STEP 1: COMPILE   ###
#######################
FROM csighub.tencentyun.com/tourism/trpc-go-compile:20200622v2 as builder

MAINTAINER quanweizhou "quanweizhou@tencent.com"

#ARG srv_name

##这里必须设置一个目录，如果是根目录就与gopath冲突，用mod=vendor好像有一些问题，是mod的一个bug
COPY . /app
WORKDIR /app

RUN wget https://git.woa.com/code/ca/raw/master/CA.zip \
&&  rm -rf CA && unzip CA.zip -d CA \
&&  mkdir -p /etc/pki/ca-trust/source/anchors/ \
&&  cp CA/* /etc/pki/ca-trust/source/anchors/ \
&&  update-ca-trust

RUN source ~/.bashrc && go env -w CGO_ENABLED=0 && go build -o main ./main.go  && chmod +x /app/main

#RUN GOOS=linux CGO_ENABLED=0 go build -o /go/bin/${srv_name} -mod=vendor ./main.go
#RUN mkdir /app
#RUN chmod +x /go/bin/${srv_name}

#######################
# STEP 2: BUILD IMAGE #
#######################

#FROM docker.oa.com:8080/library/busybox
FROM mirrors.tencent.com/iyunwei/busybox-linux-amd64
#ARG srv_name
EXPOSE 8000 8080
COPY --from=builder /app/main /app/
RUN mkdir -p /app/conf
#COPY ./conf/rcc.yaml /app/conf/


CMD ["/app/main","-conf","/app/conf/rcc.yaml"]
