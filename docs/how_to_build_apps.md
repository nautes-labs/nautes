## How to build apps

This doc describe how to build and test apps in this repo

### Build

```shell
make build
```

### Run

```shell
mkdir -p /opt/nautes/configs
touch /opt/nautes/configs/config

# Run api server
./bin/api-server -conf=../app/api-server/configs/config.yaml

# Run argo opertor
./bin/argo-operator

# Run base operator
./bin/base-operator

# Run cluster operator
./bin/cluster-operator

# Run runtime operator
./bin/runtime-operator

# Run webhook
./bin/webhook
```

### Unit test

```shell
# Install vault
wget https://releases.hashicorp.com/vault/1.10.4/vault_1.10.4_linux_amd64.zip
unzip vault_1.10.4_linux_amd64.zip
sudo mv vault /usr/local/bin/

# Run unit test
make test
```