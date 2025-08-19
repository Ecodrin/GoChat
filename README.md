# Описание
Чат с TCP серверной пересылкой сообщений, информация о пользователях хранится в базе данных MySQL, сообщения на сервере распределяются с помощью Apache Kafka.

# Запуск

### Запуск kafka
```
bin/zookeeper-server-start.sh config/zookeeper.properties
bin/kafka-server-start.sh config/server.properties
```

Создание топика
``` 
bin/kafka-topics.sh --create \
  --topic msgTopic \
  --bootstrap-server localhost:9092 \
  --partitions 1 \
  --replication-factor 1
```

### Запуск MySql
```
sudo systemctl start mysql
sudo systemctl enable mysql
```
Необходимо также в создать файл 
server/keys.go и в нем глобально указать переменную SecurityMySQLRootPassword в которой хранится пароль от MySQL

При повыторных запусках ошибка ```Error 1050 (42S01): Table 'users' already exists``` означает, что происходит попытка повторной миграции базы данных, необходимо закомментировать или удалить строчки 89-93 в файле ```server/server.go```
### ссылка на архитектуру: 
```
https://miro.com/app/board/uXjVIjIJ9VI=/?share_link_id=334440692895
```