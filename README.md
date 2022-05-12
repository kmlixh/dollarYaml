# dollar-yaml
a [Spring] like yaml tool

this tool implement from gopkg.in/yaml.v2 v2.4.0

You can read the Environment Variable

```yaml

mysql:
  maxIde: ${MYSQL_MAXIDE:10}
  maxOpen: ${MYSQL_MAXOPEN:10}
  user: ${MYSQL_USR:remote}
  password: ${MYSQL_PWD:remote123}
  server: ${MYSQL_SERVER:10.0.1.5:3306}
  database: ${MYSQL_DB:fochan}


```
