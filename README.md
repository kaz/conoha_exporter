# conoha_exporter

## build

```shell
cd $GOPATH/src
git clone https://github.com/traPtitech/conoha_exporter.git

cd conoha_exporter

go mod download

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
