# conoha_exporter

## build

```shell
cd $GOPATH/src
git clone https://github.com/traPtitech/conoha_exporter.git

cd conoha_exporter

go mod download

go build
./conoha_exporter
```

## usage

Put `conoha_exporter_config.yaml` in the same directory

```yaml
# Port to listen on
port: 3030
# Conoha region
region: tyo1
tenant_id: your-tenant-id
username: conoha-api-username
password: conoha-api-password
```
