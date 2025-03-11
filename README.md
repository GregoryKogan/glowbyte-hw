# glowbyte-hw

- [Задания](#задания)
- [Установка Postgres](#установка-postgres)

## Задания

1. Установить где-то у себя Постгрес (на виртуальной машине, например)
2. На пункте 1 научиться делать бекап и рестор. Уметь отвечать на вопросы, а что там происходит.
3. Научиться отвечать на вопросы, что такое вакуум, вакуум фулл, аналайз и что будет, если их не делать?
4. Типы блокировок - какие есть, для чего нужны, где посмотреть?
5. Что такое MVCC, для чего он нужен, и как реализован в Постгресе?
6. Изучить сервисные таблицы (pg_stat_activity, например). Что там лежит? В каких случаях туда надо идти смотреть?

## Установка Postgres

Конфигурация сервисов `postgres` и `pgadmin` описана в `docker-compose.yml`.
Так же есть сервис приложения на Golang для заполнения бд тестовыми данными.

### Запуск

Только `postgres` и `pgadmin`:

```shell
docker compose up
```

С заполнение базы данных:

```shell
docker compose --profile seed up --build
```

После запуска `pgadmin` будет доступен на [localhost:5050](http://localhost:5050)

### Шаблон `.env`

```env
POSTGRES_PASSWORD=
POSTGRES_DB=
PGADMIN_DEFAULT_EMAIL=
PGADMIN_DEFAULT_PASSWORD=
```

Переименуйте и заполните файл `.env.template`.

## Backup & Restore

Документация: [postgresql.org/docs/backup](https://www.postgresql.org/docs/current/backup.html)

### SQL dump

> The idea behind this dump method is to generate a file with SQL commands that, when fed back to the server, will recreate the database in the same state as it was at the time of the dump.

#### Особенности

- Вывод в `stdout`.
- Позволяет создавать резервные копии с любого удаленного хоста, имеющего доступ к базе данных.
- Требует доступ на чтение ко всем таблицам. Полный бекап базы данных обычно требует привилегий супер-пользователя. Частичные бекапы можно делать например с помощью `-n schema` или `-t table`.
- Вывод, в отличие от других методов, совместим с более новыми версиями PostgreSQL.
- Единственный метод, подходящий для переноса баз данных между различными архитектурами (например 32-бит -> 64-бит).
- Внутренняя консистентность: представляет собой снимок базы данных в начале процесса создания дампа.
- Не блокирует большинство операций с базой данных.

#### Backup

```shell
# docker exec -t <container> pg_dumpall -c -U <role> > <filename>.sql

docker exec -t glowbyte-pg pg_dumpall -c -U postgres > dump_`date +%Y-%m-%d"_"%H_%M_%S`.sql
```

#### Restore

```shell
# cat <filename>.sql | docker exec -i <container> psql -U <role>

cat $(ls dump*.sql -rt created | head -n1) | docker exec -i glowbyte-pg psql -U postgres
```
