# This project is DEPRECATED

# RSDL
Redshift DownLoader is a small tool that allows one to stream
Redshift or PostgreSQL tables and views as TAB Separated data. 

# Docker
Use the included Dockerfile to build a docker image which can be deployed to CF directly.

```bash
$ git clone https://github.com/philips-labs/rsdl.git
$ cd rsdl
$ docker build -t rsdl .
```

# Deployment
See the below manifest.yml file as an example. 

```yaml
---
applications:
- name: rsdl
  docker:
    image: loafoe/rsdl:latest
  instances: 1
  memory: 64M
  disk_quota: 128M
  routes:
  - route: my-rsdl.eu-west.philips-healthsuite.com
  env:
    RSDL_PASSWORD: RandomPassw0rdHer3
    RSDL_SCHEMA: default_schema
  services:
  - redshift
```

## Endpoints
The following endpoints are available

| Endpoint | Description |
|----------|-------------|
| `/redshift/default_schema/:table/full.csv` | Dumps `:table` from `default_schema` |
| `/redhsift/:schema/:table/full.csv` | Dumps `:table` from given `:schema`

## Configuration

| Environment variable | Description | Default value |
|----------------------|-------------|---------|
| RSDL_PASSWORD | The password to use. Hardcoded username is `redshift` |
| RSDL_SCHEMA | The default schema to use ||
| RSDL_GZIP | Enable or disable GZIP compression | true |
 
# Usage
 
 ```shell script
curl -uredshift:RandomPassw0rdHer3 https://my-rsdl.eu-west.philips-healthsuite.com:4443/redshift/myschema/mytable/full.csv > mytable_full.csv
```
# Tips for Cloud foundry deployments

* Use port `4443` in your URL e.g. `https://my-rsdl.eu-west.philips-healthsuite.com:4443/redshift/mydb/mytable`
> Using port `4443` will allow the app to detect dropped connections. This ensures that streaming stops if the client connection is dropped. Otherwise the full request will continue to run potentially wasting resources and tying up a DB connection.

* Try with and without GZIP compression enabled
> If you have a very fast connection then compressing the data might actually make tranfers slower. When testing on a `500Mbit` fiber connection we found streaming data rates were higher with compression *disabled*.

  
# Maintainers
See [MAINTAINERS.md](MAINTAINERS.md)

# License
License is MIT

