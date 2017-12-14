# conoha_exporter

## build

```shell
cd $GOPATH/src
git clone https://github.com/kaz/conoha_exporter.git

cd conoha_exporter

go get -u github.com/golang/dep/cmd/dep
PATH=$PATH:$GOPATH/bin dep ensure

go build
./conoha_exporter --help
```

## usage

```
Usage of ./conoha_exporter:
  -password string
    	ConoHa API user password
  -port string
    	Port number to listen on
  -region string
    	ConoHa region (default "tyo1")
  -tenant-id string
    	ConoHa tenant ID
  -username string
    	ConoHa API user name
```
